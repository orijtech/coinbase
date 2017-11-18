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

package coinbase_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/orijtech/coinbase/v2"
)

func TestMyProfile(t *testing.T) {
	rt := &backend{route: myProfileRoute}
	tests := [...]struct {
		creds       *coinbase.Credentials
		wantErr     bool
		wantProfile *coinbase.Profile
	}{
		0: {creds: nil, wantErr: true},
		1: {creds: key1, wantProfile: profileFromFile(profID1)},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetCredentials(tt.creds)
		client.SetHTTPRoundTripper(rt)

		myProfile, err := client.MyProfile()
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d: expected a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}

		gotBytes, wantBytes := jsonify(myProfile), jsonify(tt.wantProfile)
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("#%d. got =%s\nwant=%s", i, gotBytes, wantBytes)
		}
	}
}

func TestFindProfileByID(t *testing.T) {
	rt := &backend{route: userProfileRoute}
	tests := [...]struct {
		creds       *coinbase.Credentials
		profileID   string
		wantErr     bool
		wantProfile *coinbase.Profile
	}{
		0: {creds: nil, wantErr: true},
		1: {creds: key1, wantProfile: profileFromFile(profID1), profileID: profID1},
		2: {creds: key1, profileID: "unknownProfileID", wantErr: true},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetCredentials(tt.creds)
		client.SetHTTPRoundTripper(rt)

		theirProfile, err := client.FindProfileByID(tt.profileID)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d: expected a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}

		gotBytes, wantBytes := jsonify(theirProfile), jsonify(tt.wantProfile)
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("#%d. got =%s\nwant=%s", i, gotBytes, wantBytes)
		}
	}
}

func TestFindAccountByID(t *testing.T) {
	rt := &backend{route: findAccountRoute}
	tests := [...]struct {
		creds       *coinbase.Credentials
		accountID   string
		wantErr     bool
		wantAccount *coinbase.Account
	}{
		0: {creds: nil, wantErr: true},
		1: {creds: key1, wantAccount: accountFromFileByID(accountID1), accountID: accountID1},
		2: {creds: key1, accountID: "unknownAccountID", wantErr: true},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetCredentials(tt.creds)
		client.SetHTTPRoundTripper(rt)

		account, err := client.FindAccountByID(tt.accountID)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d: expected a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}

		gotBytes, wantBytes := jsonify(account), jsonify(tt.wantAccount)
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("#%d. got =%s\nwant=%s", i, gotBytes, wantBytes)
		}
	}
}

func TestCreateAccount(t *testing.T) {
	rt := &backend{route: createAccountRoute}
	tests := [...]struct {
		creds   *coinbase.Credentials
		creq    *coinbase.CreateAccountRequest
		wantErr bool
	}{
		0: {creds: nil, wantErr: true},
		1: {creds: key1, creq: &coinbase.CreateAccountRequest{Name: "cool calm collected"}},
		2: {creds: key1, creq: &coinbase.CreateAccountRequest{Name: " "}, wantErr: true},
		3: {creds: key1, creq: &coinbase.CreateAccountRequest{Name: ""}, wantErr: true},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetCredentials(tt.creds)
		client.SetHTTPRoundTripper(rt)

		account, err := client.CreateAccount(tt.creq)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d: expected a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}

		if account == nil {
			t.Errorf("#%d: expected a non-nil error", i)
			continue
		}
		var blankAccount coinbase.Account
		if *account == blankAccount {
			t.Errorf("#%d: expected a non blank account", i)
		}
	}
}

func TestDeleteAccount(t *testing.T) {
	rt := &backend{route: deleteAccountRoute}
	tests := [...]struct {
		creds     *coinbase.Credentials
		accountID string
		wantErr   bool
	}{
		0: {creds: nil, wantErr: true},
		1: {
			creds:     key1,
			accountID: "2bbf394c-193b-5b2a-9155-3b4732659ede",
		},
		2: {creds: key1, accountID: "", wantErr: true},
		3: {creds: key1, accountID: "   ", wantErr: true},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetCredentials(tt.creds)
		client.SetHTTPRoundTripper(rt)

		err := client.DeleteAccountByID(tt.accountID)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d: expected a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}
	}
}

