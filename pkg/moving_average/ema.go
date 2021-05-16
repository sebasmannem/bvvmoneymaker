package moving_average

import (
	"fmt"
	"github.com/shopspring/decimal"
	"math"
)

type eMAAvgVal struct {
	sum   float64
	count uint
}

func (av *eMAAvgVal) Add(value float64) {
	av.sum += value
	av.count += 1
}

func (av *eMAAvgVal) Sub(value float64) error {
	if av.count <= 1 {
		return MAError{
			fmt.Errorf("cannot substract eMAAvgVal with count 0"),
		}
	}
	av.sum -= value
	av.count -= 1
	return nil
}

func (av eMAAvgVal) Get() (value float64, err error) {
	if av.count < 1 {
		return value, MAError{
			fmt.Errorf("cannot get eMAAvgVal with count 0"),
		}
	}
	return av.sum / float64(av.count),nil
}

// EMA is calculated with historic values only and therefore will on average be lower than the current value.
// Therefore, we also track the average offset so we can better predict the current expected EMA.

type EMAHistVals []EMAHistVal
type EMAHistVal struct {
	abs decimal.Decimal
	exp float64
}

type EMA struct {
	value   eMAAvgVal
	offset  eMAAvgVal
	history EMAHistVals
	window  uint
}

func NewEMA(window uint) (ema *EMA) {
	return &EMA{window: window}
}

func (ema *EMA) AddValue(value decimal.Decimal) {
	// Make a float
	fValue, _ := value.Float64()
	// Calculate exponential value (E of EMA)
	exp := math.Log(fValue)
	ema.value.Add(exp)
	ema.history = append(ema.history, EMAHistVal{abs: value, exp: exp})
	if ema.value.count > ema.window && ema.window > 0 {
		_ = ema.value.Sub(ema.history[0].exp)
		ema.history = ema.history[1:]
	}
	avg, _ := ema.value.Get()
	ema.offset.Add(exp - avg)
}

func (ema EMA) GetExp() (value float64, err error) {
	return ema.value.Get()
}

func (ema EMA) Get() (ret decimal.Decimal, err error) {
	exp, err := ema.value.Get()
	if err != nil {
		// Weird, so we have an offset, but we don't have a value?
		return
	}
	// This is where we can convert our exp back to a decimal
	dec := math.Exp(exp)
	return decimal.NewFromFloat(dec), nil
}

func (ema EMA) GetWithOffset() (ret decimal.Decimal, err error) {
	offset, err := ema.offset.Get()
	if err != nil {
		return
	}
	exp, err := ema.value.Get()
	if err != nil {
		// Weird, so we have an offset, but we don't have a value?
		return
	}
	// This is where we can convert our exp back to a decimal
	dec := math.Exp(exp+offset)
	return decimal.NewFromFloat(dec), nil
}

func (ema EMA) GetBandwidth() (bw MABandwidth, err error) {
	if len(ema.history) < 1 {
		return bw, MAError{
			fmt.Errorf("cannot get bandwidth without history"),
		}
	}
	bw.Min = ema.history[0].abs
	bw.Max = ema.history[0].abs
	for _, val := range ema.history {
		if bw.Min.GreaterThan(val.abs) {
			bw.Min = val.abs
		}
		if bw.Max.LessThan(val.abs) {
			bw.Max = val.abs
		}
	}
	bw.Cur, err = ema.Get()
	return bw, err
}