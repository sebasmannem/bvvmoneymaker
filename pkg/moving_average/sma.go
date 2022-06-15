package moving_average

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type SMAValues []decimal.Decimal
type SimpleMovingAverages []SimpleMovingAverage

type SimpleMovingAverage struct {
	values SMAValues
	sum    decimal.Decimal
	window int
}

func NewSimpleMovingAverage(Window int) (sma SimpleMovingAverage, err error) {

	if Window == 0 || Window < -1 {
		return sma, MAError{
			fmt.Errorf("invalid window size %d", Window),
		}
	}
	sma.window = Window
	return sma, nil
}

func (sma *SimpleMovingAverage) AddValue(value decimal.Decimal) {
	var oldValue decimal.Decimal
	sma.values = append(sma.values, value)
	if len(sma.values) > sma.window {
		oldValue = sma.values[0]
		sma.values = sma.values[1:]
	}
	sma.sum = sma.sum.Add(value).Sub(oldValue)
}

func (sma *SimpleMovingAverage) GetCurrentMA() (value decimal.Decimal, err error) {
	if len(sma.values) == 0 {
		return value, MAError{
			fmt.Errorf("cannot get MA for 0 elements"),
		}
	}
	fSum, _ := sma.sum.Float64()
	return decimal.NewFromFloat(fSum / float64(len(sma.values))), nil
}
