// Copyright 2017 orijtech. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package coinbase

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/orijtech/otils"
)

type Account struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Primary  bool   `json:"primary"`
	Type     string `json:"type"`
	Currency string `json:"currency"`

	Balance       *Balance `json:"balance"`
	NativeBalance *Balance `json:"native_balance"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type Balance struct {
	Amount otils.NullableFloat64 `json:"amount"`

	Currency string `json:"currency"`
}

func makeCanceler() (chan struct{}, func()) {
	signaler := make(chan struct{}, 1)
	var closeOnce sync.Once
	fn := func() {
		closeOnce.Do(func() { close(signaler) })
	}
	return signaler, fn
}

type AccountsRequest struct {
	MaxPage int64 `json:"max_page"`

	AccountsPerPage   int64  `json:"accounts_per_page"`
	StartingAccountID string `json:"starting_account_id"`
	EndingAccountID   string `json:"ending_account_id"`
	OrderBy           string `json:"order_by"`

	ThrottleDurationMs int64 `json:"throttle_duration_ms"`
}

const NoThrottle int64 = -1

type AccountsPage struct {
	Accounts   []*Account `json:"accounts"`
	PageNumber int64      `json:"page_number"`

	Err error `json:"error"`
}

var (
	errEmptyAccountID = errors.New("expecting a non-empty accountID")

	errBlankName = errors.New("expecting a non-blank name")
)

type pagination struct {
	EndingBefore   otils.NullableString `json:"ending_before"`
	StartingBefore otils.NullableString `json:"starting_before"`
	Limit          int64                `json:"limit"`

	PreviousURI otils.NullableString `json:"previous_uri"`
	NextURI     otils.NullableString `json:"next_uri"`
}

type accountsPageWrap struct {
	Pagination *pagination `json:"pagination"`
	Accounts   []*Account  `json:"data"`
}

type accountWrap struct {
	Account *Account `json:"data"`
}

type CreateAccountRequest struct {
	Name string `json:"name"`
}

func (creq *CreateAccountRequest) Validate() error {
	if creq == nil || strings.TrimSpace(creq.Name) == "" {
		return errBlankName
	}
	return nil
}

type UpdateAccountRequest struct {
	Name string `json:"name"`
	ID   string `json:"account_id"`
}

func (ureq *UpdateAccountRequest) Validate() error {
	if ureq == nil || strings.TrimSpace(ureq.ID) == "" {
		return errEmptyAccountID
	}
	if strings.TrimSpace(ureq.Name) == "" {
		return errBlankName
	}
	return nil
}

func (c *Client) UpdateAccount(ureq *UpdateAccountRequest) (*Account, error) {
	if err := ureq.Validate(); err != nil {
		return nil, err
	}
	fullURL := fmt.Sprintf("%s/accounts/%s", baseURL, ureq.ID)
	blob, err := json.Marshal(map[string]string{"name": ureq.Name})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("PUT", fullURL, bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}
	return c.authAndRetrieveAccount(req)
}

func (c *Client) CreateAccount(creq *CreateAccountRequest) (*Account, error) {
	if err := creq.Validate(); err != nil {
		return nil, err
	}
	blob, err := json.Marshal(creq)
	if err != nil {
		return nil, err
	}
	fullURL := fmt.Sprintf("%s/accounts", baseURL)
	req, err := http.NewRequest("POST", fullURL, bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}
	return c.authAndRetrieveAccount(req)
}

func (c *Client) SetAccountAsPrimary(accountID string) (*Account, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, errEmptyAccountID
	}
	fullURL := fmt.Sprintf("%s/accounts/%s/primary", baseURL, accountID)
	req, err := http.NewRequest("POST", fullURL, nil)
	if err != nil {
		return nil, err
	}
	return c.authAndRetrieveAccount(req)
}

func (c *Client) DeleteAccountByID(accountID string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return errEmptyAccountID
	}
	fullURL := fmt.Sprintf("%s/accounts/%s", baseURL, accountID)
	req, err := http.NewRequest("DELETE", fullURL, nil)
	if err != nil {
		return err
	}
	_, _, err = c.doAuthAndReq(req)
	return err
}

func (c *Client) FindAccountByID(accountID string) (*Account, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, errEmptyAccountID
	}
	fullURL := fmt.Sprintf("%s/accounts/%s", baseURL, accountID)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	return c.authAndRetrieveAccount(req)
}

func (c *Client) authAndRetrieveAccount(req *http.Request) (*Account, error) {
	blob, _, err := c.doAuthAndReq(req)
	if err != nil {
		return nil, err
	}
	aWrap := new(accountWrap)
	if err := json.Unmarshal(blob, aWrap); err != nil {
		return nil, err
	}
	return aWrap.Account, nil
}

func maxPageChecker(maxPage int64) func(int64) bool {
	return func(pageNumber int64) bool {
		if maxPage <= 0 {
			return false
		}
		return pageNumber > maxPage
	}
}

func (c *Client) ListAccounts(req *AccountsRequest) (*AccountsListResponse, error) {
	if req == nil {
		req = new(AccountsRequest)
	}

	pagesChan := make(chan *AccountsPage)
	pageExceeds := maxPageChecker(req.MaxPage)
	canceler, cancelFn := makeCanceler()

	go func() {
		defer close(pagesChan)

		var throttleDuration time.Duration
		if req.ThrottleDurationMs != NoThrottle && req.ThrottleDurationMs > 0 {
			throttleDuration = time.Duration(req.ThrottleDurationMs) * time.Millisecond
		}

		queryValues := make(url.Values)
		if limit := req.AccountsPerPage; limit > 0 {
			queryValues.Set("limit", fmt.Sprintf("%d", limit))
		}
		if startAccountID := strings.TrimSpace(req.StartingAccountID); startAccountID != "" {
			queryValues.Set("starting_after", startAccountID)
		}
		if endAccountID := strings.TrimSpace(req.EndingAccountID); endAccountID != "" {
			queryValues.Set("ending_before", endAccountID)
		}
		if orderBy := req.OrderBy; orderBy != "" {
			queryValues.Set("order", orderBy)
		}

		var nextURI otils.NullableString = "/v2/accounts"
		if len(queryValues) > 0 {
			nextURI = otils.NullableString(fmt.Sprintf("%s?%s", nextURI, queryValues.Encode()))
		}

		pageNumber := int64(0)

		for {
			fullURL := fmt.Sprintf("%s%s", unversionedBaseURL, nextURI)
			page := new(AccountsPage)
			page.PageNumber = pageNumber
			req, err := http.NewRequest("GET", fullURL, nil)
			if err != nil {
				page.Err = err
				pagesChan <- page
				return
			}
			blob, _, err := c.doAuthAndReq(req)
			if err != nil {
				page.Err = err
				pagesChan <- page
				return
			}
			pWrap := new(accountsPageWrap)
			if err := json.Unmarshal(blob, pWrap); err != nil {
				page.Err = err
				pagesChan <- page
				return
			}
			page.Accounts = pWrap.Accounts
			pagesChan <- page

			pageNumber += 1
			if pageExceeds(pageNumber) || len(page.Accounts) == 0 {
				return
			}

			nextURI = ""
			if pWrap.Pagination != nil {
				nextURI = pWrap.Pagination.NextURI
			}

			select {
			case <-time.After(throttleDuration):
			case <-canceler:
				return
			}
			if nextURI == "" {
				break
			}
		}
	}()

	res := &AccountsListResponse{
		Cancel:    cancelFn,
		PagesChan: pagesChan,
	}

	return res, nil
}

type AccountsListResponse struct {
	PagesChan chan *AccountsPage
	Cancel    func()
}