func TestSetAccountAsPrimary(t *testing.T) {
	rt := &backend{route: setAccountAsPrimaryRoute}
	tests := [...]struct {
		creds     *coinbase.Credentials
		accountID string
		wantErr   bool
	}{
		0: {creds: nil, wantErr: true},
		1: {
			creds:     key1,
			accountID: "2bbf394c-193b-5b2a-9155-3b4732659ede",
		},
		2: {creds: key1, accountID: "", wantErr: true},
		3: {creds: key1, accountID: "   ", wantErr: true},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetCredentials(tt.creds)
		client.SetHTTPRoundTripper(rt)

		updatedAccount, err := client.SetAccountAsPrimary(tt.accountID)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d: expected a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}

		if updatedAccount == nil {
			t.Errorf("#%d: expected a non-nil updated account", i)
			continue
		}
		var blankAccount coinbase.Account
		if *updatedAccount == blankAccount {
			t.Errorf("#%d: expected a non blank updated account", i)
		}
		if !updatedAccount.Primary {
			t.Errorf("#%d: expected at \"Primary\" to have been set", i)
		}
	}
}

func TestUpdateAccount(t *testing.T) {
	rt := &backend{route: updateAccountRoute}
	tests := [...]struct {
		creds   *coinbase.Credentials
		ureq    *coinbase.UpdateAccountRequest
		wantErr bool
	}{
		0: {creds: nil, wantErr: true},
		1: {
			creds: key1,
			ureq: &coinbase.UpdateAccountRequest{
				Name: "cool calm collected",
				ID:   "2bbf394c-193b-5b2a-9155-3b4732659ede",
			},
		},
		2: {creds: key1, ureq: &coinbase.UpdateAccountRequest{Name: " "}, wantErr: true},
		3: {creds: key1, ureq: &coinbase.UpdateAccountRequest{Name: ""}, wantErr: true},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetCredentials(tt.creds)
		client.SetHTTPRoundTripper(rt)

		updatedAccount, err := client.UpdateAccount(tt.ureq)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d: expected a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}

		if updatedAccount == nil {
			t.Errorf("#%d: expected a non-nil updated account", i)
			continue
		}
		var blankAccount coinbase.Account
		if *updatedAccount == blankAccount {
			t.Errorf("#%d: expected a non blank updated account", i)
		}
	}
}

func TestListAccounts(t *testing.T) {
	rt := &backend{route: accountsRoute}
	tests := [...]struct {
		creds   *coinbase.Credentials
		req     *coinbase.AccountsRequest
		wantErr bool
	}{
		0: {creds: nil, wantErr: true},
		1: {
			creds: key1, req: &coinbase.AccountsRequest{
				StartingAccountID: page1AccountID,
			},
		},
		2: {
			creds: key1, req: &coinbase.AccountsRequest{
				StartingAccountID: page2AccountID,
			},
		},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetCredentials(tt.creds)
		client.SetHTTPRoundTripper(rt)

		// Speed up the tests by removing throttling
		if tt.req != nil {
			tt.req.ThrottleDurationMs = coinbase.NoThrottle
		}

		res, err := client.ListAccounts(tt.req)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("#%d: unexpected error: %v", i, err)
			}
			continue
		}

		var foundAccounts []*coinbase.Account
		var errs []error
		for page := range res.PagesChan {
			if page.Err != nil {
				errs = append(errs, page.Err)
				continue
			}
			foundAccounts = append(foundAccounts, page.Accounts...)
		}

		if len(errs) > 0 {
			if !tt.wantErr {
				for ie, err := range errs {
					t.Errorf("#%d: (%d) unexpected errors: %#v", i, ie, err)
				}
			}
			continue
		}

		if len(foundAccounts) == 0 {
			if !tt.wantErr {
				t.Errorf("#%d: expecting at least one account", i)
			}
		}
	}
}

