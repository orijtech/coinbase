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
	"fmt"
	"log"

	"github.com/orijtech/coinbase/v2"
)

func Example_client_FindUser() {
	client, err := coinbase.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}

	profile, err := client.FindProfileByID("c50f4e4e-0f25-5a26-8901-03772a074af1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("The User profile: %+v\n", profile)
}

func Example_client_MyProfile() {
	client, err := coinbase.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}

	myProfile, err := client.MyProfile()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("My profile: %+v\n", myProfile)
}

func Example_client_ListAccounts() {
	client, err := coinbase.NewDefaultClient()
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.ListAccounts(&coinbase.AccountsRequest{
		MaxPage: 2,
	})
	if err != nil {
		log.Fatal(err)
	}

	for page := range res.PagesChan {
		if err := page.Err; err != nil {
			log.Printf("Page #%d err: %v", page.PageNumber, err)
			continue
		}

		for i, account := range page.Accounts {
			fmt.Printf("Page #%d:: (%d) Account: %#v\n", page.PageNumber, i, account)
		}
	}
}
