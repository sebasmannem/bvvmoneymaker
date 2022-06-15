package internal

import (
	"fmt"

	"github.com/bitvavo/go-bitvavo-api"
	"github.com/shopspring/decimal"
)

type Rate struct {
	From decimal.Decimal
	To   decimal.Decimal
}

func (r *Rate) ExchangeFromTrade(trade bitvavo.Trades) (err error) {
	var dFrom decimal.Decimal
	var dPrice decimal.Decimal
	var dTo decimal.Decimal
	if dFrom, err = decimal.NewFromString(trade.Amount); err != nil {
		return fmt.Errorf("cannot convert `%s` to Decimal: %e", trade.Amount, err)
	} else if dPrice, err = decimal.NewFromString(trade.Price); err != nil {
		return fmt.Errorf("cannot convert `%s` to Decimal: %e", trade.Price, err)
	} else {
		dTo = dPrice.Mul(dFrom)
		if trade.Side == "buy" {
			r.Buy(dFrom, dTo)
		} else {
			r.Sell(dFrom, dTo)
		}
		//fmt.Printf("%d - %s (%s): %s/%s=%s (%s/%s)\n", trade.Timestamp, trade.Market, trade.Side, r.From, r.To, r.From.Div(r.To), dFrom, dTo)
	}
	return nil
}

func (r *Rate) Buy(from decimal.Decimal, to decimal.Decimal) {
	r.From = r.From.Add(from)
	r.To = r.To.Add(to)
}

func (r *Rate) Sell(from decimal.Decimal, to decimal.Decimal) {
	r.From = r.From.Sub(from)
	r.To = r.To.Sub(to)
}

func (r Rate) Average() (decimal.Decimal, error) {
	if r.From.Equals(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("cannot calculate average without regstering any exchanges")
	}
	return r.To.Div(r.From), nil
}
