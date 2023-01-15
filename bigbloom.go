package bloom

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"strings"
)

type BigBloom struct {
	// current number of entries
	n int

	// bloom filter bytes
	bs []byte

	// number of bytes
	len int

	// optional, maximum number of entries allowed
	cap *int

	// optional, the maximum allowed false positive rate until no more entries accepted
	maxFalsePositiveRate *float64
}

//
// Constructors
//

func NewBigBloom(bytes int) *BigBloom {
	return &BigBloom{
		n: 0,
		bs: make([]byte, bytes),
		len: bytes,
		maxFalsePositiveRate: nil,
		cap: nil,
	}
}

// constuctor that sets the minimum filter size that fulfills constaints
func NewBigBloomAlloc(cap int, maxFalsePositiveRate float64) (*BigBloom, error) {
	if cap < 1 {
		return nil, errors.New("capacity cannot be less than 1")
	}
	if maxFalsePositiveRate <= 0 {
		return nil, errors.New("false positive rate must be greater than 0")
	}

	bits := float64(cap) / math.Log(1-maxFalsePositiveRate)
	bytes := int(math.Ceil(bits/8))

	return &BigBloom{
		n: 0,
		bs: make([]byte, bytes),
		len: bytes,
		maxFalsePositiveRate: &maxFalsePositiveRate,
		cap: &cap,
	}, nil

}

// cap: max number of entries
// min_accuracy: max allowed
// ex: if maxFalsePositiveRate is 0.1 then 10% chance of false positive when capacity is full
func NewBigBloomConstrain(bits int, cap *int, maxFalsePositiveRate *float64) (*BigBloom, error) {
	b := NewBigBloom(bits)

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
	bytes := int(math.Ceil(float64(bits)/8))
	calcMaxFalsePositiveRate := falsePositiveRate(bytes, *cap)
	if calcMaxFalsePositiveRate > *maxFalsePositiveRate  {
		// if the maximum calculated false positive rate is greater user inputed allowed false positive rate, fail.
		return nil, errors.New("false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")
	}

	return b, nil
}

//
// Methods
//


// adds string to bloom filter
func (b *BigBloom) PutStr(s string) (*BigBloom, error) {
	bs := []byte(s)
	return b.PutBytes(bs)
}

// adds byte data to bloom filter
func (b *BigBloom) PutBytes(bs []byte) (*BigBloom, error) {
	if b.cap != nil && b.n == *b.cap{
		return b, &CapacityError{cap: *b.cap}
	}

	if b.maxFalsePositiveRate != nil {
		if falsePositiveRate(b.len, b.n + 1) > *b.maxFalsePositiveRate {
			return b, &AccuracyError{acc: *b.maxFalsePositiveRate}
		}
	}

	// concatenate a nonce that increments every 256 bits in order to enlargen the hash
	var nonce int
	var h [32]byte
	for i := 0; i < b.len; i++ {
		if i % 32 == 0 {
			// new hash unique has to constructed after 256 bits
			bsNonce := append(bs, byte(nonce))
			h = sha256.Sum256(bsNonce)
			nonce++
		}
		// set bloom byte to old byte OR hash
		b.bs[i] = b.bs[i] | h[i % 32]
	}
	b.n++
	return b, nil
}

// checks for membership of string element
func (b *BigBloom) ExistsStr(s string) (bool, float64) {
	bs := []byte(s)
	return b.ExistsBytes(bs)
}


// checks for membership of bytes element
func (b *BigBloom) ExistsBytes(bs []byte) (bool, float64) {
	var nonce int
	var h [32]byte
	for i := 0; i < b.len; i++ {
		if i % 32 == 0 {
			// new hash unique has to constructed after 256 bits
			bsNonce := append(bs, byte(nonce))
			h = sha256.Sum256(bsNonce)
			nonce++
		}
		if (b.bs[i] | h[i % 32]) != b.bs[i] {
			// bloom OR hash changes bloom which means there are 1's present in hash not in bloom
			return false, 1
		}
	}
	return true, b.Accuracy()
}

// get false positive rate
func (b *BigBloom) Accuracy() float64 {
	if b.n == 0 {
		return 1
	}
	return falsePositiveRate(b.len, b.n)
}

func (b *BigBloom) String() string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("%d-bit bloom filter: %d entries", 8*b.len, b.n))
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
// -------------------------------------
//

// func NewBigBloomCap(bits, cap int) *BigBloom {
// 	bytes := int(math.Ceil(float64(bits)/8))
// 	return &BigBloom{
// 		n: 0,
// 		bs: make([]byte, bytes),
// 		bits: bits,
// 		maxFalsePositiveRate: nil,  // don't care about accuracy unless specified
// 		cap: &cap,        			// don't care about capacity unless specified
// 	}
// }

// func NewBigBloomAcc(bits int, maxFalsePositiveRate float64) *BigBloom {
// 	bytes := int(math.Ceil(float64(bits)/8))
// 	return &BigBloom{
// 		n: 0,
// 		bs: make([]byte, bytes),
// 		bits: bits,
// 		maxFalsePositiveRate: nil,  // don't care about accuracy unless specified
// 		cap: nil,        			// don't care about capacity unless specified
// 	}
// }

// func NewBigBloomConstain(cap int, maxFalsePositiveRate float64) *BigBloom {
// 	return &BigBloom{
// 		n: 0,
// 		bs: make([]byte, bits),
// 		bits: bits,
// 		maxFalsePositiveRate: nil,  // don't care about accuracy unless specified
// 		cap: nil,        			// don't care about capacity unless specified
// 	}
// }

