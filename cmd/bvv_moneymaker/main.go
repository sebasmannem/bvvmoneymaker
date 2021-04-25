package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/sebasmannem/bvvmoneymaker/internal"
)

// Use this definition to make passing optionals easier.
// e.g. bitvavo.Markets(Options{ "market": "BTC-EUR" })
type Options map[string]string

// Use this function to print a human readable version of the returned struct.
func PrettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return
}

func main() {
	bvv, err := internal.NewBvvHandler()
	if err != nil {
		log.Fatalf("Error occurred on getting config: %e", err)
	}

	balances, err := bvv.GetBalances(false)
	if err != nil {
		log.Fatalf("Error occurred on getting balances: %e", err)
	}
	for _, balance := range balances {
		internal.PrettyPrint(balance)
	}
	//bvv.GetBalances()
	//bvv.GetAssets()
	//bvv.GetMarkets()
	//testWebsocket(bvv)
}
