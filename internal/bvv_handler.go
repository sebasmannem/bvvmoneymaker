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
type bvvMarkets map[string]*bvvMarket

type bvvMarket struct {
	From      string          `yaml:"symbol"`
	To        string          `yaml:"fiat"`
	inverse   *bvvMarket
	Available decimal.Decimal `yaml:"available"`
	InOrder   decimal.Decimal `yaml:"inOrder"`
	Price     decimal.Decimal `yaml:"price"`
	Min       decimal.Decimal `yaml:"min"`
	Max       decimal.Decimal `yaml:"max"`
}

type BvvHandler struct {
	connection bitvavo.Bitvavo
	config     BvvConfig
	markets    bvvMarkets
	// internal temp list of current
	prices     map[string]decimal.Decimal
}

func (bh BvvHandler) newBvvMarket(symbol string, fiatSymbol, available string, inOrder string, min string,
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
		From:         symbol,
		To:           fiatSymbol,
		Available:    decAvailable,
		InOrder:      decInOrder,
	}
	market.setPrice(bh.prices)
	market.inverse, err = market.reverse()
	market.inverse.inverse = &market

	// Because Max and Min are in EUR, not in Crypto, we set them in inverse and calculate for market from inverse
	market.inverse.Max = decMax
	market.inverse.Min = decMin
	market.Max = market.inverse.exchange(decMax)
	market.Min = market.inverse.exchange(decMin)
	if err != nil {
		return bvvMarket{}, err
	}
	bh.markets[market.Name()] = &market
	bh.markets[market.inverse.Name()] = market.inverse
	return market, nil
}