func TestCreateAddress(t *testing.T) {
	rt := &backend{route: createAddressRoute}
	tests := [...]struct {
		creds   *coinbase.Credentials
		req     *coinbase.CreateAddressRequest
		wantErr bool
	}{
		0: {creds: nil, wantErr: true},
		1: {
			creds: key1,
			// No AccountID passed in.
			req:     &coinbase.CreateAddressRequest{},
			wantErr: true,
		},
		2: {
			creds: key1, req: &coinbase.CreateAddressRequest{
				AccountID: page2AccountID,
				Name:      "Clockwise-Counterclockwise",
			},
		},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetCredentials(tt.creds)
		client.SetHTTPRoundTripper(rt)

		addr, err := client.CreateAddress(tt.req)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d: expected a non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}

		if addr == nil {
			t.Errorf("#%d: expecting a non-blank address", i)
			continue
		}

		var blankAddress coinbase.Address
		if blankAddress == *addr {
			t.Errorf("#%d: expecting a non-blank address", i)
		}
	}
}

func TestListAddresses(t *testing.T) {
	rt := &backend{route: listAddressesRoute}
	tests := [...]struct {
		creds   *coinbase.Credentials
		req     *coinbase.AddressesRequest
		wantErr bool
	}{
		0: {creds: nil, wantErr: true},
		1: {
			creds: key1,
			// No AccountID passed in.
			req:     &coinbase.AddressesRequest{},
			wantErr: true,
		},
		2: {
			creds: key1, req: &coinbase.AddressesRequest{
				AccountID: page2AccountID,
			},
		},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetCredentials(tt.creds)
		client.SetHTTPRoundTripper(rt)

		// Speed up the tests by removing throttling
		if tt.req != nil {
			tt.req.ThrottleDurationMs = coinbase.NoThrottle
		}

		res, err := client.ListAddresses(tt.req)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("#%d: unexpected error: %v", i, err)
			}
			continue
		}

		var foundAddresses []*coinbase.Address
		var errs []error
		for page := range res.PagesChan {
			if page.Err != nil {
				errs = append(errs, page.Err)
				continue
			}
			foundAddresses = append(foundAddresses, page.Addresses...)
		}

		if len(errs) > 0 {
			if !tt.wantErr {
				for ie, err := range errs {
					t.Errorf("#%d: (%d) unexpected errors: %#v", i, ie, err)
				}
			}
			continue
		}

		if len(foundAddresses) == 0 {
			if !tt.wantErr {
				t.Errorf("#%d: expecting at least one address", i)
			}
		}
	}
}

const (
	profID1 = "prof1"

	accountID1 = "2bbf394c-193b-5b2a-9155-3b4732659ede"

	addressID1 = "dd3183eb-af1d-5f5d-a90d-cbff946435ff"
)

func jsonify(v interface{}) []byte {
	blob, _ := json.MarshalIndent(v, "", "  ")
	return blob
}

type backend struct {
	route string
}

var _ http.RoundTripper = (*backend)(nil)

const (
	orderRoute       = "/orders"
	myProfileRoute   = "/user"
	userProfileRoute = "/users"
	accountsRoute    = "/accounts"

	findAccountRoute = "/account-id"

	createAccountRoute = "/create-account"
	updateAccountRoute = "/update-account"
	deleteAccountRoute = "/delete-account"

	setAccountAsPrimaryRoute = "/set-account-as-primary"

	createAddressRoute = "/create-address"
	listAddressesRoute = "/list-addresses"

	exchangeRateRoute = "/rate"
	cancelOrderRoute  = "/cancel-order"
)

type profileWrap struct {
	Profile *coinbase.Profile `json:"data"`
}

type accountWrap struct {
	Account *coinbase.Account `json:"data"`
}

func profileIDPath(profID string) string {
	return fmt.Sprintf("./testdata/profile-data-%s.json", profID)
}

func accountIDPath(accountID string) string {
	return fmt.Sprintf("./testdata/account-%s.json", accountID)
}

