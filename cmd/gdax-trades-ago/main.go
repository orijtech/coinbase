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

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/orijtech/coinbase/v2"
)

func main() {
	var durationAgo string
	var product string
	flag.StringVar(&durationAgo, "dur-ago", "8760h", "the duration ago to go back to")
	flag.StringVar(&product, "product", "ETH-USD", "the product to retrieve trades for")
	flag.Parse()

	now := time.Now()
	pastDuration, err := time.ParseDuration(durationAgo)
	if err != nil || pastDuration <= 0 {
		pastDuration = 365 * 24 * time.Hour
	}

	client, err := coinbase.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}

	csres, err := client.CandleSticks(&coinbase.CandleStickRequest{
		Product:   product,
		StartTime: now.Add(-1 * pastDuration),
		EndTime:   now,
	})
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create("data.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	fmt.Fprintf(f, "data,timeEpoch,high,low,open,close,volume\n")
	bw := bufio.NewWriter(f)
	defer bw.Flush()

	for csPage := range csres.PagesChan {
		if csPage.Err != nil {
			log.Printf("PageNumber #%d err: %v", csPage.PageNumber, csPage.Err)
			continue
		}
		if len(csPage.CandleSticks) == 0 {
			continue
		}
		for _, cs := range csPage.CandleSticks {
			ts := int64(cs.Time)
			t := time.Unix(ts, 0)
			iso8601 := t.Format("2006-01-02T15:04:05.00000Z")
			fmt.Fprintf(bw, "%s,%d,%.4f,%.4f,%.4f,%.4f,%.4f\n", iso8601, ts, cs.High, cs.Low, cs.Open, cs.Close, cs.Volume)
		}
		bw.Flush()
		log.Printf("Flushed page: #%d", csPage.PageNumber)
	}
}
