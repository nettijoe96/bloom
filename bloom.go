package bloom

import (
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"math"
	"strings"
)

type Bloomer interface {
	// put in bloom: true if successful
	PutStr(string) (bool)
	PutBytes([]byte) (bool)

	// checks for existance: returns true if exists and float64 for false positive rate
	ExistsStr(string) (bool, float64);
	ExistsBytes([]byte) (bool, float64);

	// checks accuracy: returns current false positive rate
	Accuracy() float64;
}

type Bloom struct {
	// current number of entries
	n int

	// bloom filter bytes
	bs []byte

	// number of bytes. only 32 or 64 allowed
	size int

	// optional, maximum number of entries allowed
	cap *int

	// optional, the maximum allowed false positive rate until no more entries accepted
	maxFalsePositiveRate *float64
}

type CapacityError struct{
	cap int
}
func (e *CapacityError) Error() string {
	return fmt.Sprintf("failed to add entry: bloom filter at max capacity %d", e.cap)
}

type AccuracyError struct{
	acc float64
}
func (e *AccuracyError) Error() string {
	return fmt.Sprintf("failed to add entry: bloom filter constrained by max false positive rate %f", e.acc)
}

//
// Bloom type constructors
//

// 256 bit bloom filter
func NewBloom256() *Bloom {
	return &Bloom{
		n: 0,
		bs: make([]byte, 32),
		size: 32,
		maxFalsePositiveRate: nil,  // don't care about accuracy unless specified
		cap: nil,        			// don't care about capacity unless specified
	}
}

// 512 bit bloom filter
func NewBloom512() *Bloom {
	return &Bloom{
		n: 0,
		bs: make([]byte, 64),
		size: 64,
		maxFalsePositiveRate: nil,  // don't care about accuracy unless specified
		cap: nil,        			// don't care about capacity unless specified
	}
}

// makes a new bloom of 512 bits
func NewBloom() *Bloom {
	return NewBloom512()
}

// cap: max number of entries
// min_accuracy: max allowed
// ex: if maxFalsePositiveRate is 0.1 then 10% chance of false positive when capacity is full
func NewBloomConstrain(cap *int, maxFalsePositiveRate *float64) (*Bloom, error) {
	// use larger 512 bloom filter
	b := NewBloom512()

	if maxFalsePositiveRate != nil {
		if *maxFalsePositiveRate < 0 {
			return nil, errors.New("false positive rate cannot be negative")
		}
		b.maxFalsePositiveRate = maxFalsePositiveRate
	}
	if cap != nil {
		if *cap < 0 {
			return nil, errors.New("capacity cannot be negative")
		}
		b.cap = cap
	}
	if cap == nil || maxFalsePositiveRate == nil {
		// do not need to check for compatibility of constraints.
		return b, nil
	}

	// check if contraints capacity and maxFalsePositiveRate are compatible together with this size bloom filter
	calcMaxFalsePositiveRate := falsePositiveRate(b.size, *cap)
	if calcMaxFalsePositiveRate > *maxFalsePositiveRate  {
		// if the maximum calculated false positive rate is greater user inputed allowed false positive rate, fail.
		return nil, errors.New("false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")
	}

	return b, nil
}

//
// Methods
//

// adds byte data to bloom filter
func (b *Bloom) PutBytes(bs []byte) (*Bloom, error) {
	if b.cap != nil && b.n == *b.cap{
		return b, &CapacityError{cap: *b.cap}
	}

	if b.maxFalsePositiveRate != nil {
		if falsePositiveRate(b.size, b.n + 1) > *b.maxFalsePositiveRate {
			return b, &AccuracyError{acc: *b.maxFalsePositiveRate}
		}
	}

	if b.size == 32 {
		var h [32]byte = sha256.Sum256(bs)
		for i := 0; i < b.size; i++ {
			// set bloom byte to old byte OR hash
			b.bs[i] = b.bs[i] | h[i]
		}
	}
	if b.size == 64 {
		var h [64]byte = sha512.Sum512(bs)
		for i := 0; i < b.size; i++ {
			for i := 0; i < b.size; i++ {
				// set bloom byte to old byte OR hash
				b.bs[i] = b.bs[i] | h[i]
			}
		}
	}
	b.n++
	return b, nil
}

// adds string to bloom filter
func (b *Bloom) PutStr(s string) (*Bloom, error) {
	bs := []byte(s)
	return b.PutBytes(bs)
}

// checks for membership of bytes element
func (b *Bloom) ExistsBytes(bs []byte) (bool, float64) {
	if b.size == 32 {
		var h [32]byte = sha256.Sum256(bs)
		for i := 0; i < b.size; i++ {
			if (b.bs[i] | h[i]) != b.bs[i] {
				// bloom OR hash changes bloom which means there are 1's present in hash not in bloom
				return false, 1
			}
		}
	}
	if b.size == 64 {
		var h [64]byte = sha512.Sum512(bs)
		for i := 0; i < b.size; i++ {
			if (b.bs[i] | h[i]) != b.bs[i] {
				// bloom OR hash changes bloom which means there are 1's present in hash not in bloom
				return false, 1
			}
		}
	}
	return true, b.Accuracy()
}

// checks for membership of string element
func (b *Bloom) ExistsStr(s string) (bool, float64) {
	bs := []byte(s)
	return b.ExistsBytes(bs)
}

// get false positive rate
func (b *Bloom) Accuracy() float64 {
	if b.n == 0 {
		return 1
	}
	return falsePositiveRate(b.size, b.n)
}

func (b *Bloom) String() string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("%d-bit bloom filter: %d entries", 8*b.size, b.n))
	if b.cap != nil {
		buf.WriteString(fmt.Sprintf(", max cap %d", *b.cap))
	}
	if b.maxFalsePositiveRate != nil {
		buf.WriteString(fmt.Sprintf(", max false positive rate %f", *b.maxFalsePositiveRate))
	}
	if b.cap == nil && b.maxFalsePositiveRate == nil {
		buf.WriteString(", no constraints")
	}

	return buf.String()
}

//
// helpers
//

func falsePositiveRate(bytes, n int) float64 {
	// equation: 1-((1 - (1/m))^n) where m is bits and n is entries. k variable (# hash functions) not implemented.
	// math here: https://brilliant.org/wiki/bloom-filter/
	base := 1 - float64(1)/float64(bytes*8)
	falsePositiveRate := 1 - math.Pow(base, float64(n))
	return falsePositiveRate
}
