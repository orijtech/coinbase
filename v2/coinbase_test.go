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
				t.Errorf("#%d: unexpected errors: %#v", i, errs)
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

const (
	profID1 = "prof1"

	accountID1 = "2bbf394c-193b-5b2a-9155-3b4732659ede"
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
	myProfileRoute   = "/user"
	userProfileRoute = "/users"
	accountsRoute    = "/accounts"

	findAccountRoute = "/account-id"
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
	case findAccountRoute:
		return b.findAccountByIDRoundTrip(req)
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
	key1 = &coinbase.Credentials{APIKey: "unoKey", APISecret: "unoSecret$"}
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
	}

	mac := hmac.New(sha256.New, []byte(akey.APISecret))
	mac.Write([]byte(fmt.Sprintf("%s%s%s%s", timestamp, req.Method, req.URL.Path, body)))
	wantSignature := fmt.Sprintf("%x", mac.Sum(nil))
	if gotSignature != wantSignature {
		return makeResp("Invalid signature", http.StatusBadRequest, nil)
	}

	return nil
}