func accountFromFileByID(id string) *coinbase.Account {
	f, err := os.Open(accountIDPath(id))
	if err != nil {
		return nil
	}
	defer f.Close()

	slurp, err := ioutil.ReadAll(f)
	if err != nil {
		return nil
	}
	aw := new(accountWrap)
	if err := json.Unmarshal(slurp, aw); err != nil {
		return nil
	}
	return aw.Account
}

func profileFromFile(id string) *coinbase.Profile {
	f, err := os.Open(profileIDPath(id))
	if err != nil {
		return nil
	}
	defer f.Close()

	slurp, err := ioutil.ReadAll(f)
	if err != nil {
		return nil
	}
	pw := new(profileWrap)
	if err := json.Unmarshal(slurp, pw); err != nil {
		return nil
	}
	return pw.Profile
}

func (b *backend) RoundTrip(req *http.Request) (*http.Response, error) {
	switch b.route {
	case myProfileRoute:
		return b.myProfileRoundTrip(req)
	case userProfileRoute:
		return b.userProfileRoundTrip(req)
	case accountsRoute:
		return b.accountsRoundTrip(req)
	case createAddressRoute:
		return b.createAddressRoundTrip(req)
	case findAccountRoute:
		return b.findAccountByIDRoundTrip(req)
	case createAccountRoute:
		return b.createAccountRoundTrip(req)
	case updateAccountRoute:
		return b.updateAccountRoundTrip(req)
	case setAccountAsPrimaryRoute:
		return b.setAccountAsPrimaryRoundTrip(req)
	case listAddressesRoute:
		return b.listAddressesRoundTrip(req)
	case deleteAccountRoute:
		return b.deleteAccountRoundTrip(req)
	case exchangeRateRoute:
		return b.exchangeRateRoundTrip(req)
	case orderRoute:
		return b.orderRoundTrip(req)
	case cancelOrderRoute:
		return b.cancelOrderRoundTrip(req)
	default:
		return makeResp("no such route", http.StatusNotFound, nil), nil
	}
}

func accountsPagePath(pageNumber int) string {
	return fmt.Sprintf("./testdata/accounts-page-%d.json", pageNumber)
}

const (
	page1AccountID = "0"
	page2AccountID = "1"
)

func accountByIDPath(id string) string {
	return fmt.Sprintf("./testdata/account-%s.json", id)
}

func (b *backend) findAccountByIDRoundTrip(req *http.Request) (*http.Response, error) {
	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}
	splits := strings.Split(req.URL.Path, "/")
	if len(splits) < 2 {
		return makeResp("expecting a path of /accounts/<accountID>", http.StatusBadRequest, nil), nil
	}
	accountID := splits[len(splits)-1]

	fullPath := accountByIDPath(accountID)
	f, err := os.Open(fullPath)
	if err != nil {
		return makeResp(err.Error(), http.StatusNotFound, nil), nil
	}

	return makeResp("OK", http.StatusOK, f), nil
}

func (b *backend) accountsRoundTrip(req *http.Request) (*http.Response, error) {
	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}

	var pageNumber int
	query := req.URL.Query()
	switch query.Get("starting_after") {
	default:
		pageNumber = 2 // Terminal page
	case page1AccountID:
		pageNumber = 0
	case page2AccountID:
		pageNumber = 1
	}

	f, err := os.Open(accountsPagePath(pageNumber))
	if err != nil {
		return makeResp(err.Error(), http.StatusNotFound, nil), nil
	}

	return makeResp("OK", http.StatusOK, f), nil
}

func (b *backend) myProfileRoundTrip(req *http.Request) (*http.Response, error) {
	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}

	// Once authenticated, we can now send back the profile
	f, err := os.Open(profileIDPath(profID1))
	if err != nil {
		return makeResp(err.Error(), http.StatusNotFound, nil), nil
	}

	return makeResp("OK", http.StatusOK, f), nil
}

var blankOrder = new(coinbase.Order)

