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
	"time"

	"github.com/odeke-em/semalim"
	"github.com/orijtech/otils"
)

type CandleStickRequest struct {
	Product   string    `json:"product,omitempty"`
	StartTime time.Time `json:"start,omitempty"`
	EndTime   time.Time `json:"end,omitempty"`

	MaxPageNumber int64 `json:"max_page,omitempty"`

	ThrottleDurationMs int64 `json:"throttle_duration_ms"`

	GranularityInSeconds int `json:"granularity,omitempty"`
}

type actualCandleStickRequest struct {
	Product   string `json:"product,omitempty"`
	StartTime string `json:"start,omitempty"`
	EndTime   string `json:"end,omitempty"`

	GranularityInSeconds int `json:"granularity,omitempty"`
}

type CandleStickPage struct {
	StartTime time.Time `json:"start,omitempty"`
	EndTime   time.Time `json:"end,omitempty"`

	Err error `json:"err,omitempty"`

	CandleSticks []*CandleStick `json:"candlesticks,omitempty"`

	PageNumber int64 `json:"page,omitempty"`
}

var errBlankProduct = errors.New("expecting a non-blank product")

func (csr *CandleStickRequest) Validate() error {
	if csr == nil || csr.Product == "" {
		return errBlankProduct
	}
	return nil
}

var zeroTime time.Time

type CandleSticksResponse struct {
	Cancel    func() error
	PagesChan chan *CandleStickPage
}

func (c *Client) CandleSticks(ocsr *CandleStickRequest) (*CandleSticksResponse, error) {
	if err := ocsr.Validate(); err != nil {
		return nil, err
	}

	minStartTime := ocsr.StartTime
	maxEndTime := ocsr.EndTime
	maxPageNumber := ocsr.MaxPageNumber

	canPaginateTime := minStartTime.After(zeroTime) && maxEndTime.After(minStartTime)

	// The granularity calculations should calculate
	// through the pagination duration to avoid.
	durationIncrement := 5 * time.Hour
	if gd := ocsr.GranularityInSeconds; gd > 0 && gd <= 30 {
		durationIncrement = time.Duration(gd) * time.Second
	}

	if minStartTime.After(zeroTime) && canPaginateTime {
		maxPageNumber = int64(maxEndTime.Sub(minStartTime) / durationIncrement)
	}
	shouldTerminate := func(startTime, endTime time.Time, pageNumber int64) bool {
		// If startTime is not defined or endTime is not
		// defined. At least as of: Tue 12 Sep 2017 14:30:06 MDT
		// https://api.gdax.com/products/ETH-USD/candles?end=2017-09-02T16:50:20.00000Z
		// https://api.gdax.com/products/ETH-USD/candles?start=2017-09-02T15:25:00.00000Z
		// just return a single page
		if minStartTime.Equal(zeroTime) || maxEndTime.Equal(zeroTime) {
			return true
		}
		if startTime.After(maxEndTime) || endTime.After(maxEndTime) {
			return true
		}

		// Otherwise just paginate by maxPageNumber
		return maxPageNumber > 0 && pageNumber >= maxPageNumber
	}

	cancelChan, cancelFn := makeCanceler()
	cspChan := make(chan *CandleStickPage)
	go func() {
		defer close(cspChan)

		var throttleDuration time.Duration
		if ocsr.ThrottleDurationMs != NoThrottle {
			if ocsr.ThrottleDurationMs > 0 {
				throttleDuration = time.Duration(ocsr.ThrottleDurationMs) * time.Millisecond
			} else {
				throttleDuration = 350 * time.Millisecond
			}
		}

		startTime, endTime := ocsr.StartTime, ocsr.EndTime
		if canPaginateTime {
			// Now setup the startTime for increments
			endTime = startTime.Add(durationIncrement)
		}

		csr := new(actualCandleStickRequest)
		csr.Product = ocsr.Product
		pageNumber := int64(0)

		jobsChan := make(chan semalim.Job)
		go func() {
			defer close(jobsChan)

			for {
				csr.StartTime = iso8601(startTime)
				csr.EndTime = iso8601(endTime)

				ccsr := new(actualCandleStickRequest)
				*ccsr = *csr
				pageNumber += 1
				jobsChan <- &candleStickGetter{id: pageNumber, csr: ccsr, client: c}

				if shouldTerminate(startTime, endTime, pageNumber) {
					return
				}

				// Otherwise, now change the startDate
				newEndTime := endTime.Add(durationIncrement)
				startTime = endTime
				endTime = newEndTime

				select {
				case <-time.After(throttleDuration):
				case <-cancelChan:
					return
				}
			}
		}()

		resChan := semalim.Run(jobsChan, 4)
		for res := range resChan {
			val, err, pageNumber := res.Value(), res.Err(), res.Id().(int64)
			csPage := new(CandleStickPage)
			csPage.Err = err
			csPage.PageNumber = pageNumber
			if val != nil {
				csPage.CandleSticks = val.([]*CandleStick)
			}
			cspChan <- csPage
		}
	}()

	csRes := &CandleSticksResponse{
		Cancel:    cancelFn,
		PagesChan: cspChan,
	}

	return csRes, nil
}

