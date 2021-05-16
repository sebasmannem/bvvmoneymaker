package internal

import (
	"fmt"
	"log"

	"github.com/bitvavo/go-bitvavo-api"
)

func testWebsocket(bitvavo *bitvavo.Bitvavo) {
	websocket, errChannel := bitvavo.NewWebsocket()
	// Once close is called on the websocket, nothing will be received until bitvavo.NewWebsocket() is called again.
	defer websocket.Close()

	timeChannel := websocket.Time()
	// marketsChannel := websocket.Markets(bvvOptions{})
	// assetsChannel := websocket.Assets(bvvOptions{})

	// bookChannel := websocket.Book("BTC-EUR", bvvOptions{})
	// publicTradesChannel := websocket.PublicTrades("BTC-EUR", bvvOptions{})
	// candlesChannel := websocket.Candles("LTC-EUR", "1h", bvvOptions{})

	// tickerPriceChannel := websocket.TickerPrice(bvvOptions{})
	// tickerBookChannel := websocket.TickerBook(bvvOptions{})
	// ticker24hChannel := websocket.Ticker24h(bvvOptions{})

	// placeOrderChannel := websocket.PlaceOrder("BTC-EUR", "buy", "limit", bvvOptions{"amount": "0.1", "price": "2000"})
	// updateOrderChannel := websocket.UpdateOrder("BTC-EUR", "556314b8-f719-466f-b63d-bf429b724ad2", bvvOptions{"amount": "0.2"})
	// getOrderChannel := websocket.GetOrder("BTC-EUR", "556314b8-f719-466f-b63d-bf429b724ad2")
	// cancelOrderChannel := websocket.CancelOrder("BTC-EUR", "556314b8-f719-466f-b63d-bf429b724ad2")
	// getOrdersChannel := websocket.GetOrders("BTC-EUR", bvvOptions{})
	// cancelOrdersChannel := websocket.CancelOrders(bvvOptions{"market": "BTC-EUR"})
	// ordersOpenChannel := websocket.OrdersOpen(bvvOptions{})

	// tradesChannel := websocket.Trades("BTC-EUR", bvvOptions{})

	// accountChannel := websocket.Account()
	// balanceChannel := websocket.Balance(bvvOptions{})
	// depositAssetsChannel := websocket.DepositAssets("BTC")
	// withdrawAssetsChannel := websocket.WithdrawAssets("EUR", "50", "NL123BIM", bvvOptions{})
	// depositHistoryChannel := websocket.DepositHistory(bvvOptions{})
	// withdrawalHistoryChannel := websocket.WithdrawalHistory(bvvOptions{})

	// subscriptionTickerChannel := websocket.SubscriptionTicker("BTC-EUR")
	// subscriptionTicker24hChannel := websocket.SubscriptionTicker24h("BTC-EUR")
	// subscriptionAccountOrderChannel, subscriptionAccountFillChannel := websocket.SubscriptionAccount("BTC-EUR")
	// subscriptionCandlesChannel := websocket.SubscriptionCandles("BTC-EUR", "1h")
	// subscriptionTradesChannel := websocket.SubscriptionTrades("BTC-EUR")
	// subscriptionBookUpdateChannel := websocket.SubscriptionBookUpdate("BTC-EUR")
	// subscriptionBookChannel := websocket.SubscriptionBook("BTC-EUR", bvvOptions{})

	// Keeps program running
	for {
		select {
		case result := <-errChannel:
			fmt.Println("Error received", result)
		case result := <-timeChannel:
			err := PrettyPrint(result)
			if err != nil {
				log.Printf("Error on PrettyPrint: %e", err)
			}
			// case result := <-marketsChannel:
			//   PrettyPrint(result)
			// case result := <-assetsChannel:
			//   PrettyPrint(result)
			// case result := <-bookChannel:
			//   PrettyPrint(result)
			// case result := <-publicTradesChannel:
			//   PrettyPrint(result)
			// case result := <-candlesChannel:
			//   PrettyPrint(result)
			// case result := <-tickerPriceChannel:
			//   PrettyPrint(result)
			// case result := <-tickerBookChannel:
			//   PrettyPrint(result)
			// case result := <-ticker24hChannel:
			//   PrettyPrint(result)
			// case result := <-placeOrderChannel:
			//   PrettyPrint(result)
			// case result := <-getOrderChannel:
			//   PrettyPrint(result)
			// case result := <-updateOrderChannel:
			//   PrettyPrint(result)
			// case result := <-cancelOrderChannel:
			//   PrettyPrint(result)
			// case result := <-getOrdersChannel:
			//   PrettyPrint(result)
			// case result := <-cancelOrdersChannel:
			//   PrettyPrint(result)
			// case result := <-ordersOpenChannel:
			//   PrettyPrint(result)
			// case result := <-tradesChannel:
			//   PrettyPrint(result)
			// case result := <-accountChannel:
			//   PrettyPrint(result)
			// case result := <-balanceChannel:
			//   PrettyPrint(result)
			// case result := <-depositAssetsChannel:
			//   PrettyPrint(result)
			// case result := <-withdrawAssetsChannel:
			//   PrettyPrint(result)
			// case result := <-depositHistoryChannel:
			//   PrettyPrint(result)
			// case result := <-withdrawalHistoryChannel:
			//   PrettyPrint(result)
			// case result := <-subscriptionTickerChannel:
			//   PrettyPrint(result)
			// case result := <-subscriptionTicker24hChannel:
			//     PrettyPrint(result)
			// case result := <-subscriptionAccountOrderChannel:
			//   PrettyPrint(result)
			// case result := <-subscriptionAccountFillChannel:
			//   PrettyPrint(result)
			// case result := <-subscriptionCandlesChannel:
			//   PrettyPrint(result)
			// case result := <-subscriptionTradesChannel:
			//   PrettyPrint(result)
			// case result := <-subscriptionBookUpdateChannel:
			// PrettyPrint(result)
			// case result := <-subscriptionBookChannel:
			//   PrettyPrint(result)
		}
	}

}
