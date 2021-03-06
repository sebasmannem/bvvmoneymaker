package internal

import (
	"fmt"
	"log"

	"github.com/bitvavo/go-bitvavo-api"
	"github.com/shopspring/decimal"
)

// Use this definition to make passing optionals easier.
// e.g. bitvavo.Markets(Options{ "market": "BTC-EUR" })
type bvvOptions map[string]string

type BvvHandler struct {
	connection *bitvavo.Bitvavo
	config     BvvConfig
	markets    BvvMarkets
	// internal temp list of current
	prices map[string]decimal.Decimal
	assets map[string]bitvavo.Assets
}

func NewBvvHandler() (bh *BvvHandler, err error) {
	log.Printf("BVV MoneyMaker version: %s", appVersion)
	var config BvvConfig
	if config, err = NewConfig(); err != nil {
		return bh, err
	} else {
		connection := bitvavo.Bitvavo{
			ApiKey:       config.Api.Key,
			ApiSecret:    config.Api.Secret,
			RestUrl:      "https://api.bitvavo.com/v2",
			WsUrl:        "wss://ws.bitvavo.com/v2/",
			AccessWindow: 10000,
			Debugging:    config.Api.Debug,
		}
		handler := BvvHandler{
			config:     config,
			connection: &connection,
		}
		if err = handler.GetAssets(); err != nil {
			return bh, err
		}
		return &handler, nil
	}
}

func (bh BvvHandler) Evaluate() {
	markets, err := bh.GetMarkets(false)
	if err != nil {
		log.Fatalf("Error occurred on getting markets: %e", err)
	}
	for _, market := range markets.Sorted() {
		if market.To != bh.config.Fiat {
			// This probably is a reverse market. Skipping.
			continue
		}
		if market.mah != nil {
			expectedRate, err := market.GetExpectedRate()
			if err != nil {
				log.Fatalf("Error occurred on getting GetExpectedRate for market %s: %e", market.Name(), err)
			}
			var direction string
			var percent decimal.Decimal
			hundred := decimal.NewFromInt(100)
			if expectedRate.GreaterThan(market.Price) {
				direction = "under"
				percent = hundred.Sub(market.Price.Div(expectedRate).Mul(hundred))
			} else {
				direction = "over"
				percent = hundred.Sub(expectedRate.Div(market.Price).Mul(hundred))
			}
			log.Printf("%s is %s%% %srated (expected %s vs actual %s)\n", market.Name(), percent.Round(2),
				direction, expectedRate.Round(2), market.Price)
			bw, err := market.GetBandWidth()
			if err != nil {
				log.Fatalf("Error occurred on getting GetBandWidth for market %s: %e", market.Name(), err)
			}
			log.Printf("%s bandwidth is between -%s%% and +%s%%.\n", market.Name(), bw.GetMinPercent().Round(2),
				bw.GetMaxPercent().Round(2))
		}
		log.Printf("%s: min: %s, max: %s, total: %s", market.Name(), market.Min.String(), market.Max.String(), market.Total().String())
		if market.Max.GreaterThan(decimal.Zero) && market.Max.LessThan(market.Total()) {
			err := bh.Sell(*market, market.Total().Sub(market.Max))
			if err != nil {
				log.Fatalf("Error occurred while selling %s: %e", market.Name(), err)
			}
		} else if avgRate, err := market.rate.Average(); err != nil {
			log.Printf("Could not determine average rate from market %s", market.Name())
		} else if avgRate.GreaterThan(market.Price) && !market.config.BuyUnderwater && !bh.config.BuyUnderwater {
			log.Printf("market %s is %s%% under water (%s>%s)", market.Name(),
				decimalPercent(avgRate, market.Price), avgRate, market.Price)
		} else if market.Min.GreaterThan(decimal.Zero) && market.Min.GreaterThan(market.Total()) {
			err := bh.Buy(*market, market.Min.Sub(market.Total()))
			if err != nil {
				log.Fatalf("Error occurred while buying %s: %e", market.Name(), err)
			}
		}
	}
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
		return bh.prices, err
	} else {
		for _, price := range tickerPriceResponse {
			if price.Price == "" {
				continue
			}
			prices[price.Market], err = decimal.NewFromString(price.Price)
			if err != nil {
				return bh.prices, err
			}
		}
	}
	bh.prices = prices
	return prices, err
}

