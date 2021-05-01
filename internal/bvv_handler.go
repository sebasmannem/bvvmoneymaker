package internal

import (
	"fmt"
  "github.com/bitvavo/go-bitvavo-api"
  "log"

	"github.com/shopspring/decimal"
)

// Use this definition to make passing optionals easier.
// e.g. bitvavo.Markets(Options{ "market": "BTC-EUR" })
type bvvOptions map[string]string

type bvvMarket struct {
	Symbol          string          `yaml:"symbol"`
	NativeSymbol    string          `yaml:"native"`
	//NativeAvailable string          `yaml:"nativeAvailable"`
	Available       decimal.Decimal `yaml:"available"`
	InOrder         decimal.Decimal `yaml:"inOrder"`
	Price           decimal.Decimal `yaml:"price"`
	Min             decimal.Decimal `yaml:"min"`
	Max             decimal.Decimal `yaml:"max"`
}

type BvvHandler struct {
	connection bitvavo.Bitvavo
	config     bvvConfig
	markets    map[string]bvvMarket
	// internal temp list of current
	prices map[string]decimal.Decimal
}

func newBvvMarket(symbol string, nativeSymbol, available string, inOrder string, min string,
	max string) (market bvvMarket, err error) {
	decMin, err := decimal.NewFromString(min)
	if err != nil {
		return market, fmt.Errorf("Could not convert min to Decimal %s: %e", min, err)
	}
	decMax, err := decimal.NewFromString(max)
	if err != nil {
		return market, fmt.Errorf("Could not convert max to Decimal %s: %e", max, err)
	}
	decAvailable, err := decimal.NewFromString(available)
	if err != nil {
		return market, fmt.Errorf("Could not convert available to Decimal %s: %e", available, err)
	}
	decInOrder, err := decimal.NewFromString(inOrder)
	if err != nil {
		return market, fmt.Errorf("Could not convert inOrder to Decimal %s: %e", inOrder, err)
	}
	market = bvvMarket{
		Symbol:       symbol,
		NativeSymbol: nativeSymbol,
		Available:    decAvailable,
		InOrder:      decInOrder,
		Min:          decMin,
		Max:          decMax,
	}

	return market, nil
}
func NewBvvHandler() (bvv BvvHandler, err error) {
	bvv.config, err = NewConfig()

	if err != nil {
		return bvv, err
	}
	bvv.connection = bitvavo.Bitvavo{
		ApiKey:       bvv.config.Api.Key,
		ApiSecret:    bvv.config.Api.Secret,
		RestUrl:      "https://api.bitvavo.com/v2",
		WsUrl:        "wss://ws.bitvavo.com/v2/",
		AccessWindow: 10000,
		Debugging:    bvv.config.Api.Debug,
	}

	return bvv, err
}

func (bvv BvvHandler) Evaluate () {
	markets, err := bvv.GetMarkets(false)
	if err != nil {
		log.Fatalf("Error occurred on getting markets: %e", err)
	}
	for _, market := range markets {
		if market.NativeTotal().GreaterThan(market.Max) {
			fmt.Printf("We should sell %s: %s\n", market.Name(), market.NativeTotal())
		}
	}
}

func (bm bvvMarket) NativeTotal() (balance decimal.Decimal) {
	return bm.Price.Mul(bm.Available.Add(bm.InOrder))
}

func (bm bvvMarket) NativeAvailable() (balance decimal.Decimal) {
	return bm.Price.Mul(bm.Available)
}

func (bm bvvMarket) NativeOrder() (balance decimal.Decimal) {
	return bm.Price.Mul(bm.InOrder)
}

func (bm bvvMarket) Name() (name string) {
	return fmt.Sprintf("%s-%s", bm.Symbol, bm.NativeSymbol)
}

func (bm *bvvMarket) SetPrice(prices map[string]decimal.Decimal) (err error) {
	var found bool
	bm.Price, found = prices[bm.Name()]
	if !found {
		return fmt.Errorf("could not find price for market %s", bm.Name())
	}
	// With this, pretty-print will also print this one
	//bm.NativeAvailable = fmt.Sprintf("%s %s", bm.NativeSymbol, bm.MarketNativeCurrency())
	return nil
}

func (bvv BvvHandler) GetBvvTime() (time bitvavo.Time, err error) {
	return bvv.connection.Time()
}

