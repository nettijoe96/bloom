package bloom

import (
	"math"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
)

func TestBloomConstained(t *testing.T) {
	// test negative capacity
	negCap := -1
	_, err := NewBloomConstrain(&negCap, nil)
	assert.EqualError(t, err, "capacity cannot be less than 1")

	// test negative accuracy
	negAcc := float64(-1)
	_, err = NewBloomConstrain(nil, &negAcc)
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
func TestPutStr(t *testing.T) {
	b := NewBloom()

	// make sure n increases on put
	b.PutStr("test")
	assert.Equal(t, 1, b.n)

	// TODO: add test here if I choose to make n not increase when already exists on put
}

// TestExistsStr also tests ExistsBytes because ExistsStr calls ExistsBytes
func TestExistsStr(t *testing.T) {

	type ExistTest struct {
		entry string
		expected bool
	}

	validEntries := []ExistTest{
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

	invalidEntries := []ExistTest{
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

func TestCapacityConstaint(t *testing.T) {
	cap := 5
	b, _ := NewBloomConstrain(&cap, nil)
	for i := 0; i < cap; i++ {
		_, err := b.PutStr(strconv.Itoa(i))
		assert.Nil(t, err)
	}
	// TODO: add test here if I choose to make n not increase when already exists on put
	// should fail on 6th try
	_, err := b.PutStr("fail")
	assert.IsType(t, err, &CapacityError{})
}

func TestAccuracyConstaint(t *testing.T) {
	acc := float64(0.001)
	b, _ := NewBloomConstrain(nil, &acc)
	_, err := b.PutStr("fail")
	assert.IsType(t, err, &AccuracyError{})
}

func TestAccuracy(t *testing.T) {
	// test if accuracy is 1 when no entries
	b := NewBloom()
	assert.Equal(t, float64(1), b.Accuracy())
	// rest of accuracy tested in TestFalsePositiveRate
}

func TestFalsePositiveRate(t *testing.T) {

	type FalsePositiveTest struct {
		bytes int
		n int
		expected float64
	}

	tests := []FalsePositiveTest{
		{
			bytes: 32,
			n: 1,
			expected: 0.00390625, // from Wolfram alpha
		},
		{
			bytes: 32,
			n: 100,
			expected: 0.3238835348904526709877393077971628097768978560726550437785281290, // from Wolfram alpha
		},
		{
			bytes: 64,
			n: 1,
			expected: 0.001953125, // from Wolfram alpha
		},
		{
			bytes: 64,
			n: 100,
			expected: 0.1775795214086141608493458302357950078335925130130317388097481965, // from Wolfram alpha
		},
	}

	unit := 0.000000000000001
	for _, test := range tests {
		got := falsePositiveRate(test.bytes, test.n)
		got = round(got, unit)
		expected := round(test.expected, unit)
		assert.Equal(t, expected, got)
	}

}

//
// helpers
//

func round(x, unit float64) float64 {
    return math.Round(x/unit) * unit
}
