package bloom

import (
	"crypto/sha512"
	"errors"
	"math"
)

type Bloomer interface {
	// put in bloom. true if successful
	putStr(string) (bool)
	putBytes([]byte) (bool)

	// checks for existance. Float is accuracy: (1 - probability of false positive).
	existsStr(string) (bool, float64);
	existsBytes([]byte) (bool, float64);

	// returns accuracy: (probability of exist returning a false positive at current n entries)
	accuracy() float64;
}

type Bloom struct {
	// current number of entries
	n int

	// bloom filter bytes
	bs []byte

	// number of bytes. only 32 or 64 allowed
	size int

	// optional, the maximum allowed false positive rate until no more entries accepted
	maxFalsePositiveRate *float64

	// optional, maximum number of entries allowed
	cap *int
}

// Bloom type constructors

func NewBloom32() *Bloom {
	return &Bloom{
		n: 0,
		bs: make([]byte, 32),
		size: 32,
		maxFalsePositiveRate: nil,  // don't care about accuracy unless specified
		cap: nil,        			// don't care about capacity unless specified
	}
}

func NewBloom64() *Bloom {
	return &Bloom{
		n: 0,
		bs: make([]byte, 64),
		size: 64,
		maxFalsePositiveRate: nil,  // don't care about accuracy unless specified
		cap: nil,        			// don't care about capacity unless specified
	}
}

func NewBloom() *Bloom {
	return NewBloom64()
}

// cap: max number of entries
// min_accuracy: max allowed
// ex: if maxFalsePositiveRate is 0.1 then 10% chance of false positive when capacity is full
func NewBloomConstrain(cap *int, maxFalsePositiveRate *float64) (*Bloom, error) {
	// use 64
	b := NewBloom64()

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

	// check if the capacity and maxFalsePositiveRate are even possible together with 64 byte size
	calcMaxFalsePositiveRate := falsePositiveRate(64, *cap)
	if calcMaxFalsePositiveRate > *maxFalsePositiveRate  {
		// if the maximum calculated false positive rate is greater user inputed allowed false positive rate, fail.
		return nil, errors.New("false positive rate will be higher in full capacity bloom filter than the maxFalsePositiveRate provided")
	}

	return b, nil
}

// checks for membership of bytes element
func (b *Bloom) existsBytes(bs []byte) (bool, float64) {
	h := sha512.Sum512(bs)
	for i := 0; i < b.size; i++ {
		if (b.bs[i] | h[i]) != b.bs[i] {
			// bloom OR hash changes bloom which means there are 1's present in hash not in bloom
			return false, 1
		}
	}
	return true, b.accuracy()
}

// checks for membership of string element
func (b *Bloom) existsStr(s string) (bool, float64) {
	bs := []byte(s)
	return b.existsBytes(bs)
}

// get false positive rate
func (b *Bloom) accuracy() float64 {
	if b.n == 0 {
		return 1
	}
	return falsePositiveRate(b.size, b.n)
}

func falsePositiveRate(n, size int) float64 {
	// equation: 1 - (1/m)^n where m is bits and n is entries
	// math here: https://brilliant.org/wiki/bloom-filter/
	base := 1 - float64(1)/float64(8*size)
	falsePositiveRate := math.Pow(base, float64(n))
	return falsePositiveRate
}
