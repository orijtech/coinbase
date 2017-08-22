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
	"strconv"
)

// Reference: https://developers.coinbase.com/api/v2#exchange-rates

type Value float64

func (v *Value) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(b, "\"")
	i64, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return err
	}
	*v = Value(i64)
	return nil
}

type ExchangeRateResponse struct {
	From  Currency           `json:"from"`
	Rates map[Currency]Value `json:"rates"`
}

type exchangeRateResponseWrap struct {
	Data *ExchangeRateResponse `json:"data"`
}

func (c *Client) ExchangeRate(from Currency) (*ExchangeRateResponse, error) {
	// Exchange Rate reference https://developers.coinbase.com/api/v2#exchange-rates
	// is unauthenticated.
	fullURL := fmt.Sprintf("%s/exchange-rates", baseURL)
	if from != "" {
		qv := make(url.Values)
		qv.Set("currency", string(from))
		fullURL = fmt.Sprintf("%s?%s", fullURL, qv.Encode())
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	blob, _, err := c.doHTTPReq(req)
	if err != nil {
		return nil, err
	}

	cwrap := new(exchangeRateResponseWrap)
	if err := json.Unmarshal(blob, cwrap); err != nil {
		return nil, err
	}
	return cwrap.Data, nil
}
