package bloom

import (
	"fmt"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
)

func TestNewBigBloomFromCapacity(t *testing.T) {
	// test zero capacity
	_, err := NewBigBloomFromCap(32, 0)
	assert.EqualError(t, err, "capacity cannot be less than 1")
}

func TestNewBigBloomFromAcc(t *testing.T) {
	// test zero accuracy
	_, err := NewBigBloomFromAcc(32, 0)
	assert.EqualError(t, err, "false positive rate must be between 0 and 1")
	// test one accuracy
	_, err = NewBigBloomFromAcc(32, 1)
	assert.EqualError(t, err, "false positive rate must be between 0 and 1")
}

func TestNewBigBloomFromK(t *testing.T) {
	// test zero k
	_, err := NewBigBloomFromK(32, 0)
	assert.EqualError(t, err, "k cannot be less than 1")
}

func TestNewBigBloomAlloc(t *testing.T) {
	// test zero capacity
	zeroCap := 0
	_, err := NewBigBloomAlloc(zeroCap, 1)
	assert.EqualError(t, err, "capacity cannot be less than 1")

	// test zero false postive rate
	zeroAcc := float64(0)
	_, err = NewBigBloomAlloc(1, zeroAcc)
	assert.EqualError(t, err, "false positive rate must be between 0 and 1")

	type allocTest struct {
		cap         int
		acc         float64
		expectedLen int
	}

	// wolfram alpha: https://www.wolframalpha.com/input?i2d=true&i=Power%5B%5C%2840%291-Power%5B%5C%2840%291%E2%88%92%5C%2840%29Divide%5B1%2C4096%5D%5C%2841%29%5C%2841%29%2C3000%5D%5C%2841%29%2C3%5D
	tests := []allocTest{
		{
			cap:         10,
			acc:         0.0013597239492769301702394684205393411304012623906373231471869292,
			expectedLen: 18, // 144 bits
		},
		{
			cap:         1000,
			acc:         0.1400406877800123403129581978899597802443405570160297883718149039,
			expectedLen: 512, // 512 bytes
		},
	}

	for _, test := range tests {
		b, _ := NewBigBloomAlloc(test.cap, test.acc)
		assert.Equal(t, test.expectedLen, b.len)
	}
}

// TestPutStr also tests PutBytes because PutStr calls PutBytes
// most of put functionality tested in TestExistsStr
func TestBigBloomPutStr(t *testing.T) {
	b, err := NewBigBloomFromK(32, testk)
	assert.Nil(t, err)

	// make sure n increases on put
	b.PutStr("test")
	assert.Equal(t, 1, b.n)

	// make sure n stays the same after same insertion
	b.PutStr("test")
	assert.Equal(t, 1, b.n)
}

// TestExistsStr also tests ExistsBytes because ExistsStr calls ExistsBytes
func TestBigBloomExistsStr(t *testing.T) {

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

	// small filter test
	// 10 bytes will require 1 hash
	b10, err := NewBigBloomFromK(5, testk)
	assert.Nil(t, err)

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
	b1000, err := NewBigBloomFromK(1000, 3)
	assert.Nil(t, err)
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
	b10000, err := NewBigBloomFromK(10000, 3)
	assert.Nil(t, err)
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

func TestBigBloomAccuracy(t *testing.T) {
	// test if accuracy is 1 when no entries
	b, err := NewBigBloomFromK(32, testk)
	assert.Nil(t, err)
	assert.Equal(t, float64(1), b.Accuracy())

	// cannot calculate accuracy if loaded in
	b.isLoaded = true
	assert.Equal(t, float64(-1), b.Accuracy())

	// rest of accuracy tested in TestFalsePositiveRate
}

// tests huge bloom filter
func TestTrillionBitBloom(t *testing.T) {
	m := 125000000000
	b, err := NewBigBloomFromCap(m, 100000)
	assert.Nil(t, err)
	_, err = b.PutStr("test")
	assert.Nil(t, err)
	exists, _ := b.ExistsStr("test")
	assert.True(t, exists)
	exists, _ = b.ExistsStr("fail")
	assert.False(t, exists)
}

//
// Benchmarks
//

// benchmark for increasing bloom filter len
func BenchmarkBigBloomPutStr(b *testing.B) {
	for i := 512; i < 10000; i += 512 {
		bloom, err := NewBigBloomFromK(i, 3)
		assert.Nil(b, err)
		b.Run(fmt.Sprintf("len_%d_bytes", i), func(b *testing.B) {
			for j := 0; j < 100; j++ {
				bloom.PutStr(strconv.Itoa(j))
			}
		})
	}
}

// benchmark for exists for increasing bloom filter len
func BenchmarkBigBloomExistsStr(b *testing.B) {
	for i := 512; i < 10000; i += 512 {
		bloom, err := NewBigBloomFromK(i, 3)
		assert.Nil(b, err)
		for j := 0; j < 100; j++ {
			bloom.PutStr(strconv.Itoa(j))
		}
		b.Run(fmt.Sprintf("len_%d_bytes", i), func(b *testing.B) {
			for j := 0; j < 100; j++ {
				bloom.ExistsStr(strconv.Itoa(j))
			}
		})
	}
}