func (bh *BvvHandler) GetMarkets(reset bool) (markets BvvMarkets, err error) {
	if len(bh.markets) > 0 && !reset {
		return bh.markets, nil
	}
	bh.markets = make(BvvMarkets)
	markets = make(BvvMarkets)

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
			_, err := NewBvvMarket(bh, b.Symbol, bh.config.Fiat, b.Available, b.InOrder)
			if mErr, ok := err.(MarketNotInConfigError); ok {
				if bh.config.Debug {
					log.Printf("%s.\n", mErr.Error())
				}
				continue
			}
			if err != nil {
				return bh.markets, err
			}
		}
	}
	return bh.markets, nil
}

func (bh BvvHandler) Sell(market BvvMarket, amount decimal.Decimal) (err error) {
	if market.MinimumAmount().GreaterThan(amount) {
		amount = market.MinimumAmount()
	}
	if !bh.config.ActiveMode {
		log.Printf("We should sell %s: %s\n", market.Name(), amount)
		bh.PrettyPrint(market.inverse)
		return nil
	}
	log.Printf("I am selling %s: %s\n", market.Name(), amount)
	var decimals int32
	if asset, exists := bh.assets[market.From]; !exists {
		return fmt.Errorf("unknown asset %s", market.From)
	} else {
		decimals = int32(asset.Decimals)
	}

	bh.PrettyPrint(market.inverse)
	placeOrderResponse, err := bh.connection.PlaceOrder(
		market.Name(),
		"sell",
		"market",
		bvvOptions{"amount": amount.Round(decimals).String()})
	if err != nil {
		return err
	} else {
		bh.PrettyPrint(placeOrderResponse)
	}
	return nil
}

