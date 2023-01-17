package bloom

import (
	"fmt"
	"math"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
)

func TestNewBloomConstain(t *testing.T) {
	// test zero capacity
	zeroCap := 0
	_, err := NewBloomConstrain(&zeroCap, nil)
	assert.EqualError(t, err, "capacity cannot be less than 1")

	// test zero false postive rate
	zeroAcc := float64(0)
	_, err = NewBloomConstrain(nil, &zeroAcc)
	assert.EqualError(t, err, "false positive rate must be greater than 0")

	// test capacity and accuracy incompatibility
	cap := 1000
	acc := .1
	_, err = NewBloomConstrain(&cap, &acc)
	assert.EqualError(t, err, "false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")

	// test capacity and accuracy compatibility
	acc = .86
	_, err = NewBloomConstrain(&cap, &acc)
	assert.Nil(t, err)
}

// TestPutStr also tests PutBytes because PutStr calls PutBytes
// most of put functionality tested in TestExistsStr
func TestBloomPutStr(t *testing.T) {
	b := NewBloom()

	// make sure n increases on put
	b.PutStr("test")
	assert.Equal(t, 1, b.n)

	// make sure n stays the same after same insertion
	b.PutStr("test")
	assert.Equal(t, 1, b.n)
}

// TestExistsStr also tests ExistsBytes because ExistsStr calls ExistsBytes
func TestBloomExistsStr(t *testing.T) {

	type existTest struct {
		entry    string
		expected bool
	}

	validEntries := []existTest{
		{
			entry:    "exists1",
			expected: true,
		},
		{
			entry:    "exists2",
			expected: true,
		},
		{
			entry:    "exists3",
			expected: true,
		},
	}

	invalidEntries := []existTest{
		{
			entry:    "not-exists1",
			expected: false,
		},
		{
			entry:    "not-exists2",
			expected: false,
		},
		{
			entry:    "not-exists3",
			expected: false,
		},
	}
	tests := append(validEntries, invalidEntries...)

	// 512 bits
	b512 := NewBloom()
	// populate
	for _, test := range validEntries {
		b512.PutStr(test.entry)
	}
	// check if exists
	for _, test := range tests {
		got, _ := b512.ExistsStr(test.entry)
		assert.Equal(t, test.expected, got)
	}

}

func TestBlooomCapacityConstaint(t *testing.T) {
	cap := 5
	b, _ := NewBloomConstrain(&cap, nil)
	for i := 0; i < cap; i++ {
		_, err := b.PutStr(strconv.Itoa(i))
		assert.Nil(t, err)
	}
	// test already added
	_, err := b.PutStr(strconv.Itoa(0))
	assert.Nil(t, err)
	// should fail on 6th try
	_, err = b.PutStr("fail")
	assert.IsType(t, err, &CapacityError{})
}

func TestBloomAccuracyConstaint(t *testing.T) {
	acc := float64(0.001)
	b, _ := NewBloomConstrain(nil, &acc)
	_, err := b.PutStr("fail")
	assert.IsType(t, err, &AccuracyError{})
}

func TestBloomAccuracy(t *testing.T) {
	// test if accuracy is 1 when no entries
	b := NewBloom()
	assert.Equal(t, float64(1), b.Accuracy())
	// rest of accuracy tested in TestFalsePositiveRate
}

func TestFalsePositiveRate(t *testing.T) {
	// wolfram alpha: https://www.wolframalpha.com/input?i2d=true&i=Power%5B%5C%2840%291-Power%5B%5C%2840%291%E2%88%92%5C%2840%29Divide%5B1%2C256%5D%5C%2841%29%5C%2841%29%2C3%5D%5C%2841%29%2C3%5D

	type falsePositiveTest struct {
		len      int
		n        int
		k        int
		expected float64
	}

	tests := []falsePositiveTest{
		{
			len:      32,
			n:        1,
			k:        3,
			expected: 0.000001590564065, // from Wolfram alpha
		},
		{
			len:      32,
			n:        100,
			k:        5,
			expected: 0.466912801365704, // from Wolfram alpha
		},
		{
			len:      64,
			n:        1,
			k:        1,
			expected: 0.001953125, // from Wolfram alpha
		},
		{
			len:      64,
			n:        100,
			k:        4,
			expected: 0.0866265531605745800663284180779795784124460538781544397141586291, // from Wolfram alpha
		},
	}

	unit := 0.000000000000001
	for _, test := range tests {
		got := falsePositiveRate(test.len, test.n, test.k)
		got = round(got, unit)
		expected := round(test.expected, unit)
		assert.Equal(t, expected, got)
	}

}

// benchmark for PutStr
func BenchmarkBloomPutStr(b *testing.B) {
	bloom := NewBloom()
	b.Run(fmt.Sprintf("len_%d_bytes", 64), func(b *testing.B) {
		for j := 0; j < 100; j++ {
			bloom.PutStr(strconv.Itoa(j))
		}
	})
}

// benchmark for exists for ExistsStr
func BenchmarkBloomExistsStr(b *testing.B) {
	bloom := NewBloom()
	for j := 0; j < 100; j++ {
		bloom.PutStr(strconv.Itoa(j))
	}
	b.Run(fmt.Sprintf("len_%d_bytes", 64), func(b *testing.B) {
		for j := 0; j < 100; j++ {
			bloom.ExistsStr(strconv.Itoa(j))
		}
	})
}

//
// helpers
//

func round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}
