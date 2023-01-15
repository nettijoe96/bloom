package bloom

import (
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
)

func TestNewBigBloomAlloc(t *testing.T) {
	// test zero capacity
	zeroCap := 0
	_, err := NewBigBloomConstrain(32, &zeroCap, nil)
	assert.EqualError(t, err, "capacity cannot be less than 1")

	// test zero false postive rate
	zeroAcc := float64(0)
	_, err = NewBigBloomConstrain(32, nil, &zeroAcc)
	assert.EqualError(t, err, "false positive rate must be greater than 0")

	type allocTest struct {
		cap int
		acc float64
		expectedLen int
	}

	tests := []allocTest{
		{
			cap: 10,
			acc: 0.038382958383573,
			expectedLen: 32, // 256 bits
		},
		{
			cap: 1000,
			acc: 0.01,
			expectedLen: 12438, // wolfram alpha
		},
	}

	for _, test := range tests {
		b, _ := NewBigBloomAlloc(test.cap, test.acc)
		assert.Equal(t, test.expectedLen, b.len)
	}
}

func TestNewBigBloomConstain(t *testing.T) {
	// test zero capacity
	zeroCap := 0
	_, err := NewBigBloomConstrain(32, &zeroCap, nil)
	assert.EqualError(t, err, "capacity cannot be less than 1")

	// test zero false postive rate
	zeroAcc := float64(0)
	_, err = NewBigBloomConstrain(32, nil, &zeroAcc)
	assert.EqualError(t, err, "false positive rate must be greater than 0")

	// test capacity and accuracy incompatibility
	cap := 1000
	acc := .1
	_, err = NewBigBloomConstrain(32, &cap, &acc)
	assert.EqualError(t, err, "false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")

	// test capacity and accuracy compatibility
	acc = .86
	_, err = NewBigBloomConstrain(64, &cap, &acc)
	assert.Nil(t, err)
}

// TestPutStr also tests PutBytes because PutStr calls PutBytes
// most of put functionality tested in TestExistsStr
func TestBigBloomPutStr(t *testing.T) {
	b := NewBigBloom(32)

	// make sure n increases on put
	b.PutStr("test")
	assert.Equal(t, 1, b.n)

	// TODO: add test here if I choose to make n not increase when already exists on put
}

// TestExistsStr also tests ExistsBytes because ExistsStr calls ExistsBytes
func TestBigBloomExistsStr(t *testing.T) {

	type existTest struct {
		entry string
		expected bool
	}

	validEntries := []existTest{
		{
			entry: "exists1",
			expected: true,
		},
		{
			entry: "exists2",
			expected: true,
		},
		{
			entry: "exists3",
			expected: true,
		},
	}

	invalidEntries := []existTest{
		{
			entry: "not-exists1",
			expected: false,
		},
		{
			entry: "not-exists2",
			expected: false,
		},
		{
			entry: "not-exists3",
			expected: false,
		},
	}
	tests := append(validEntries, invalidEntries...)

	// small filter test
	// 10 bytes will require 1 hash
	b10 := NewBigBloom(5)
	// populate
	for _, test := range validEntries {
		b10.PutStr(test.entry)
	}
	// check if exists
	for _, test := range tests {
		got, _ := b10.ExistsStr(test.entry)
		assert.Equal(t, test.expected, got)
	}

	// big filter test
	// 1000 bytes will require 32 hashes
	b1000 := NewBigBloom(1000)
	// populate
	for _, test := range validEntries {
		b1000.PutStr(test.entry)
	}
	// check if exists
	for _, test := range tests {
		got, _ := b1000.ExistsStr(test.entry)
		assert.Equal(t, test.expected, got)
	}

	// huge filter test
	// 10000 bytes will require 320 hashes
	b10000 := NewBigBloom(10000)
	// populate
	for _, test := range validEntries {
		b10000.PutStr(test.entry)
	}
	// check if exists
	for _, test := range tests {
		got, _ := b10000.ExistsStr(test.entry)
		assert.Equal(t, test.expected, got)
	}

}

func TestBigBloomCapacityConstaint(t *testing.T) {
	cap := 5
	b, _ := NewBigBloomConstrain(32, &cap, nil)
	for i := 0; i < cap; i++ {
		_, err := b.PutStr(strconv.Itoa(i))
		assert.Nil(t, err)
	}
	// TODO: add test here if I choose to make n not increase when already exists on put
	// should fail on 6th try
	_, err := b.PutStr("fail")
	assert.IsType(t, err, &CapacityError{})
}

func TestBigBloomAccuracyConstaint(t *testing.T) {
	acc := float64(0.001)
	b, _ := NewBigBloomConstrain(32, nil, &acc)
	_, err := b.PutStr("fail")
	assert.IsType(t, err, &AccuracyError{})
}

func TestBigBloomAccuracy(t *testing.T) {
	// test if accuracy is 1 when no entries
	b := NewBigBloom(32)
	assert.Equal(t, float64(1), b.Accuracy())
	// rest of accuracy tested in TestFalsePositiveRate
}