func (bh BvvHandler) Buy(market BvvMarket, amount decimal.Decimal) (err error) {
	if market.MinimumAmount().GreaterThan(amount) {
		amount = market.MinimumAmount()
	}
	if !bh.config.ActiveMode {
		log.Printf("We should buy %s: %s\n", market.Name(), amount)
		bh.PrettyPrint(market.inverse)
		return nil
	}
	log.Printf("I am buying %s: %s\n", market.Name(), amount)
	var decimals int32
	if asset, exists := bh.assets[market.From]; !exists {
		return fmt.Errorf("unknown asset %s", market.From)
	} else {
		decimals = int32(asset.Decimals)
	}
	//log.Fatal("Not actually buying yet!!!")
	bh.PrettyPrint(market.inverse)
	placeOrderResponse, err := bh.connection.PlaceOrder(
		market.Name(),
		"buy",
		"market",
		bvvOptions{"amount": amount.Round(decimals).String()})
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
//		log.Println(marketsErr)
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

func (bh *BvvHandler) GetAssets() (err error) {
	if len(bh.assets) > 0 {
		return nil
	}
	bh.assets = make(map[string]bitvavo.Assets)
	assetsResponse, assetsErr := bh.connection.Assets(bvvOptions{})
	if assetsErr != nil {
		return assetsErr
	} else {
		for _, asset := range assetsResponse {
			bh.assets[asset.Symbol] = asset
			//bh.PrettyPrint(asset)
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

//log.Println("Book")
//bookResponse, bookErr := bitvavo.Book("BTC-EUR", bvvOptions{})
//if bookErr != nil {
// log.Println(bookErr)
//} else {
// PrettyPrint(bookResponse)
//}

// publicTradesResponse, publicTradesErr := bitvavo.PublicTrades("BTC-EUR", bvvOptions{})
// if publicTradesErr != nil {
//   log.Println(publicTradesErr)
// } else {
//   for _, trade := range publicTradesResponse {
//     PrettyPrint(trade)
//   }
// }

// candlesResponse, candlesErr := bitvavo.Candles("BTC-EUR", "1h", bvvOptions{})
// if candlesErr != nil {
//   log.Println(candlesErr)
// } else {
//   for _, candle := range candlesResponse {
//     PrettyPrint(candle)
//   }
// }

// tickerPriceResponse, tickerPriceErr := bitvavo.TickerPrice(bvvOptions{})
// if tickerPriceErr != nil {
//   log.Println(tickerPriceErr)
// } else {
//   for _, price := range tickerPriceResponse {
//     PrettyPrint(price)
//   }
// }

// tickerBookResponse, tickerBookErr := bitvavo.TickerBook(bvvOptions{})
// if tickerBookErr != nil {
//   log.Println(tickerBookErr)
// } else {
//   for _, book := range tickerBookResponse {
//     PrettyPrint(book)
//   }
// }

// ticker24hResponse, ticker24hErr := bitvavo.Ticker24h(bvvOptions{})
// if ticker24hErr != nil {
//   log.Println(ticker24hErr)
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
//   log.Println(placeOrderErr)
// } else {
//   PrettyPrint(placeOrderResponse)
// }

// placeOrderResponse, placeOrderErr := bitvavo.PlaceOrder(
//   "BTC-EUR",
//   "sell",
//   "stopLoss",
//   bvvOptions{"amount": "0.1", "triggerType": "price", "triggerReference": "lastTrade", "triggerAmount": "5000"})
// if placeOrderErr != nil {
//   log.Println(placeOrderErr)
// } else {
//   PrettyPrint(placeOrderResponse)
// }

// updateOrderResponse, updateOrderErr := bitvavo.UpdateOrder("BTC-EUR", "68c72b7a-2cf5-4516-8915-703a5d38c77e", bvvOptions{"amount": "0.4"})
// if updateOrderErr != nil {
//   log.Println(updateOrderErr)
// } else {
//   PrettyPrint(updateOrderResponse)
// }

// getOrderResponse, getOrderErr := bitvavo.GetOrder("BTC-EUR", "68c72b7a-2cf5-4516-8915-703a5d38c77e")
// if getOrderErr != nil {
//   log.Println(getOrderErr)
// } else {
//   PrettyPrint(getOrderResponse)
// }

// cancelOrderResponse, cancelOrderErr := bitvavo.CancelOrder("BTC-EUR", "68c72b7a-2cf5-4516-8915-703a5d38c77e")
// if cancelOrderErr != nil {
//   log.Println(cancelOrderErr)
// } else {
//   PrettyPrint(cancelOrderResponse)
// }

//log.Println("Orders")
//getOrdersResponse, getOrdersErr := bitvavo.GetOrders("BTC-EUR", bvvOptions{})
//if getOrdersErr != nil {
//  log.Println(getOrdersErr)
//} else {
//  for _, order := range getOrdersResponse {
//    PrettyPrint(order)
//  }
//}

// cancelOrdersResponse, cancelOrdersErr := bitvavo.CancelOrders(bvvOptions{"market": "BTC-EUR"})
// if cancelOrdersErr != nil {
//   log.Println(cancelOrdersErr)
// } else {
//   for _, order := range cancelOrdersResponse {
//     PrettyPrint(order)
//   }
// }

// ordersOpenResponse, ordersOpenErr := bitvavo.OrdersOpen(bvvOptions{"market": "BTC-EUR"})
// if ordersOpenErr != nil {
//   log.Println(ordersOpenErr)
// } else {
//   for _, order := range ordersOpenResponse {
//     PrettyPrint(order)
//   }
// }

// tradesResponse, tradesErr := bitvavo.Trades("BTC-EUR", bvvOptions{})
// if tradesErr != nil {
//   log.Println(tradesErr)
// } else {
//   for _, trade := range tradesResponse {
//     PrettyPrint(trade)
//   }
// }

// accountResponse, accountErr := bitvavo.Account()
// if accountErr != nil {
//   log.Println(accountErr)
// } else {
//   PrettyPrint(accountResponse)
// }

// depositAssetsResponse, depositAssetsErr := bitvavo.DepositAssets("BTC")
// if depositAssetsErr != nil {
//   log.Println(depositAssetsErr)
// } else {
//   PrettyPrint(depositAssetsResponse)
// }

// withdrawAssetsResponse, withdrawAssetsErr := bitvavo.WithdrawAssets("BTC", "1", "BitcoinAddress", bvvOptions{})
// if withdrawAssetsErr != nil {
//   log.Println(withdrawAssetsErr)
// } else {
//   PrettyPrint(withdrawAssetsResponse)
// }

// depositHistoryResponse, depositHistoryErr := bitvavo.DepositHistory(bvvOptions{})
// if depositHistoryErr != nil {
//   log.Println(depositHistoryErr)
// } else {
//   for _, deposit := range depositHistoryResponse {
//     PrettyPrint(deposit)
//   }
// }

// withdrawalHistoryResponse, withdrawalHistoryErr := bitvavo.WithdrawalHistory(bvvOptions{})
// if withdrawalHistoryErr != nil {
//   log.Println(withdrawalHistoryErr)
// } else {
//   for _, withdrawal := range withdrawalHistoryResponse {
//     PrettyPrint(withdrawal)
//   }
// }
