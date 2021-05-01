package internal

import (
	"fmt"
	"github.com/shopspring/decimal"
)

type BvvMarkets map[string]*bvvMarket

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

func (bm bvvMarket) reverse() (reverse *bvvMarket, err error) {
	if bm.Price.Equal(decimal.NewFromInt32(0)) {
		return &bvvMarket{}, fmt.Errorf("cannot create a reverse when the prise is 0")
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