func (b *backend) orderRoundTrip(req *http.Request) (*http.Response, error) {
	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}
	if req.Method != "POST" {
		return makeResp(`only accepting method "POST"`, http.StatusMethodNotAllowed, nil), nil
	}
	// Also ensure that the passphrase
	// header if the client has 2FA enabled.
	if passphrase := req.Header.Get("CB-ACCESS-PASSPHRASE"); passphrase == "" {
		badResp := makeResp(`expecting header "CB-ACCESS-PASSPHRASE" to have been set`, http.StatusUnauthorized, nil)
		return badResp, nil
	}
	blob, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	oreq := new(coinbase.Order)
	if err := json.Unmarshal(blob, oreq); err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	if reflect.DeepEqual(oreq, blankOrder) {
		return makeResp("expecting a non-blank request", http.StatusBadRequest, nil), nil
	}
	if err := oreq.Validate(); err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	side := coinbase.SideBuy
	if ss := oreq.Side; ss != "" {
		side = ss
	}
	path := fmt.Sprintf("./testdata/%s-%s-%v.json", oreq.Product, side, oreq.PostOnly)
	return makeRespFromFile(path)
}

const (
	orderID1 = "order-1"
	orderID2 = "order-X"
)

var knownOrders = map[string]bool{
	orderID1: true,
	orderID2: true,
}

func (b *backend) cancelOrderRoundTrip(req *http.Request) (*http.Response, error) {
	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}
	if req.Method != "DELETE" {
		return makeResp(`only accepting method "DELETE"`, http.StatusMethodNotAllowed, nil), nil
	}
	// Also ensure that the passphrase
	// header if the client has 2FA enabled.
	if passphrase := req.Header.Get("CB-ACCESS-PASSPHRASE"); passphrase == "" {
		badResp := makeResp(`expecting header "CB-ACCESS-PASSPHRASE" to have been set`, http.StatusUnauthorized, nil)
		return badResp, nil
	}

	splits := strings.Split(req.URL.Path, "/")
	// Expecting the path to like this: "/orders/<ORDER_ID>"
	if len(splits) < 2 || splits[len(splits)-2] != "orders" {
		badResp := makeResp(`expecting path "/orders/<ORDER_ID>"`, http.StatusUnauthorized, nil)
		return badResp, nil
	}
	orderID := splits[len(splits)-1]
	if orderID == "" {
		badResp := makeResp(`expecting a non blank orderID in path "/orders/<ORDER_ID>"`, http.StatusUnauthorized, nil)
		return badResp, nil
	}
	if _, knownOrderID := knownOrders[orderID]; !knownOrderID {
		badResp := makeResp("unknown orderID", http.StatusUnauthorized, nil)
		return badResp, nil
	}
	return makeResp("200 OK", http.StatusOK, nil), nil
}

func makeRespFromFile(p string) (*http.Response, error) {
	f, err := os.Open(p)
	if err != nil {
		return makeResp(err.Error(), http.StatusInternalServerError, nil), nil
	}
	return makeResp("200 OK", http.StatusOK, f), nil
}

func (b *backend) exchangeRateRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "GET" {
		return makeResp(`only accepting method "GET"`, http.StatusMethodNotAllowed, nil), nil
	}

	qv := req.URL.Query()
	currencyStr := qv.Get("currency")
	if currencyStr == "" {
		// Let's match the gdax API which returns by
		// default USD if no currency has been specified.
		currencyStr = "BTC"
	}
	rateFilepath := fmt.Sprintf("./testdata/rates_%s.json", currencyStr)
	f, err := os.Open(rateFilepath)
	if err != nil {
		return makeResp(err.Error(), http.StatusNotFound, nil), nil
	}
	return makeResp("OK", http.StatusOK, f), nil
}

