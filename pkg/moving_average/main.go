package moving_average

import "github.com/shopspring/decimal"

type MAError struct {
	error
}

type MABandwidth struct {
	Min decimal.Decimal
	Cur decimal.Decimal
	Max decimal.Decimal
}

func (bw MABandwidth) GetMinPercent() decimal.Decimal {
	hundred := decimal.NewFromInt(100)
	return hundred.Sub(bw.Min.Div(bw.Cur).Mul(hundred))
}

func (bw MABandwidth) GetMaxPercent() decimal.Decimal {
	hundred := decimal.NewFromInt(100)
	return hundred.Sub(bw.Cur.Div(bw.Max).Mul(hundred))
}