func (bm bvvMarket) reverse() (reverse *bvvMarket, err error) {
	if bm.Price.Equal(decimal.NewFromInt32(0)) {
		return &bvvMarket{}, fmt.Errorf("Cannot create a reverse when the prise is 0")
	}
	reverse = &bvvMarket{
		From:         bm.To,
		To:           bm.From,
		Price:        decimal.NewFromInt32(1).Div(bm.Price),
		Available:    bm.exchange(bm.Available),
		InOrder:      bm.exchange(bm.InOrder),
	}
	return reverse, nil
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

func (bh BvvHandler) Evaluate () {
	markets, err := bh.GetMarkets(false)
	if err != nil {
		log.Fatalf("Error occurred on getting markets: %e", err)
	}
	for _, market := range markets {
		if market.To != bh.config.Fiat {
			// This probably is a reverse market. Skipping.
			continue
		}
		if market.Max.LessThan(market.Total()) {
			bh.Sell(*market, market.Total().Sub(market.Min))
		}
	}
}

func (bm bvvMarket) exchange(amount decimal.Decimal) (balance decimal.Decimal) {
	return bm.Price.Mul(amount)
}

func (bm bvvMarket) Total() (total decimal.Decimal) {
	return bm.Available.Add(bm.InOrder)
}

func (bm bvvMarket) Name() (name string) {
	return fmt.Sprintf("%s-%s", bm.From, bm.To)
}

func (bm *bvvMarket) setPrice(prices map[string]decimal.Decimal) (err error) {
	var found bool
	bm.Price, found = prices[bm.Name()]
	if !found {
		return fmt.Errorf("could not find price for market %s", bm.Name())
	}
	// With this, pretty-print will also print this one
	//bm.FiatAvailable = fmt.Sprintf("%s %s", bm.FiatSymbol, bm.MarketFiatCurrency())
	return nil
}

func (bh BvvHandler) GetBvvTime() (time bitvavo.Time, err error) {
	return bh.connection.Time()
}

func (bh BvvHandler) GetRemainingLimit() (limit int) {
	return bh.connection.GetRemainingLimit()
}

func (bh *BvvHandler) getPrices(reset bool) (prices map[string]decimal.Decimal, err error) {
	if len(bh.prices) > 0 && !reset {
		return bh.prices, nil
	}
	bh.prices = make(map[string]decimal.Decimal)
	prices = make(map[string]decimal.Decimal)
	tickerPriceResponse, tickerPriceErr := bh.connection.TickerPrice(bvvOptions{})
	if tickerPriceErr != nil {
		fmt.Println(tickerPriceErr)
	} else {
		for _, price := range tickerPriceResponse {
			prices[price.Market], err = decimal.NewFromString(price.Price)
			if err != nil {
				return bh.prices, err
			}
		}
	}
	bh.prices = prices
	return prices, err
}

func (bh *BvvHandler) GetMarkets(reset bool) (markets bvvMarkets, err error) {
	if len(bh.markets) > 0 && !reset {
		return bh.markets, nil
	}
	bh.markets = make(bvvMarkets)
	markets = make(bvvMarkets)

	_, err = bh.getPrices(false)
	if err != nil {
		return markets, err
	}
	balanceResponse, balanceErr := bh.connection.Balance(bvvOptions{})
	if balanceErr != nil {
		return markets, err
	} else {
		for _, b := range balanceResponse {
			if b.Symbol == bh.config.Fiat {
				continue
			}
			levels, found := bh.config.Markets[b.Symbol]
			if !found {
				//fmt.Printf("Skipping symbol %s (not in config)\n", b.Symbol)
				continue
			}
			_, err := bh.newBvvMarket(b.Symbol, bh.config.Fiat, b.Available, b.InOrder, levels.MinLevel,
				levels.MaxLevel)
			if err != nil {
				return bh.markets, err
			}
		}
	}
	return bh.markets, nil
}

func (bh BvvHandler) Sell(market bvvMarket, amount decimal.Decimal) (err error) {
	if ! bh.config.ActiveMode {
		fmt.Printf("We should sell %s: %s\n", market.Name(), amount)
		return nil
	}
	fmt.Printf("I am selling %s: %s\n", market.Name(), amount)
	bh.PrettyPrint(market.inverse)
	placeOrderResponse, err := bh.connection.PlaceOrder(
	  market.Name(),
	  "sell",
	  "market",
	  bvvOptions{"amount": amount.String()})
	if err != nil {
		return err
	} else {
	  bh.PrettyPrint(placeOrderResponse)
	}
	return nil
}

//func (bh BvvHandler) GetMarkets() (err error) {
//	marketsResponse, marketsErr := bh.connection.Markets(bvvOptions{})
//	if marketsErr != nil {
//		fmt.Println(marketsErr)
//	} else {
//		for _, value := range marketsResponse {
//			err = bh.PrettyPrint(value)
//			if err != nil {
//				log.Printf("Error on PrettyPrint: %e", err)
//			}
//		}
//	}
//	return nil
//}

func (bh BvvHandler) GetAssets() (err error) {
	assetsResponse, assetsErr := bh.connection.Assets(bvvOptions{})
	if assetsErr != nil {
		fmt.Println(assetsErr)
	} else {
		for _, value := range assetsResponse {
			bh.PrettyPrint(value)
		}
	}
	return nil
}

func (bh BvvHandler) PrettyPrint(v interface{}) {
	if bh.config.Debug {
		err := PrettyPrint(v)
		if err != nil {
			log.Printf("Error on PrettyPrint: %e", err)
		}
	}
}
//fmt.Println("Book")
//bookResponse, bookErr := bitvavo.Book("BTC-EUR", bvvOptions{})
//if bookErr != nil {
// fmt.Println(bookErr)
//} else {
// PrettyPrint(bookResponse)
//}

// publicTradesResponse, publicTradesErr := bitvavo.PublicTrades("BTC-EUR", bvvOptions{})
// if publicTradesErr != nil {
//   fmt.Println(publicTradesErr)
// } else {
//   for _, trade := range publicTradesResponse {
//     PrettyPrint(trade)
//   }
// }

// candlesResponse, candlesErr := bitvavo.Candles("BTC-EUR", "1h", bvvOptions{})
// if candlesErr != nil {
//   fmt.Println(candlesErr)
// } else {
//   for _, candle := range candlesResponse {
//     PrettyPrint(candle)
//   }
// }

// tickerPriceResponse, tickerPriceErr := bitvavo.TickerPrice(bvvOptions{})
// if tickerPriceErr != nil {
//   fmt.Println(tickerPriceErr)
// } else {
//   for _, price := range tickerPriceResponse {
//     PrettyPrint(price)
//   }
// }

// tickerBookResponse, tickerBookErr := bitvavo.TickerBook(bvvOptions{})
// if tickerBookErr != nil {
//   fmt.Println(tickerBookErr)
// } else {
//   for _, book := range tickerBookResponse {
//     PrettyPrint(book)
//   }
// }

// ticker24hResponse, ticker24hErr := bitvavo.Ticker24h(bvvOptions{})
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
//   bvvOptions{"amount": "0.3", "price": "2000"})
// if placeOrderErr != nil {
//   fmt.Println(placeOrderErr)
// } else {
//   PrettyPrint(placeOrderResponse)
// }

// placeOrderResponse, placeOrderErr := bitvavo.PlaceOrder(
//   "BTC-EUR",
//   "sell",
//   "stopLoss",
//   bvvOptions{"amount": "0.1", "triggerType": "price", "triggerReference": "lastTrade", "triggerAmount": "5000"})
// if placeOrderErr != nil {
//   fmt.Println(placeOrderErr)
// } else {
//   PrettyPrint(placeOrderResponse)
// }

// updateOrderResponse, updateOrderErr := bitvavo.UpdateOrder("BTC-EUR", "68c72b7a-2cf5-4516-8915-703a5d38c77e", bvvOptions{"amount": "0.4"})
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
//getOrdersResponse, getOrdersErr := bitvavo.GetOrders("BTC-EUR", bvvOptions{})
//if getOrdersErr != nil {
//  fmt.Println(getOrdersErr)
//} else {
//  for _, order := range getOrdersResponse {
//    PrettyPrint(order)
//  }
//}

// cancelOrdersResponse, cancelOrdersErr := bitvavo.CancelOrders(bvvOptions{"market": "BTC-EUR"})
// if cancelOrdersErr != nil {
//   fmt.Println(cancelOrdersErr)
// } else {
//   for _, order := range cancelOrdersResponse {
//     PrettyPrint(order)
//   }
// }

// ordersOpenResponse, ordersOpenErr := bitvavo.OrdersOpen(bvvOptions{"market": "BTC-EUR"})
// if ordersOpenErr != nil {
//   fmt.Println(ordersOpenErr)
// } else {
//   for _, order := range ordersOpenResponse {
//     PrettyPrint(order)
//   }
// }

// tradesResponse, tradesErr := bitvavo.Trades("BTC-EUR", bvvOptions{})
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

// withdrawAssetsResponse, withdrawAssetsErr := bitvavo.WithdrawAssets("BTC", "1", "BitcoinAddress", bvvOptions{})
// if withdrawAssetsErr != nil {
//   fmt.Println(withdrawAssetsErr)
// } else {
//   PrettyPrint(withdrawAssetsResponse)
// }

// depositHistoryResponse, depositHistoryErr := bitvavo.DepositHistory(bvvOptions{})
// if depositHistoryErr != nil {
//   fmt.Println(depositHistoryErr)
// } else {
//   for _, deposit := range depositHistoryResponse {
//     PrettyPrint(deposit)
//   }
// }

// withdrawalHistoryResponse, withdrawalHistoryErr := bitvavo.WithdrawalHistory(bvvOptions{})
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

	// Once close is called on the websocket, nothing will be received until bitvavo.NewWebsocket() is called again.
	websocket.Close()
}