func (b *backend) deleteAccountRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "DELETE" {
		return makeResp(`only accepting method "DELETE"`, http.StatusMethodNotAllowed, nil), nil
	}

	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}

	// Expecting a URL path of the form:
	// /v2/accounts/<account_id>
	splits := strings.Split(req.URL.Path, "/")
	if len(splits) < 3 {
		return makeResp("invalid URL expecting /accounts/<account_id>", http.StatusBadRequest, nil), nil
	}
	accountID := splits[len(splits)-1]

	// Look for existence of that ID
	accountPath := accountByIDPath(accountID)
	_, err := os.Stat(accountPath)
	if err != nil {
		if os.IsNotExist(err) {
			return makeResp(`no such account found`, http.StatusNotFound, nil), nil
		} else {
			return makeResp(err.Error(), http.StatusBadRequest, nil), nil
		}
	}

	return makeResp("No Content", http.StatusNoContent, nil), nil
}

func addressesPathByAccountID(accountID string) string {
	return fmt.Sprintf("./testdata/%s-addresses.json", accountID)
}

func (b *backend) listAddressesRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "GET" {
		return makeResp(`only accepting method "GET"`, http.StatusMethodNotAllowed, nil), nil
	}

	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}

	// Expecting a URL path of the form:
	// /v2/accounts/<account_id>/addresses
	splits := strings.Split(req.URL.Path, "/")
	if len(splits) < 4 || splits[len(splits)-1] != "addresses" {
		return makeResp("invalid URL expecting /accounts/<account_id>/addresses", http.StatusBadRequest, nil), nil
	}
	accountID := splits[len(splits)-2]

	// Otherwise, retrieve and send back that requested account
	fullPath := addressesPathByAccountID(accountID)
	f, err := os.Open(fullPath)
	if err != nil {
		return makeResp(err.Error(), http.StatusNotFound, nil), nil
	}

	return makeResp("OK", http.StatusOK, f), nil

}

func (b *backend) setAccountAsPrimaryRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "POST" {
		return makeResp(`only accepting method "POST"`, http.StatusMethodNotAllowed, nil), nil
	}

	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}

	// Expecting a URL path of the form:
	// /v2/accounts/<account_id>/primary
	splits := strings.Split(req.URL.Path, "/")
	if len(splits) < 4 || splits[len(splits)-1] != "primary" {
		return makeResp("invalid URL expecting /accounts/<account_id>/primary", http.StatusBadRequest, nil), nil
	}
	accountID := splits[len(splits)-2]

	// Otherwise, retrieve and send back that requested account
	fullPath := accountByIDPath(accountID + "-as-primary")
	f, err := os.Open(fullPath)
	if err != nil {
		return makeResp(err.Error(), http.StatusNotFound, nil), nil
	}

	return makeResp("OK", http.StatusOK, f), nil
}

func (b *backend) updateAccountRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "PUT" {
		return makeResp(`only accepting method "PUT"`, http.StatusMethodNotAllowed, nil), nil
	}

	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}

	// Expecting a URL path of the form:
	// /v2/accounts/<account_id>
	splits := strings.Split(req.URL.Path, "/")
	if len(splits) < 3 {
		return makeResp("invalid URL expecting /accounts/<account_id>", http.StatusBadRequest, nil), nil
	}
	accountID := splits[len(splits)-1]

	if req.Body == nil {
		return makeResp("expecting a non-empty body", http.StatusBadRequest, nil), nil
	}

	blob, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	recv := new(coinbase.UpdateAccountRequest)
	if err := json.Unmarshal(blob, recv); err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	var blankUReq coinbase.UpdateAccountRequest
	if *recv == blankUReq {
		return makeResp("failed to parse an updateAccountRequest", http.StatusBadRequest, nil), nil
	}

	// Otherwise, retrieve and send back that requested account
	fullPath := accountByIDPath(accountID)
	f, err := os.Open(fullPath)
	if err != nil {
		return makeResp(err.Error(), http.StatusNotFound, nil), nil
	}

	return makeResp("OK", http.StatusOK, f), nil

}

