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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/orijtech/otils"
)

type Profile struct {
	ID        string `json:"id,omitempty"`
	Username  string `json:"username,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`

	// Name is the user's public name.
	Name otils.NullableString `json:"name,omitempty"`

	URL   otils.NullableString `json:"profile_url,omitempty"`
	Email otils.NullableString `json:"email,omitempty"`

	// Location is the location for the user's public profile.
	Location  otils.NullableString `json:"profile_location,omitempty"`
	Biography otils.NullableString `json:"profile_bio,omitempty"`

	Timezone       otils.NullableString `json:"time_zone,omitempty"`
	NativeCurrency otils.NullableString `json:"native_currency,omitempty"`
	BitcoinUnit    otils.NullableString `json:"bitcoin_unit,omitempty"`
	State          otils.NullableString `json:"state,omitempty"`
	Country        *Country             `json:"country,omitempty"`
	CreatedAt      *time.Time           `json:"created_at,omitempty"`
}

type Country struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var (
	errBlankProfileID = errors.New("expecting a non-blank userID")

	errUnimplemented = errors.New("unimplemented")
)

func (c *Client) MyProfile() (*Profile, error) {
	fullURL := fmt.Sprintf("%s/user", baseURL)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	return c.fetchProfile(req)
}

func (c *Client) FindProfileByID(profileID string) (*Profile, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil, errBlankProfileID
	}

	fullURL := fmt.Sprintf("%s/users/%s", baseURL, profileID)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	return c.fetchProfile(req)
}

type profileWrap struct {
	Profile *Profile `json:"data"`
}

func (c *Client) fetchProfile(req *http.Request) (*Profile, error) {
	slurp, _, err := c.doAuthAndReq(req)
	if err != nil {
		return nil, err
	}
	pwrap := new(profileWrap)
	if err := json.Unmarshal(slurp, pwrap); err != nil {
		return nil, err
	}
	return pwrap.Profile, nil
}
