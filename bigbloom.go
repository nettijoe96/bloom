package bloom

import (
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

// BigBloom is a bloom filter with a variable length. It uses SHA512 hashes with a nonce.
type BigBloom struct {
	// current number of unique entries
	n int

	// number of hash functions
	k int

	// bloom filter bytes
	bs []byte

	// number of bytes
	len int

	// optional, maximum number of unique entries allowed
	cap *int

	// optional, the maximum allowed false positive rate until no more entries accepted
	maxFalsePositiveRate *float64

	// is loaded using FromBytes. This is used to ignore accuracy calculations
	isLoaded bool
}

//
// Constructors
//

// Constructs len-byte bloom filter with no constraints.
func NewBigBloom(len int) *BigBloom {
	return &BigBloom{
		n:                    0,
		k:                    3,
		bs:                   make([]byte, len),
		len:                  len,
		maxFalsePositiveRate: nil,
		cap:                  nil,
		isLoaded:             false,
	}
}

// Constructs bloom filter with the minimum length to satisify both constraints.
func NewBigBloomAlloc(cap int, maxFalsePositiveRate float64) (*BigBloom, error) {
	if cap < 1 {
		return nil, errors.New("capacity cannot be less than 1")
	}
	if maxFalsePositiveRate <= 0 {
		return nil, errors.New("false positive rate must be greater than 0")
	}

	// solving for m in equation: acc = 1 - e^(-n/m)
	// see math https://brilliant.org/wiki/bloom-filter/
	bits := -1 * (float64(cap) / math.Log(1-maxFalsePositiveRate))
	// take ceiling, rounding down could cause the constaint to be reached before max capacity
	len := int(math.Ceil(bits / 8))

	return &BigBloom{
		n:                    0,
		k:                    3,
		bs:                   make([]byte, len),
		len:                  len,
		maxFalsePositiveRate: &maxFalsePositiveRate,
		cap:                  &cap,
		isLoaded:             false,
	}, nil

}

// Equivalent to NewBloomConstain. Constructs len-byte bloom filter with constraints.
// If cap is provided, bloom filter will not allow for more than that amount of unique elements.
// If maxFalsePositiveRate is provided then the false positive rate of the filter will not be allowed to increase beyond that amount.
// ex: if maxFalsePositiveRate is 0.1 then 10% chance of false positive when capacity is full
func NewBigBloomConstrain(len int, cap *int, maxFalsePositiveRate *float64) (*BigBloom, error) {
	b := NewBigBloom(len)

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
	calcMaxFalsePositiveRate := falsePositiveRate(len, *cap)
	if calcMaxFalsePositiveRate > *maxFalsePositiveRate {
		// if the maximum calculated false positive rate is greater user inputed allowed false positive rate, fail.
		return nil, errors.New("false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")
	}

	return b, nil
}

// Load bloom filter from bytes of bloom filter
// This is useful for loading in a Bloom filter over the wire.
// This mechanism will disable accuracy calculations because n is unknown
func FromBytes(bs []byte) *BigBloom {
	return &BigBloom{
		n:                    0,
		k:                    3,
		bs:                   bs,
		len:                  len(bs),
		maxFalsePositiveRate: nil,
		cap:                  nil,
		isLoaded:             true,
	}
}

//
// Methods
//

// Inserts string element into bloom filter. Returns an error if a constraint is violated.
func (b *BigBloom) PutStr(s string) (*BigBloom, error) {
	bs := []byte(s)
	return b.PutBytes(bs)
}

// Inserts bytes element into bloom filter. Returns an error if a constraint is violated.
func (b *BigBloom) PutBytes(bs []byte) (*BigBloom, error) {
	// if exists already just return filter and don't increase n
	if exists, _ := b.ExistsBytes(bs); exists {
		return b, nil
	}

	if b.cap != nil && b.n == *b.cap {
		return b, &CapacityError{cap: *b.cap}
	}

	if b.maxFalsePositiveRate != nil {
		if falsePositiveRate(b.len, b.n+1) > *b.maxFalsePositiveRate {
			return b, &AccuracyError{acc: *b.maxFalsePositiveRate}
		}
	}

	totBits := len(b.bs) * 8
	var h [64]byte = sha512.Sum512(bs)
	for i := 0; i < b.k; i++ {
		bytes := h[i : i+8]
		bitI := binary.BigEndian.Uint64(bytes) % uint64(totBits)
		byteI := int(math.Floor(float64(bitI) / float64(8)))
		iInByte := bitI % 8
		bitFlip := byte(1 << iInByte)
		b.bs[byteI] = b.bs[byteI] | bitFlip
	}

	b.n++
	return b, nil
}

// Checks for existance of a string in a bloom filter. Returns boolean and false positive rate.
func (b *BigBloom) ExistsStr(s string) (bool, float64) {
	bs := []byte(s)
	return b.ExistsBytes(bs)
}

// Checks for existance of bytes element in a bloom filter. Returns boolean and false positive rate.
func (b *BigBloom) ExistsBytes(bs []byte) (bool, float64) {

	totBits := len(b.bs) * 8
	var h [64]byte = sha512.Sum512(bs)
	for i := 0; i < b.k; i++ {
		bytes := h[i : i+8]
		bitI := binary.BigEndian.Uint64(bytes) % uint64(totBits)
		byteI := int(math.Floor(float64(bitI) / float64(8)))
		iInByte := bitI % 8
		bitFlip := byte(1 << iInByte)
		if b.bs[byteI] != b.bs[byteI]|bitFlip {
			return false, 1
		}
	}
	return true, b.Accuracy()
}

// Get false positive rate
// -1 means cannot be calcuated because it is loaded in
func (b *BigBloom) Accuracy() float64 {
	if b.isLoaded {
		return -1
	}
	if b.n == 0 {
		return 1
	}
	return falsePositiveRate(b.len, b.n)
}

func (b *BigBloom) String() string {
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