func (b *backend) createAccountRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "POST" {
		return makeResp(`only accepting method "POST"`, http.StatusMethodNotAllowed, nil), nil
	}

	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}

	if req.Body == nil {
		return makeResp("expecting a non-empty body", http.StatusBadRequest, nil), nil
	}

	blob, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	recv := new(coinbase.CreateAccountRequest)
	if err := json.Unmarshal(blob, recv); err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	var blankCReq coinbase.CreateAccountRequest
	if *recv == blankCReq {
		return makeResp("failed to parse a createAccountRequest", http.StatusBadRequest, nil), nil
	}

	// Otherwise, now send back an account
	fullPath := accountByIDPath(accountID1)
	f, err := os.Open(fullPath)
	if err != nil {
		return makeResp(err.Error(), http.StatusNotFound, nil), nil
	}

	return makeResp("OK", http.StatusOK, f), nil
}

func addressPathByID(id string) string {
	return fmt.Sprintf("./testdata/address-%s.json", id)
}

func (b *backend) createAddressRoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "POST" {
		return makeResp(`only accepting method "POST"`, http.StatusMethodNotAllowed, nil), nil
	}

	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}

	if req.Body == nil {
		return makeResp("expecting a non-empty body", http.StatusBadRequest, nil), nil
	}

	blob, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	recv := new(coinbase.CreateAddressRequest)
	if err := json.Unmarshal(blob, recv); err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	var blankCReq coinbase.CreateAddressRequest
	if *recv == blankCReq {
		return makeResp("failed to parse a createAddressRequest", http.StatusBadRequest, nil), nil
	}

	// Otherwise, now send back an address
	fullPath := addressPathByID(addressID1)
	f, err := os.Open(fullPath)
	if err != nil {
		return makeResp(err.Error(), http.StatusNotFound, nil), nil
	}

	return makeResp("OK", http.StatusOK, f), nil
}

func (b *backend) userProfileRoundTrip(req *http.Request) (*http.Response, error) {
	if badAuthResp := b.badAuthCheck(req); badAuthResp != nil {
		return badAuthResp, nil
	}

	// Expecting a route of: /users/userID
	splits := strings.Split(req.URL.Path, "/")
	if len(splits) < 2 {
		return makeResp("invalid URL expecting /users/<userID>", http.StatusBadRequest, nil), nil
	}
	userID := splits[len(splits)-1]

	// Once authenticated, we can now send back the profile
	f, err := os.Open(profileIDPath(userID))
	if err != nil {
		return makeResp(err.Error(), http.StatusNotFound, nil), nil
	}

	return makeResp("OK", http.StatusOK, f), nil
}

var (
	key1 = &coinbase.Credentials{APIKey: "unoKey", APISecret: "unoSecret$", Passphrase: "^Foo$Bar<"}
)

var keyToAccessKey = map[string]*coinbase.Credentials{
	key1.APIKey: key1,
}

func makeResp(status string, code int, body io.ReadCloser) *http.Response {
	return &http.Response{
		Status:     status,
		StatusCode: code,
		Header:     make(http.Header),
		Body:       body,
	}
}

func (b *backend) badAuthCheck(req *http.Request) *http.Response {
	akey, knownKey := keyToAccessKey[req.Header.Get("CB-ACCESS-KEY")]
	if !knownKey {
		return makeResp("Unauthorized API key", http.StatusUnauthorized, nil)
	}

	// Expecting headers:
	timestamp := req.Header.Get("CB-ACCESS-TIMESTAMP")
	if tsInt, err := strconv.ParseInt(timestamp, 10, 64); err != nil || tsInt <= 0 {
		return makeResp(`expecting "CB-ACCESS-TIMESTAMP" time as an integer since unix epoch`, http.StatusBadRequest, nil)
	}

	// Now perform the HMAC checks.
	gotSignature := req.Header.Get("CB-ACCESS-SIGN")

	var body []byte
	if req.Body != nil {
		defer req.Body.Close()
		var err error
		body, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return makeResp(fmt.Sprintf("fail to read body: %v", err.Error()), http.StatusBadRequest, nil)
		}

		// Now replace the slurped body
		prc, pwc := io.Pipe()
		go func() {
			defer pwc.Close()
			pwc.Write(body)
		}()
		req.Body = prc
	}

	mac := hmac.New(sha256.New, []byte(akey.APISecret))
	urlPath := req.URL.Path
	if q := req.URL.Query(); len(q) > 0 {
		urlPath += "?" + q.Encode()
	}
	mac.Write([]byte(fmt.Sprintf("%s%s%s%s", timestamp, req.Method, urlPath, body)))
	wantSignature := fmt.Sprintf("%x", mac.Sum(nil))
	if gotSignature != wantSignature {
		return makeResp("Invalid signature", http.StatusBadRequest, nil)
	}

	return nil
}

