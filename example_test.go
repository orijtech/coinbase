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
