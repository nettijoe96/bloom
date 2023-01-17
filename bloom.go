package bloom

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
)

type Bloomer interface {
	// put in bloom: true if successful
	PutStr(string) bool
	PutBytes([]byte) bool

	// checks for existance: returns true if exists and float64 for false positive rate
	ExistsStr(string) (bool, float64)
	ExistsBytes([]byte) (bool, float64)

	// checks accuracy: returns current false positive rate. returns -1 if accuracy cannot be calculated
	Accuracy() float64
}

// Bloom type is a 512-bit bloom filter that uses a single SHA256 hash.
type Bloom struct {
	// current number of unique entries.
	n int

	// number of hash functions
	k int

	// bloom filter bytes
	bs [64]byte

	// always set to 64
	len int

	// optional, maximum number of unique entries allowed
	cap *int

	// optional, the maximum allowed false positive rate until no more entries accepted
	maxFalsePositiveRate *float64
}

type CapacityError struct {
	cap int
}

func (e *CapacityError) Error() string {
	return fmt.Sprintf("failed to add entry: bloom filter at max capacity %d", e.cap)
}

type AccuracyError struct {
	acc float64
}

func (e *AccuracyError) Error() string {
	return fmt.Sprintf("failed to add entry: bloom filter constrained by max false positive rate %f", e.acc)
}

//
// Bloom type constructors
//

// Constructs 512-bit bloom filter with no constaints
func NewBloom() *Bloom {
	return &Bloom{
		n:                    0,
		k:                    3, // TODO make variable
		bs:                   [64]byte{},
		len:                  64,
		maxFalsePositiveRate: nil, // don't care about accuracy unless specified
		cap:                  nil, // don't care about capacity unless specified
	}
}

// Constructs 512-bit bloom filter with constraints. If cap is provided, bloom filter will not allow for more than that amount of unique elements.
// If maxFalsePositiveRate is provided then the false positive rate of the filter will not be allowed to increase beyond that amount
// ex: if maxFalsePositiveRate is 0.1 then 10% chance of false positive when capacity is full
func NewBloomConstrain(cap *int, maxFalsePositiveRate *float64) (*Bloom, error) {
	b := NewBloom()

	if cap != nil {
		if *cap < 1 {
			return nil, errors.New("capacity cannot be less than 1")
		}
		b.cap = cap
	}
	if maxFalsePositiveRate != nil {
		if *maxFalsePositiveRate <= 0 {
			return nil, errors.New("false positive rate must be greater than 0")
		}
		b.maxFalsePositiveRate = maxFalsePositiveRate
	}
	if cap == nil || maxFalsePositiveRate == nil {
		// do not need to check for compatibility of constraints.
		return b, nil
	}

	// check if contraints capacity and maxFalsePositiveRate are compatible together with this size bloom filter
	calcMaxFalsePositiveRate := falsePositiveRate(b.len, *cap, b.k)
	if calcMaxFalsePositiveRate > *maxFalsePositiveRate {
		// if the maximum calculated false positive rate is greater user inputed allowed false positive rate, fail.
		return nil, errors.New("false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")
	}

	return b, nil
}

//
// Methods
//

// Inserts string element into bloom filter. Returns an error if a constraint is violated.
func (b *Bloom) PutStr(s string) (*Bloom, error) {
	bs := []byte(s)
	return b.PutBytes(bs)
}

// Inserts bytes element into bloom filter. Returns an error if a constraint is violated.
func (b *Bloom) PutBytes(bs []byte) (*Bloom, error) {
	// if exists already just return filter and don't increase n
	if exists, _ := b.ExistsBytes(bs); exists {
		return b, nil
	}

	if b.cap != nil && b.n == *b.cap {
		return b, &CapacityError{cap: *b.cap}
	}

	if b.maxFalsePositiveRate != nil {
		if falsePositiveRate(b.len, b.n+1, b.k) > *b.maxFalsePositiveRate {
			return b, &AccuracyError{acc: *b.maxFalsePositiveRate}
		}
	}

	var h [32]byte = sha256.Sum256(bs)
	for i := 0; i < b.k; i++ {
		// two bytes is more than enough to cover 512 possibilities
		bytes := h[i : i+2]
		// find index of bit
		bitI := binary.BigEndian.Uint16(bytes) % 512
		// find index of byte
		byteI := int(math.Floor(float64(bitI) / float64(8)))
		// bit shift 1
		iInByte := bitI % 8
		// bit shift 1
		bitFlip := byte(1 << iInByte)
		// set bit to 1
		b.bs[byteI] = b.bs[byteI] | bitFlip
	}
	b.n++
	return b, nil
}

// Checks for existance of a string in a bloom filter. Returns boolean and false positive rate.
func (b *Bloom) ExistsStr(s string) (bool, float64) {
	bs := []byte(s)
	return b.ExistsBytes(bs)
}

// Checks for existance of bytes element in a bloom filter. Returns boolean and false positive rate.
func (b *Bloom) ExistsBytes(bs []byte) (bool, float64) {
	var h [32]byte = sha256.Sum256(bs)
	for i := 0; i < b.k; i++ {
		// two bytes is more than enough to cover 512 possibilities
		bytes := h[i : i+2]
		// find index of bit
		bitI := binary.BigEndian.Uint16(bytes) % 512
		// find index of byte
		byteI := int(math.Floor(float64(bitI) / float64(8)))
		// find index of bit within byte
		iInByte := bitI % 8
		// bit shift 1
		bitFlip := byte(1 << iInByte)
		// it doesn't exists if there is a bitFlip
		if b.bs[byteI] != b.bs[byteI]|bitFlip {
			return false, 1
		}
	}
	return true, b.Accuracy()
}

// Get false positive rate
func (b *Bloom) Accuracy() float64 {
	if b.n == 0 {
		return 1
	}
	return falsePositiveRate(b.len, b.n, b.k)
}

func (b *Bloom) String() string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("%d-bit bloom filter: %d unique entries", 8*b.len, b.n))
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

// converts bytes of bloom filter to hex string
func (b *Bloom) Hex() string {
	return hex.EncodeToString(b.bs[:])
}

//
// helpers
//

func falsePositiveRate(len, n, k int) float64 {
	// equation: 1-((1 - (1/m))^nk)^k where m is bits, n is unique entries, and k is number of hashes
	// math here: https://brilliant.org/wiki/bloom-filter/
	m := len * 8
	base := 1 - float64(1)/float64(m)
	inner := 1 - math.Pow(base, float64(n*k))
	falsePositiveRate := math.Pow(inner, float64(k))
	return falsePositiveRate
}
