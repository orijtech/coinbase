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

const (
	profID1 = "prof1"
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
)

type profileWrap struct {
	Profile *coinbase.Profile `json:"data"`
}

func profileIDPath(profID string) string {
	return fmt.Sprintf("./testdata/profile-data-%s.json", profID)
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
	default:
		return makeResp("no such route", http.StatusNotFound, nil), nil
	}
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