func TestExchangeRate(t *testing.T) {
	rt := &backend{route: exchangeRateRoute}
	tests := [...]struct {
		from    coinbase.Currency
		wantErr bool
	}{
		0: {coinbase.USD, false},
		1: {"unknown", true},
		2: {"LTC-USD", false},
		3: {"LTC-USD-BTC-ETH", false},
		4: {"", false}, // Must return the default currency
	}
	client := new(coinbase.Client)
	client.SetHTTPRoundTripper(rt)

	for i, tt := range tests {
		resp, err := client.ExchangeRate(tt.from)
		if tt.wantErr {
			if err == nil {
				t.Errorf("#%d: want non-nil error", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected err: %v", i, err)
			continue
		}

		if resp == nil || len(resp.Rates) == 0 {
			t.Errorf("#%d: want more than rates", i)
		}
	}
}

func TestOrder(t *testing.T) {
	rt := &backend{route: orderRoute}

	tests := [...]struct {
		order   *coinbase.Order
		creds   *coinbase.Credentials
		wantErr string
	}{
		0: {nil, key1, "non-blank product"},
		1: {&coinbase.Order{}, key1, "non-blank product"},
		2: {&coinbase.Order{Product: "BTC-USD"}, key1, "either price or size to have been set"},
		3: {
			&coinbase.Order{Product: "BTC-USD", Price: 100, Side: coinbase.SideSell}, key1, "",
		},
		4: {&coinbase.Order{Side: coinbase.SideBuy, Product: "BTC-USD", Price: 100}, nil, "Unauthorized"},
		5: {
			&coinbase.Order{Product: "Fake-Product", Side: coinbase.SideSell, Price: 100},
			key1, "no such",
		},
		6: {
			&coinbase.Order{
				Product:     "BTC-USD",
				Price:       100,
				Side:        coinbase.SideSell,
				CancelAfter: coinbase.Day,
			},
			key1, "to be GTT",
		},
		7: {
			&coinbase.Order{
				Product:     "BTC-USD",
				Size:        94.5,
				Side:        coinbase.SideBuy,
				TimeInForce: coinbase.GTT,
				CancelAfter: coinbase.Day,
			},
			key1, "",
		},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetHTTPRoundTripper(rt)
		client.SetCredentials(tt.creds)

		ores, err := client.Order(tt.order)
		if tt.wantErr != "" {
			if err == nil {
				t.Errorf("#%d: got a nil error", i)
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("#%d: got=%q\nwant substring: %q", i, err, tt.wantErr)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}

		if ores == nil {
			t.Errorf("#%d: wanted back a response", i)
		}
	}
}

func TestCancelOrder(t *testing.T) {
	rt := &backend{route: cancelOrderRoute}

	tests := []struct {
		orderID string
		wantErr string
		creds   *coinbase.Credentials
	}{
		{"", "Unauthorized", nil},
		{"", "non blank orderID", key1},
		{"foo", "Unauthorized", nil},
		{orderID1, "", key1},
	}

	for i, tt := range tests {
		client := new(coinbase.Client)
		client.SetHTTPRoundTripper(rt)
		client.SetCredentials(tt.creds)

		err := client.CancelOrder(tt.orderID)
		if tt.wantErr != "" {
			if err == nil {
				t.Errorf("#%d: want non-nil error", i)
			} else if g, w := err.Error(), tt.wantErr; !strings.Contains(g, w) {
				t.Errorf("#%d\ngot: %q\nwant substring: %q", i, g, w)
			}
			continue
		}

		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
		}
	}
}