func (bvv BvvHandler) GetRemainingLimit() (limit int) {
	return bvv.connection.GetRemainingLimit()
}

func (bvv *BvvHandler) getPrices(reset bool) (prices map[string]decimal.Decimal, err error) {
	if len(bvv.prices) > 0 && !reset {
		return bvv.prices, nil
	}
	bvv.prices = make(map[string]decimal.Decimal)
	prices = make(map[string]decimal.Decimal)
	tickerPriceResponse, tickerPriceErr := bvv.connection.TickerPrice(map[string]string{})
	if tickerPriceErr != nil {
		fmt.Println(tickerPriceErr)
	} else {
		for _, price := range tickerPriceResponse {
			prices[price.Market], err = decimal.NewFromString(price.Price)
			if err != nil {
				return bvv.prices, err
			}
		}
	}
	bvv.prices = prices
	return prices, err
}

func (bvv *BvvHandler) GetMarkets(reset bool) (markets map[string]bvvMarket, err error) {
	if len(bvv.markets) > 0 && !reset {
		return bvv.markets, nil
	}
	bvv.markets = make(map[string]bvvMarket)
	markets = make(map[string]bvvMarket)

	_, err = bvv.getPrices(false)
	if err != nil {
		return markets, err
	}
	balanceResponse, balanceErr := bvv.connection.Balance(map[string]string{})
	if balanceErr != nil {
		return markets, err
	} else {
		for _, b := range balanceResponse {
			if b.Symbol == bvv.config.DefaultCurrency {
				continue
			}
			levels, found := bvv.config.Markets[b.Symbol]
			if !found {
				//fmt.Printf("Skipping symbol %s (not in config)\n", b.Symbol)
				continue
			}
			market, err := newBvvMarket(b.Symbol, bvv.config.DefaultCurrency, b.Available, b.InOrder, levels.MinLevel,
				levels.MaxLevel)
			if err != nil {
				return bvv.markets, err
			}
			market.SetPrice(bvv.prices)
			markets[b.Symbol] = market
		}
	}
	bvv.markets = markets
	return markets, nil
}

func (bvv BvvHandler) Sell() {
	placeOrderResponse, placeOrderErr := bvv.connection.PlaceOrder(
	  "BTC-EUR",
	  "sell",
	  "stopLoss",
	  map[string]string{"amount": "0.1", "triggerType": "price", "triggerReference": "lastTrade", "triggerAmount": "5000"})
	if placeOrderErr != nil {
	  fmt.Println(placeOrderErr)
	} else {
	  PrettyPrint(placeOrderResponse)
	}
}

//func (bvv BvvHandler) GetMarkets() (err error) {
//	marketsResponse, marketsErr := bvv.connection.Markets(map[string]string{})
//	if marketsErr != nil {
//		fmt.Println(marketsErr)
//	} else {
//		for _, value := range marketsResponse {
//			err = PrettyPrint(value)
//			if err != nil {
//				log.Printf("Error on PrettyPrint: %e", err)
//			}
//		}
//	}
//	return nil
//}

func (bvv BvvHandler) GetAssets() (err error) {
	assetsResponse, assetsErr := bvv.connection.Assets(map[string]string{})
	if assetsErr != nil {
		fmt.Println(assetsErr)
	} else {
		for _, value := range assetsResponse {
			err = PrettyPrint(value)
			if err != nil {
				log.Printf("Error on PrettyPrint: %e", err)
			}
		}
	}
	return nil
}

//fmt.Println("Book")
//bookResponse, bookErr := bitvavo.Book("BTC-EUR", map[string]string{})
//if bookErr != nil {
// fmt.Println(bookErr)
//} else {
// PrettyPrint(bookResponse)
//}

// publicTradesResponse, publicTradesErr := bitvavo.PublicTrades("BTC-EUR", map[string]string{})
// if publicTradesErr != nil {
//   fmt.Println(publicTradesErr)
// } else {
//   for _, trade := range publicTradesResponse {
//     PrettyPrint(trade)
//   }
// }

// candlesResponse, candlesErr := bitvavo.Candles("BTC-EUR", "1h", map[string]string{})
// if candlesErr != nil {
//   fmt.Println(candlesErr)
// } else {
//   for _, candle := range candlesResponse {
//     PrettyPrint(candle)
//   }
// }

