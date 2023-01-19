package bloom

import (
	"fmt"
	"math"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
)

var (
	testk int = 3 // used with NewBloomFromK
)

func TestNewBloomFromCapacity(t *testing.T) {
	// test zero capacity
	_, err := NewBloomFromCap(0)
	assert.EqualError(t, err, "capacity cannot be less than 1")
}

func TestNewBloomFromAcc(t *testing.T) {
	// test zero accuracy
	_, err := NewBloomFromAcc(0)
	assert.EqualError(t, err, "false positive rate must be between 0 and 1")
	// test one accuracy
	_, err = NewBloomFromAcc(1)
	assert.EqualError(t, err, "false positive rate must be between 0 and 1")
}

func TestNewBloomFromK(t *testing.T) {
	// test zero k
	_, err := NewBloomFromK(0)
	assert.EqualError(t, err, "k cannot be less than 1")
}

// TestPutStr also tests PutBytes because PutStr calls PutBytes
// most of put functionality tested in TestExistsStr
func TestBloomPutStr(t *testing.T) {
	b, err := NewBloomFromK(testk)
	assert.Nil(t, err)

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
	b512, err := NewBloomFromK(testk)
	assert.Nil(t, err)
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
	b, err := NewBloomFromK(testk)
	assert.Nil(t, err)
	b.AddCapacityConstraint(cap)
	for i := 0; i < cap; i++ {
		_, err := b.PutStr(strconv.Itoa(i))
		assert.Nil(t, err)
	}
	// test already added
	_, err = b.PutStr(strconv.Itoa(0))
	assert.Nil(t, err)
	// should fail on 6th try
	_, err = b.PutStr("fail")
	assert.IsType(t, err, &CapacityError{})

	// test capacity and accuracy incompatibility
	cap = 1000
	acc := .1
	b, err = NewBloomFromK(testk)
	assert.Nil(t, err)
	err = b.AddAccuracyConstraint(acc)
	assert.Nil(t, err)
	err = b.AddCapacityConstraint(cap)
	assert.EqualError(t, err, "false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")
}

func TestBloomAccuracyConstaint(t *testing.T) {
	acc := float64(0.00000001)
	b, err := NewBloomFromK(testk)
	assert.Nil(t, err)
	b.AddAccuracyConstraint(acc)
	_, err = b.PutStr("fail")
	assert.IsType(t, err, &AccuracyError{})

	// test capacity and accuracy incompatibility
	cap := 1000
	acc = .1
	b, err = NewBloomFromK(testk)
	assert.Nil(t, err)
	err = b.AddCapacityConstraint(cap)
	assert.Nil(t, err)
	err = b.AddAccuracyConstraint(acc) // accuracy call should fail
	assert.EqualError(t, err, "false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")

}

func TestBloomAccuracy(t *testing.T) {
	// test if accuracy is 1 when no entries
	b, err := NewBloomFromK(testk)
	assert.Nil(t, err)
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

func TestCalcKFromCap(t *testing.T) {

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

// tests CalcKFromCap and CalcKFromAcc
func TestCalcK(t *testing.T) {

	type calcKTest struct {
		len int
		cap int
		acc float64
		k   int
	}
	// wolfram alpha: https://www.wolframalpha.com/input?i2d=true&i=ln%5C%2840%292%5C%2841%29+*+Divide%5B8%2C1%5D
	// wolfram alpha: https://www.wolframalpha.com/input?i2d=true&i=Power%5B%5C%2840%291-Power%5B%5C%2840%291%E2%88%92%5C%2840%29Divide%5B1%2C8%5D%5C%2841%29%5C%2841%29%2C6%5D%5C%2841%29%2C6%5D
	tests := []calcKTest{
		{
			len: 1,
			cap: 1,
			acc: 0.0280464168329186605336551944777514718592436369517315402492269304,
			k:   6,
		},
		{
			len: 32,
			cap: 20,
			acc: 0.0021607936448554444622427225356997501705187205987080810803711319,
			k:   9,
		},
		{
			len: 64,
			cap: 100,
			acc: 0.0866265531605745800663284180779795784124460538781544397141586291,
			k:   4,
		},
		{
			len: 1000,
			cap: 1000,
			acc: 0.0215825752781131301828435548816586055221500561616652196050130830,
			k:   6,
		},
	}

	for _, test := range tests {
		got := calcKFromCap(test.len, test.cap)
		assert.Equal(t, test.k, got)
		got = calcKFromAcc(test.len, test.acc)
		assert.Equal(t, test.k, got)
	}
}

func TestConstraintsCompatible(t *testing.T) {

	type constaintsTest struct {
		len      int
		cap      int
		acc      float64
		k        int
		expected bool
	}

	tests := []constaintsTest{
		{
			len:      32,
			cap:      20,
			acc:      0.00001,
			k:        9,
			expected: false,
		},
		{
			len:      32,
			cap:      20,
			acc:      0.1,
			k:        9,
			expected: true,
		},
	}

	for _, test := range tests {
		got := constraintsCompatible(test.len, test.cap, test.k, test.acc)
		assert.Equal(t, test.expected, got)
	}
}

//
// Benchmarks
//

// benchmark for PutStr
func BenchmarkBloomPutStr(b *testing.B) {
	bloom, err := NewBloomFromK(testk)
	assert.Nil(b, err)
	b.Run(fmt.Sprintf("len_%d_bytes", 64), func(b *testing.B) {
		for j := 0; j < 100; j++ {
			bloom.PutStr(strconv.Itoa(j))
		}
	})
}

// benchmark for exists for ExistsStr
func BenchmarkBloomExistsStr(b *testing.B) {
	bloom, err := NewBloomFromK(testk)
	assert.Nil(b, err)
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
