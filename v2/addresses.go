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
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/orijtech/otils"
)

type Address struct {
	ID      string `json:"id"`
	Address string `json:"address"`

	// Name is the user defined label for the address.
	Name otils.NullableString `json:"name"`

	// Network is the name of the blockchain.
	Network otils.NullableString `json:"network"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type AddressPage struct {
	PageNumber int64      `json:"page_number"`
	Addresses  []*Address `json:"addresses"`
	Err        error
}

type AddressesResponse struct {
	PagesChan chan *AddressPage `json:"page"`
	Cancel    func() error
}

type AddressesRequest struct {
	AccountID string `json:"account_id"`

	MaxPage int64 `json:"max_page"`

	AddressesPerPage  int64  `json:"addresses_per_page"`
	StartingAddressID string `json:"starting_address_id"`
	EndingAddressID   string `json:"ending_address_id"`
	OrderBy           string `json:"order_by"`

	ThrottleDurationMs int64 `json:"throttle_duration_ms"`
}

func (alReq *AddressesRequest) Validate() error {
	if alReq == nil || strings.TrimSpace(alReq.AccountID) == "" {
		return errEmptyAccountID
	}
	return nil
}

type addressesPageWrap struct {
	Pagination *pagination `json:"pagination"`
	Addresses  []*Address  `json:"data"`
}

func (c *Client) ListAddresses(alReq *AddressesRequest) (*AddressesResponse, error) {
	if err := alReq.Validate(); err != nil {
		return nil, err
	}

	pagesChan := make(chan *AddressPage)
	pageExceeds := maxPageChecker(alReq.MaxPage)
	canceler, cancelFn := makeCanceler()

	go func() {
		defer close(pagesChan)

		var throttleDuration time.Duration
		if alReq.ThrottleDurationMs != NoThrottle && alReq.ThrottleDurationMs > 0 {
			throttleDuration = time.Duration(alReq.ThrottleDurationMs) * time.Millisecond
		}

		queryValues := make(url.Values)
		if limit := alReq.AddressesPerPage; limit > 0 {
			queryValues.Set("limit", fmt.Sprintf("%d", limit))
		}
		if startAddressID := strings.TrimSpace(alReq.StartingAddressID); startAddressID != "" {
			queryValues.Set("starting_after", startAddressID)
		}
		if endAddressID := strings.TrimSpace(alReq.EndingAddressID); endAddressID != "" {
			queryValues.Set("ending_before", endAddressID)
		}
		if orderBy := alReq.OrderBy; orderBy != "" {
			queryValues.Set("order", orderBy)
		}

		nextURI := otils.NullableString(fmt.Sprintf("/v2/accounts/%s/addresses", alReq.AccountID))
		if len(queryValues) > 0 {
			nextURI = otils.NullableString(fmt.Sprintf("%s?%s", nextURI, queryValues.Encode()))
		}

		pageNumber := int64(0)

		for {
			fullURL := fmt.Sprintf("%s%s", unversionedBaseURL, nextURI)
			page := new(AddressPage)
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
			pWrap := new(addressesPageWrap)
			if err := json.Unmarshal(blob, pWrap); err != nil {
				page.Err = err
				pagesChan <- page
				return
			}
			page.Addresses = pWrap.Addresses
			pagesChan <- page

			pageNumber += 1
			if pageExceeds(pageNumber) || len(page.Addresses) == 0 {
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

	res := &AddressesResponse{
		Cancel:    cancelFn,
		PagesChan: pagesChan,
	}

	return res, nil
}

type CreateAddressRequest struct {
	AccountID string `json:"account_id"`

	// Name is optional if you are
	// creating an address on demand.
	Name string `json:"name"`
}

func (caReq *CreateAddressRequest) Validate() error {
	if caReq == nil || strings.TrimSpace(caReq.AccountID) == "" {
		return errEmptyAccountID
	}
	return nil
}

type addressWrap struct {
	Address *Address `json:"data"`
}

func (c *Client) CreateAddress(caReq *CreateAddressRequest) (*Address, error) {
	if err := caReq.Validate(); err != nil {
		return nil, err
	}
	blob, err := json.Marshal(map[string]string{"name": caReq.Name})
	if err != nil {
		return nil, err
	}
	fullURL := fmt.Sprintf("%s/accounts/%s/addresses", baseURL, caReq.AccountID)
	req, err := http.NewRequest("POST", fullURL, bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}

	blob, _, err = c.doAuthAndReq(req)
	if err != nil {
		return nil, err
	}
	aWrap := new(addressWrap)
	if err := json.Unmarshal(blob, aWrap); err != nil {
		return nil, err
	}
	return aWrap.Address, nil
}