// tickerPriceResponse, tickerPriceErr := bitvavo.TickerPrice(map[string]string{})
// if tickerPriceErr != nil {
//   fmt.Println(tickerPriceErr)
// } else {
//   for _, price := range tickerPriceResponse {
//     PrettyPrint(price)
//   }
// }

// tickerBookResponse, tickerBookErr := bitvavo.TickerBook(map[string]string{})
// if tickerBookErr != nil {
//   fmt.Println(tickerBookErr)
// } else {
//   for _, book := range tickerBookResponse {
//     PrettyPrint(book)
//   }
// }

// ticker24hResponse, ticker24hErr := bitvavo.Ticker24h(map[string]string{})
// if ticker24hErr != nil {
//   fmt.Println(ticker24hErr)
// } else {
//   for _, ticker := range ticker24hResponse {
//     PrettyPrint(ticker)
//   }
// }

// placeOrderResponse, placeOrderErr := bitvavo.PlaceOrder(
//   "BTC-EUR",
//   "buy",
//   "limit",
//   map[string]string{"amount": "0.3", "price": "2000"})
// if placeOrderErr != nil {
//   fmt.Println(placeOrderErr)
// } else {
//   PrettyPrint(placeOrderResponse)
// }

// placeOrderResponse, placeOrderErr := bitvavo.PlaceOrder(
//   "BTC-EUR",
//   "sell",
//   "stopLoss",
//   map[string]string{"amount": "0.1", "triggerType": "price", "triggerReference": "lastTrade", "triggerAmount": "5000"})
// if placeOrderErr != nil {
//   fmt.Println(placeOrderErr)
// } else {
//   PrettyPrint(placeOrderResponse)
// }

// updateOrderResponse, updateOrderErr := bitvavo.UpdateOrder("BTC-EUR", "68c72b7a-2cf5-4516-8915-703a5d38c77e", map[string]string{"amount": "0.4"})
// if updateOrderErr != nil {
//   fmt.Println(updateOrderErr)
// } else {
//   PrettyPrint(updateOrderResponse)
// }

// getOrderResponse, getOrderErr := bitvavo.GetOrder("BTC-EUR", "68c72b7a-2cf5-4516-8915-703a5d38c77e")
// if getOrderErr != nil {
//   fmt.Println(getOrderErr)
// } else {
//   PrettyPrint(getOrderResponse)
// }

// cancelOrderResponse, cancelOrderErr := bitvavo.CancelOrder("BTC-EUR", "68c72b7a-2cf5-4516-8915-703a5d38c77e")
// if cancelOrderErr != nil {
//   fmt.Println(cancelOrderErr)
// } else {
//   PrettyPrint(cancelOrderResponse)
// }

//fmt.Println("Orders")
//getOrdersResponse, getOrdersErr := bitvavo.GetOrders("BTC-EUR", map[string]string{})
//if getOrdersErr != nil {
//  fmt.Println(getOrdersErr)
//} else {
//  for _, order := range getOrdersResponse {
//    PrettyPrint(order)
//  }
//}

// cancelOrdersResponse, cancelOrdersErr := bitvavo.CancelOrders(map[string]string{"market": "BTC-EUR"})
// if cancelOrdersErr != nil {
//   fmt.Println(cancelOrdersErr)
// } else {
//   for _, order := range cancelOrdersResponse {
//     PrettyPrint(order)
//   }
// }

// ordersOpenResponse, ordersOpenErr := bitvavo.OrdersOpen(map[string]string{"market": "BTC-EUR"})
// if ordersOpenErr != nil {
//   fmt.Println(ordersOpenErr)
// } else {
//   for _, order := range ordersOpenResponse {
//     PrettyPrint(order)
//   }
// }

// tradesResponse, tradesErr := bitvavo.Trades("BTC-EUR", map[string]string{})
// if tradesErr != nil {
//   fmt.Println(tradesErr)
// } else {
//   for _, trade := range tradesResponse {
//     PrettyPrint(trade)
//   }
// }

// accountResponse, accountErr := bitvavo.Account()
// if accountErr != nil {
//   fmt.Println(accountErr)
// } else {
//   PrettyPrint(accountResponse)
// }

