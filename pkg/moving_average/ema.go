package moving_average

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

type EMAAvgVal struct {
	sum   float64
	count int
}

func (av *EMAAvgVal) Add(value float64) {
	av.sum += value
	av.count += 1
}

func (av *EMAAvgVal) Sub(value float64) error {
	if av.count <= 1 {
		return MAError{
			fmt.Errorf("cannot substract EMAAvgVal with count 0"),
		}
	}
	av.sum -= value
	av.count -= 1
	return nil
}

func (av EMAAvgVal) Get() (value float64, err error) {
	if av.count < 1 {
		return value, MAError{
			fmt.Errorf("cannot get EMAAvgVal with count 0"),
		}
	}
	return av.sum / float64(av.count), nil
}

// EMA is calculated with historic values only and therefore will on average be lower than the current value.
// Therefore, we also track the average offset so we can better predict the current expected EMA.

type EMAHistVals []EMAHistVal
type EMAHistVal struct {
	abs decimal.Decimal
	exp float64
}

type EMA struct {
	value   EMAAvgVal
	offset  EMAAvgVal
	history EMAHistVals
	window  int
}

func NewEMA(window int) (ema *EMA, err error) {
	if window < 1 {
		return ema, MAError{
			fmt.Errorf("invalid window size %d", window),
		}
	}
	ema = &EMA{window: window}
	return ema, nil
}

func (ema *EMA) AddValue(value decimal.Decimal) {
	// Make a float
	fValue, _ := value.Float64()
	// Calculate exponential value (E of EMA)
	exp := math.Log(fValue)
	ema.value.Add(exp)
	ema.history = append(ema.history, EMAHistVal{abs: value, exp: exp})
	if ema.value.count >= ema.window {
		_ = ema.value.Sub(ema.history[0].exp)
		ema.history = ema.history[1:]
		avg, _ := ema.value.Get()
		ema.offset.Add(exp - avg)
	}
}

func (ema EMA) Get() (value float64, err error) {
	return ema.value.Get()
}

func (ema EMA) GetWithOffset() (ret decimal.Decimal, err error) {
	offset, err := ema.offset.Get()
	if err != nil {
		return
	}
	fValue, err := ema.value.Get()
	if err != nil {
		// Weird, so we have an offset, but we don't have a value?
		return
	}
	// This is where we can convert our exp back to a decimal
	fRet := math.Exp(fValue + offset)
	ret = decimal.NewFromFloat(fRet)
	return ret, nil
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
	bw.Cur, err = ema.GetWithOffset()
	return bw, err
}
