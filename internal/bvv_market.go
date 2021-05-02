package internal

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type BvvMarkets map[string]*BvvMarket

type MarketNotInConfigError struct {
	error
}

func newMarketNotInConfigError(marketName string) MarketNotInConfigError {
	return MarketNotInConfigError{
		fmt.Errorf("skipping market %s, not in config", marketName),
	}
}

type BvvMarket struct {
	From       string `yaml:"symbol"`
	To         string `yaml:"fiat"`
	handler    *BvvHandler
	config     bvvMarketConfig
	inverse    *BvvMarket
	Available  decimal.Decimal `yaml:"available"`
	InOrder    decimal.Decimal `yaml:"inOrder"`
	Price      decimal.Decimal `yaml:"price"`
	Min        decimal.Decimal `yaml:"min"`
	Max        decimal.Decimal `yaml:"max"`
	ma         MovingAverage
}

func NewBvvMarket(bh *BvvHandler, symbol string, fiatSymbol, available string, inOrder string) (market BvvMarket,
	err error) {
	config, found := bh.config.Markets[symbol]
	if !found {
		return market, newMarketNotInConfigError(symbol)
	}

	decMin, err := decimal.NewFromString(config.MinLevel)
	if err != nil {
		return market, fmt.Errorf("could not convert min to Decimal %s: %e", config.MinLevel, err)
	}
	decMax, err := decimal.NewFromString(config.MaxLevel)
	if err != nil {
		return market, fmt.Errorf("could not convert max to Decimal %s: %e", config.MaxLevel, err)
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
	if config.EnableMA {
		// For now hardcoded to daily candles, for last 4 years
		end := time.Now()
		start := end.AddDate(0, 0, int(0 - bvvMALimit))
		market.ma, err = NewMovingAverage(&market, bvvMAInterval, bvvMALimit, start.Unix()*1000, end.Unix()*1000)
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

	// Because Max and Min are in EUR, not in Crypto, we set them in inverse and calculate for market from inverse
	market.inverse.Max = decMax
	market.inverse.Min = decMin
	market.Max = market.inverse.exchange(decMax)
	market.Min = market.inverse.exchange(decMin)
	if err != nil {
		return BvvMarket{}, err
	}
	bh.markets[market.Name()] = &market
	bh.markets[market.inverse.Name()] = market.inverse
	return market, nil
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
