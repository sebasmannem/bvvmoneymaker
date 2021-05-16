package moving_average_test

import (
	"github.com/sebasmannem/bvvmoneymaker/pkg/moving_average"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEMAAvgVal(t *testing.T) {
	var err error
	ema := moving_average.NewEMA(2)
	_, err = ema.Get()
	assert.Error(t, err)
	val1 := decimal.NewFromInt(1)
	val2 := decimal.NewFromInt(2)
	val3 := decimal.NewFromInt(8)
	// Not that the exp average of 2 and 8 is 4
	avg := decimal.NewFromInt(4)
	ema.AddValue(val1)
	emaAvg, err := ema.Get()
	assert.NoError(t, err)
	assert.Equal(t, val1, emaAvg)
	ema.AddValue(val2)
	ema.AddValue(val3)
	emaAvg, err = ema.Get()
	assert.NoError(t, err)
	assert.Equal(t, avg, emaAvg)
}

func TestEMAAvgValNilWindow(t *testing.T) {
	var err error
	ema := moving_average.NewEMA(0)
	_, err = ema.Get()
	assert.Error(t, err)
	val1 := decimal.NewFromInt(1)
	val2 := decimal.NewFromInt(2)
	val3 := decimal.NewFromInt(4)
	// Note that the exp average of 1, 2 and 4 is 2
	avg := decimal.NewFromInt(2)
	// Note that value with offset is taken from output
	avgWithOffset, _ := decimal.NewFromString("2.828")
	ema.AddValue(val1)
	emaAvg, err := ema.Get()
	assert.NoError(t, err)
	assert.Equal(t, val1, emaAvg)
	ema.AddValue(val2)
	ema.AddValue(val3)
	emaAvg, err = ema.Get()
	assert.NoError(t, err)
	assert.Equal(t, avg, emaAvg)
	emaAvg, err = ema.GetWithOffset()
	assert.NoError(t, err)
	assert.True(t, avgWithOffset.Equal(emaAvg.Round(3)), "Avg with offset values differ: %s and %s",
		avgWithOffset, emaAvg)
}

func TestEMAAvgValBW(t *testing.T) {
	var err error
	min := decimal.NewFromInt(2)
	// Note that the exp avg of 2 and 8 is 4
	cur := decimal.NewFromInt(4)
	max := decimal.NewFromInt( 8)
	ema := moving_average.NewEMA(2)
	assert.NoError(t, err)
	ema.AddValue(min)
	ema.AddValue(max)
	bw, err := ema.GetBandwidth()
	assert.NoError(t, err)
	assert.True(t, min.Equal(bw.Min), "Min value differs: %s and %s", min, bw.Min)
	assert.True(t, cur.Equal(bw.Cur), "Cur value differs: %s and %s", cur, bw.Cur)
	assert.True(t, max.Equal(bw.Max), "Max value differs: %s and %s", max, bw.Max)
}
