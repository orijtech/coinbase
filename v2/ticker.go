// Copyright 2017 orijtech, Inc. All Rights Reserved.
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
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Ticker struct {
	TradeID uint64     `json:"trade_id,omitempty"`
	Price   float64    `json:"price,omitempty"`
	Size    float64    `json:"size,omitempty"`
	Bid     float64    `json:"bid,omitempty"`
	Volume  float64    `json:"volume,omitempty"`
	Ask     float64    `json:"ask,omitempty"`
	Time    *time.Time `json:"time,omitempty"`
}

type rawTicker struct {
	TradeID uint64     `json:"trade_id"`
	Price   float64    `json:"price,string,omitempty"`
	Size    float64    `json:"size,string,omitempty"`
	Bid     float64    `json:"bid,string,omitempty"`
	Volume  float64    `json:"volume,string,omitempty"`
	Ask     float64    `json:"ask,string,omitempty"`
	Time    *time.Time `json:"time,omitempty"`
}

func (c *Client) Ticker(productID string) (*Ticker, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, errBlankProduct
	}
	fullURL := fmt.Sprintf("https://api.gdax.com/products/%s/ticker", productID)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	blob, _, err := c.doHTTPReq(req)
	if err != nil {
		return nil, err
	}
	// Using a rawTicker since the data's float
	// values are sent back as strings yet we
	// don't want them marshalled back as strings.
	rtick := new(rawTicker)
	if err := json.Unmarshal(blob, rtick); err != nil {
		return nil, err
	}
	return (*Ticker)(rtick), nil
}
