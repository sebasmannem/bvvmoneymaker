package internal

import (
	"fmt"
	"log"
	"sort"

	"github.com/sebasmannem/bvvmoneymaker/pkg/moving_average"
	"github.com/shopspring/decimal"
)

type BvvMarkets map[string]*BvvMarket

func (bms BvvMarkets) Sorted() []*BvvMarket {
	var sortedMarkets []*BvvMarket
	for _, bm := range bms {
		sortedMarkets = append(sortedMarkets, bm)
	}
	sort.SliceStable(sortedMarkets, func(i, j int) bool {
		return sortedMarkets[i].Name() < sortedMarkets[j].Name()
	})
	return sortedMarkets
}

type MarketNotInConfigError struct {
	error
}

func newMarketNotInConfigError(marketName string) MarketNotInConfigError {
	return MarketNotInConfigError{
		fmt.Errorf("skipping market %s, not in config", marketName),
	}
}

type BvvMarket struct {
	From      string `yaml:"symbol"`
	To        string `yaml:"fiat"`
	handler   *BvvHandler
	config    bvvMarketConfig
	inverse   *BvvMarket
	Available decimal.Decimal `yaml:"available"`
	InOrder   decimal.Decimal `yaml:"inOrder"`
	Price     decimal.Decimal `yaml:"price"`
	Min       decimal.Decimal `yaml:"min"`
	Max       decimal.Decimal `yaml:"max"`
	mah       *MAHandler
	rate      Rate
}

func NewBvvMarket(bh *BvvHandler, symbol string, fiatSymbol, available string, inOrder string) (market BvvMarket,
	err error) {
	var (
		decMin decimal.Decimal
		decMax decimal.Decimal
	)
	config, found := bh.config.Markets[symbol]
	if !found {
		return market, newMarketNotInConfigError(symbol)
	}

	if decMin, err = decimal.NewFromString(config.MinLevel); err != nil {
		decMin = decimal.Zero
	} else if decMin.LessThan(decimal.Zero) {
		decMin = decimal.Zero
	}
	if decMax, err = decimal.NewFromString(config.MaxLevel); err != nil {
		decMax = decimal.Zero
	} else if decMax.LessThan(decMin) {
		decMax = decimal.Zero
	}
	decAvailable, err := decimal.NewFromString(available)
	if err != nil {
		return market, fmt.Errorf("could not convert available to Decimal %s: %e", available, err)
	}
	decInOrder, err := decimal.NewFromString(inOrder)
	if err != nil {
		return market, fmt.Errorf("could not convert inOrder to Decimal %s: %e", inOrder, err)
	}

	market = BvvMarket{
		From:      symbol,
		To:        fiatSymbol,
		handler:   bh,
		config:    config,
		Available: decAvailable,
		InOrder:   decInOrder,
	}
	if err = market.SetAvgRate(); err != nil {
		return BvvMarket{}, err
	}

	if config.MAConfig.Enabled() {
		market.mah, err = NewMAHandler(&market, config.MAConfig)
		if err != nil {
			return BvvMarket{}, nil
		}
	}
	err = market.setPrice(bh.prices)
	if err != nil {
		return BvvMarket{}, err
	}
	market.inverse, err = market.reverse()
	market.inverse.inverse = &market

	if decMin.Equal(decimal.Zero) {
		log.Printf("Disabling Min for %s\n", market.Name())
		market.inverse.Min = decimal.Zero
		market.Min = decimal.Zero
	} else {
		// Because Max and Min are in EUR, not in Crypto, we set them in inverse and calculate for market from inverse
		market.inverse.Min = decMin
		market.Min = market.inverse.exchange(decMin)
	}
	if decMax.Equal(decimal.Zero) {
		log.Printf("Disabling Max for %s\n", market.Name())
		market.inverse.Max = decimal.Zero
		market.Max = decimal.Zero
	} else {
		// Because Max and Min are in EUR, not in Crypto, we set them in inverse and calculate for market from inverse
		market.inverse.Max = decMax
		market.Max = market.inverse.exchange(decMax)
	}
	if err != nil {
		return BvvMarket{}, err
	}
	bh.markets[market.Name()] = &market
	bh.markets[market.inverse.Name()] = market.inverse
	return market, nil
}

func (bm *BvvMarket) SetAvgRate() error {
	var options = make(bvvOptions)
	if bm.config.RateWindow > 0 {
		options["limit"] = fmt.Sprintf("%d", bm.config.RateWindow)
	}
	publicTradesResponse, publicTradesErr := bm.handler.connection.Trades(bm.Name(), options)
	if publicTradesErr != nil {
		return publicTradesErr
	} else {
		// Let's sort old to new
		sort.SliceStable(publicTradesResponse, func(i, j int) bool {
			return publicTradesResponse[i].Timestamp < publicTradesResponse[j].Timestamp
		})
		for _, trade := range publicTradesResponse {
			if err := bm.rate.ExchangeFromTrade(trade); err != nil {
				return fmt.Errorf("error exchanging trade: %e", err)
			}
		}
	}
	return nil
}

func (bm BvvMarket) MinimumAmount() decimal.Decimal {
	return decimal.NewFromInt32(5).Div(bm.Price)
}

func (bm BvvMarket) reverse() (reverse *BvvMarket, err error) {
	if bm.Price.Equal(decimal.NewFromInt32(0)) {
		return &BvvMarket{}, fmt.Errorf("cannot create a reverse when the prise is 0")
	}
	reverse = &BvvMarket{
		From:      bm.To,
		To:        bm.From,
		Price:     decimal.NewFromInt32(1).Div(bm.Price),
		Available: bm.exchange(bm.Available),
		InOrder:   bm.exchange(bm.InOrder),
	}
	return reverse, nil
}

func (bm BvvMarket) exchange(amount decimal.Decimal) (balance decimal.Decimal) {
	return bm.Price.Mul(amount)
}

func (bm BvvMarket) GetExpectedRate() (total decimal.Decimal, err error) {
	if bm.mah == nil {
		return decimal.Zero, fmt.Errorf("cannot get expected rate without MAHandler")
	}
	return bm.mah.ema.GetWithOffset()
}

func (bm BvvMarket) GetBandWidth() (bw moving_average.MABandwidth, err error) {
	if bm.mah == nil {
		return moving_average.MABandwidth{}, fmt.Errorf("cannot get bandwidth without MAHandler")
	}
	return bm.mah.ema.GetBandwidth()
}

func (bm BvvMarket) Total() (total decimal.Decimal) {
	return bm.Available.Add(bm.InOrder)
}

func (bm BvvMarket) Name() (name string) {
	return fmt.Sprintf("%s-%s", bm.From, bm.To)
}

func (bm *BvvMarket) setPrice(prices map[string]decimal.Decimal) (err error) {
	var found bool
	bm.Price, found = prices[bm.Name()]
	if !found {
		return fmt.Errorf("could not find price for market %s", bm.Name())
	}
	// With this, pretty-print will also print this one
	//bm.FiatAvailable = fmt.Sprintf("%s %s", bm.FiatSymbol, bm.MarketFiatCurrency())
	return nil
}