type CandleStick struct {
	Time   float64 `json:"time,omitempty"`
	High   float64 `json:"high,omitempty"`
	Low    float64 `json:"low,omitempty"`
	Open   float64 `json:"open,omitempty"`
	Close  float64 `json:"close,omitempty"`
	Volume float64 `json:"volume,omitempty"`
}

var errInvalidCandleStickOriginalJSON = errors.New("expecting data of the form: [time, low, high, open, close, volume]")

func (cs *CandleStick) UnmarshalJSON(b []byte) error {
	var recv []float64
	if err := json.Unmarshal(b, &recv); err != nil {
		return err
	}
	// Expecting the data in the form:
	//    [time, low, high, open, close, volume]
	if len(recv) < 6 {
		return errInvalidCandleStickOriginalJSON
	}

	cs.Time = recv[0]
	cs.High = recv[1]
	cs.Low = recv[2]
	cs.Open = recv[3]
	cs.Close = recv[4]
	cs.Volume = recv[5]

	return nil
}

type candleStickGetter struct {
	id     int64
	csr    *actualCandleStickRequest
	client *Client
}

func (csg *candleStickGetter) Id() interface{} {
	return csg.id
}

func (csg *candleStickGetter) Do() (interface{}, error) {
	csr := csg.csr
	client := csg.client
	qv, err := otils.ToURLValues(csr)
	if err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf("https://api.gdax.com/products/%s/candles", csr.Product)
	if len(qv) > 0 {
		fullURL += "?" + qv.Encode()
	}
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	blob, _, err := client.doHTTPReq(req)
	if err != nil {
		return nil, err
	}
	var csticks []*CandleStick
	if err := json.Unmarshal(blob, &csticks); err != nil {
		return nil, err
	}

	return csticks, nil
}

var _ semalim.Job = (*candleStickGetter)(nil)

// iso8601 formats time into the ISO 8601 format of sample:
//   2017-09-02T15:25:00.00000Z
func iso8601(t time.Time) string {
	return t.Format("2006-01-02T15:04:05.00000Z")
}

/*
Note: Data for a CandleStick comes in the form
Expecting data back of the form
[
  [
    1504371000,
    350.74,
    350.74,
    350.74,
    350.74,
    52.50721618
  ],
  [
    1504370980,
    350.74,
    350.74,
    350.74,
    350.74,
    0.00173397
  ]
]

the goal is to transform it into
[
  {
    "time": 1504371000,
    "low": 350.74,
    "high": 350.74,
    "open": 350.74,
    "close": 350.74,
    "volume": 52.50721618
  },
  {
    "time": 1504370980,
    "low": 350.74,
    "high": 350.74,
    "open": 350.74,
    "close": 350.74,
    "volume": 0.00173397
  }
]
*/