// depositAssetsResponse, depositAssetsErr := bitvavo.DepositAssets("BTC")
// if depositAssetsErr != nil {
//   fmt.Println(depositAssetsErr)
// } else {
//   PrettyPrint(depositAssetsResponse)
// }

// withdrawAssetsResponse, withdrawAssetsErr := bitvavo.WithdrawAssets("BTC", "1", "BitcoinAddress", map[string]string{})
// if withdrawAssetsErr != nil {
//   fmt.Println(withdrawAssetsErr)
// } else {
//   PrettyPrint(withdrawAssetsResponse)
// }

// depositHistoryResponse, depositHistoryErr := bitvavo.DepositHistory(map[string]string{})
// if depositHistoryErr != nil {
//   fmt.Println(depositHistoryErr)
// } else {
//   for _, deposit := range depositHistoryResponse {
//     PrettyPrint(deposit)
//   }
// }

// withdrawalHistoryResponse, withdrawalHistoryErr := bitvavo.WithdrawalHistory(map[string]string{})
// if withdrawalHistoryErr != nil {
//   fmt.Println(withdrawalHistoryErr)
// } else {
//   for _, withdrawal := range withdrawalHistoryResponse {
//     PrettyPrint(withdrawal)
//   }
// }

func testWebsocket(bitvavo bitvavo.Bitvavo) {
	websocket, errChannel := bitvavo.NewWebsocket()

	timeChannel := websocket.Time()
	// marketsChannel := websocket.Markets(map[string]string{})
	// assetsChannel := websocket.Assets(map[string]string{})

	// bookChannel := websocket.Book("BTC-EUR", map[string]string{})
	// publicTradesChannel := websocket.PublicTrades("BTC-EUR", map[string]string{})
	// candlesChannel := websocket.Candles("LTC-EUR", "1h", map[string]string{})

	// tickerPriceChannel := websocket.TickerPrice(map[string]string{})
	// tickerBookChannel := websocket.TickerBook(map[string]string{})
	// ticker24hChannel := websocket.Ticker24h(map[string]string{})

	// placeOrderChannel := websocket.PlaceOrder("BTC-EUR", "buy", "limit", map[string]string{"amount": "0.1", "price": "2000"})
	// updateOrderChannel := websocket.UpdateOrder("BTC-EUR", "556314b8-f719-466f-b63d-bf429b724ad2", map[string]string{"amount": "0.2"})
	// getOrderChannel := websocket.GetOrder("BTC-EUR", "556314b8-f719-466f-b63d-bf429b724ad2")
	// cancelOrderChannel := websocket.CancelOrder("BTC-EUR", "556314b8-f719-466f-b63d-bf429b724ad2")
	// getOrdersChannel := websocket.GetOrders("BTC-EUR", map[string]string{})
	// cancelOrdersChannel := websocket.CancelOrders(map[string]string{"market": "BTC-EUR"})
	// ordersOpenChannel := websocket.OrdersOpen(map[string]string{})

	// tradesChannel := websocket.Trades("BTC-EUR", map[string]string{})

	// accountChannel := websocket.Account()
	// balanceChannel := websocket.Balance(map[string]string{})
	// depositAssetsChannel := websocket.DepositAssets("BTC")
	// withdrawAssetsChannel := websocket.WithdrawAssets("EUR", "50", "NL123BIM", map[string]string{})
	// depositHistoryChannel := websocket.DepositHistory(map[string]string{})
	// withdrawalHistoryChannel := websocket.WithdrawalHistory(map[string]string{})

	// subscriptionTickerChannel := websocket.SubscriptionTicker("BTC-EUR")
	// subscriptionTicker24hChannel := websocket.SubscriptionTicker24h("BTC-EUR")
	// subscriptionAccountOrderChannel, subscriptionAccountFillChannel := websocket.SubscriptionAccount("BTC-EUR")
	// subscriptionCandlesChannel := websocket.SubscriptionCandles("BTC-EUR", "1h")
	// subscriptionTradesChannel := websocket.SubscriptionTrades("BTC-EUR")
	// subscriptionBookUpdateChannel := websocket.SubscriptionBookUpdate("BTC-EUR")
	// subscriptionBookChannel := websocket.SubscriptionBook("BTC-EUR", map[string]string{})

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

	// Once close is called on the websocket, nothing will be received until bitvavo.NewWebsocket() is called again.
	websocket.Close()
}
