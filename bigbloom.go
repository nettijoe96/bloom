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

// BigBloom is a bloom filter with a variable length that uses SHA256 hashing with a nonce.
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

// Constructs len-byte bloom filter from k.
func NewBigBloomFromK(len, k int) (*BigBloom, error) {
	if k < 1 {
		return nil, errors.New("k cannot be less than 1")
	}
	return &BigBloom{
		n:                    0,
		k:                    k,
		bs:                   make([]byte, len),
		len:                  len,
		maxFalsePositiveRate: nil,
		cap:                  nil,
		isLoaded:             false,
	}, nil
}

// Constructs len-byte bloom filter from capacity
func NewBigBloomFromCap(len, cap int) (*BigBloom, error) {
	if cap < 1 {
		return nil, errors.New("capacity cannot be less than 1")
	}
	return &BigBloom{
		n:                    0,
		k:                    calcKFromCap(len, cap),
		bs:                   make([]byte, len),
		len:                  len,
		maxFalsePositiveRate: nil,
		cap:                  nil,
		isLoaded:             false,
	}, nil
}

// Constructs len-byte bloom filter from maxFalsePositiveRate
func NewBigBloomFromAcc(len int, maxFalsePositiveRate float64) (*BigBloom, error) {
	if maxFalsePositiveRate <= 0 || maxFalsePositiveRate >= 1 {
		return nil, errors.New("false positive rate must be between 0 and 1")
	}
	return &BigBloom{
		n:                    0,
		k:                    calcKFromAcc(len, maxFalsePositiveRate),
		bs:                   make([]byte, len),
		len:                  len,
		maxFalsePositiveRate: nil,
		cap:                  nil,
		isLoaded:             false,
	}, nil
}

// Constructs bloom filter with cap and maxFalsePositiveRate
func NewBigBloomAlloc(cap int, maxFalsePositiveRate float64) (*BigBloom, error) {
	if cap < 1 {
		return nil, errors.New("capacity cannot be less than 1")
	}
	if maxFalsePositiveRate <= 0 || maxFalsePositiveRate >= 1 {
		return nil, errors.New("false positive rate must be between 0 and 1")
	}

	// math:
	// eq1: k = ln(2) * m/n
	// eq2: acc = (1 - (1 - e^(-kn/m))^k
	// substitute k from eq1 into eq2 ...
	// acc = (.5)^(ln(2) * m/n)
	// log0.5(acc) = ln(2) * m/n
	// m = (n * log0.5(acc))/ln(2)
	// change of base ...
	// m = (n * ln(acc)) / (ln(0.5) * ln(2))
	numerator := float64(cap) * math.Log(maxFalsePositiveRate)
	denom := math.Log(.5) * math.Log(2)
	mFloat := numerator / denom
	len := int(math.Ceil(mFloat / 8))
	// calculate k using m
	k := calcKFromCap(len, cap)

	return &BigBloom{
		n:                    0,
		k:                    k,
		bs:                   make([]byte, len),
		len:                  len,
		maxFalsePositiveRate: &maxFalsePositiveRate,
		cap:                  &cap,
		isLoaded:             false,
	}, nil

}

// Load bloom filter from bytes of bloom filter and k
// This is useful for loading in a Bloom filter over the wire.
// This mechanism will disable accuracy calculations because n is unknown
func FromBytes(bs []byte, k int) (*BigBloom, error) {
	if k < 1 {
		return nil, errors.New("k cannot be less than 1")
	}
	return &BigBloom{
		n:                    0,
		k:                    k,
		bs:                   bs,
		len:                  len(bs),
		maxFalsePositiveRate: nil,
		cap:                  nil,
		isLoaded:             true,
	}, nil
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
		if falsePositiveRate(b.len, b.n+1, b.k) > *b.maxFalsePositiveRate {
			return b, &AccuracyError{acc: *b.maxFalsePositiveRate}
		}
	}

	totBits := len(b.bs) * 8
	for i := 0; i < b.k; i++ {
		// a single change in bs makes the whole SHA hash change, so an appended nonce is suitable
		bsNonce := append(bs, byte(i))
		var h [32]byte = sha256.Sum256(bsNonce)
		// get a random uint64 number
		bytes := h[0:8]
		// find index of bit
		bitI := binary.BigEndian.Uint64(bytes) % uint64(totBits)
		// find index of byte
		byteI := int(math.Floor(float64(bitI) / float64(8)))
		// find index of bit within byte
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
func (b *BigBloom) ExistsStr(s string) (bool, float64) {
	bs := []byte(s)
	return b.ExistsBytes(bs)
}

// Checks for existance of bytes element in a bloom filter. Returns boolean and false positive rate.
func (b *BigBloom) ExistsBytes(bs []byte) (bool, float64) {

	totBits := len(b.bs) * 8
	for i := 0; i < b.k; i++ {
		// a single change in bs makes the whole SHA hash change, so an appended nonce is suitable
		bsNonce := append(bs, byte(i))
		var h [32]byte = sha256.Sum256(bsNonce)
		// get a random uint64 number
		bytes := h[0:8]
		// find index of bit
		bitI := binary.BigEndian.Uint64(bytes) % uint64(totBits)
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
// -1 means cannot be calcuated because it is loaded in
func (b *BigBloom) Accuracy() float64 {
	if b.isLoaded {
		return -1
	}
	if b.n == 0 {
		return 1
	}
	return falsePositiveRate(b.len, b.n, b.k)
}

// Constrains bloom from not adding more than cap insertions
func (b *BigBloom) AddCapacityConstraint(cap int) error {
	if b.isLoaded {
		return errors.New("cannot add constraints to loaded bloom filters")
	}
	if cap < 1 {
		return errors.New("capacity cannot be less than 1")
	}
	if b.maxFalsePositiveRate != nil {
		// check if contraints capacity and maxFalsePositiveRate are compatible together with this size bloom filter
		if !constraintsCompatible(b.len, cap, b.k, *b.maxFalsePositiveRate) {
			return errors.New("false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")
		}
	}
	b.cap = &cap
	return nil
}

// Constrains bloom from not adding more insertions that cause accuracy to be worse than maxFalsePositiveRate
func (b *BigBloom) AddAccuracyConstraint(maxFalsePositiveRate float64) error {
	if b.isLoaded {
		return errors.New("cannot add constraints to loaded bloom filters")
	}
	if maxFalsePositiveRate <= 0 || maxFalsePositiveRate >= 1 {
		return errors.New("false positive rate must be between 0 and 1")
	}
	if b.cap != nil {
		// check if contraints capacity and maxFalsePositiveRate are compatible together with this size bloom filter
		if !constraintsCompatible(b.len, *b.cap, b.k, maxFalsePositiveRate) {
			return errors.New("false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")
		}
	}
	b.maxFalsePositiveRate = &maxFalsePositiveRate
	return nil
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

// converts bytes of bloom filter to hex string
func (b *BigBloom) Hex() string {
	return hex.EncodeToString(b.bs)
}
