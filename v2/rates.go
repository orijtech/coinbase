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
	"strings"
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

// From can either be of the form:
// * PRIMARY --> ETH
// * PRIMARY-SECONDARY --> BTC-USD
// * PRIMARY-SECONDARY1-SECONDARY2-SECONDARY3... --> LTC-USD-ETH-BTC
// Where the last two forms prune out any pairs that aren't the secondaries
func (c *Client) ExchangeRate(from Currency) (*ExchangeRateResponse, error) {
	// Exchange Rate reference https://developers.coinbase.com/api/v2#exchange-rates
	// is unauthenticated.
	// GDAX exchange rates are of the form "<PRIMARY_CURRENCY>"
	// If the user has requested "<PRIMARY_CURRENCY>-<SECONDARY_CURRENCY>"
	// That means that they are only interested in the primary-secondary rate.
	primary := string(from)
	var secondaries []string
	splits := strings.Split(primary, "-")
	if len(splits) > 0 {
		primary = splits[0]
		if len(splits) > 1 {
			secondaries = splits[1:]
		}
	}
	fullURL := fmt.Sprintf("%s/exchange-rates", baseURL)
	if from != "" {
		qv := make(url.Values)
		qv.Set("currency", primary)
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
	cwrap.Data.From = Currency(primary)
	if len(secondaries) == 0 {
		return cwrap.Data, nil
	}

	// Otherwise, they've only asked for the <primary>-<secondary1>-<secondary2>... rate
	data := cwrap.Data.Rates
	prunedRates := make(map[Currency]Value)
	for _, secondary := range secondaries {
		secCurr := Currency(secondary)
		prunedRates[secCurr] = data[secCurr]
	}
	cwrap.Data.Rates = prunedRates
	return cwrap.Data, nil
}
