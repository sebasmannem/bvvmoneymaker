package main

import (
	"log"

	"github.com/sebasmannem/bvvmoneymaker/internal"
)

func main() {
	bvv, err := internal.NewBvvHandler()
	if err != nil {
		log.Fatalf("Error occurred on getting config: %e", err)
	}

	bvv.Evaluate()
	//balances, err := bvv.GetMarkets(false)
	//if err != nil {
	//	log.Fatalf("Error occurred on getting balances: %e", err)
	//}
	//for _, balance := range balances {
	//	internal.PrettyPrint(balance)
	//}
	//bvv.GetBalances()
	//bvv.GetAssets()
	//bvv.GetMarkets()
	//testWebsocket(bvv)
}
