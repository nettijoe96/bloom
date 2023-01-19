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

	// add constaints to bloom filter
	AddAccuracyConstraint(float64) error
	AddCapacityConstraint(int) error
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

// Constructs len-byte bloom filter from k.
func NewBloomFromK(k int) (*BigBloom, error) {
	if k < 1 {
		return nil, errors.New("k cannot be less than 1")
	}
	return &BigBloom{
		n:                    0,
		k:                    k,
		bs:                   make([]byte, 64),
		len:                  64,
		maxFalsePositiveRate: nil,
		cap:                  nil,
		isLoaded:             false,
	}, nil
}

// Constructs len-byte bloom filter from capacity
func NewBloomFromCap(cap int) (*BigBloom, error) {
	if cap < 1 {
		return nil, errors.New("capacity cannot be less than 1")
	}
	return &BigBloom{
		n:                    0,
		k:                    calcKFromCap(64, cap),
		bs:                   make([]byte, 64),
		len:                  64,
		maxFalsePositiveRate: nil,
		cap:                  nil,
		isLoaded:             false,
	}, nil
}

// Constructs len-byte bloom filter from maxFalsePositiveRate
func NewBloomFromAcc(maxFalsePositiveRate float64) (*BigBloom, error) {
	if maxFalsePositiveRate <= 0 || maxFalsePositiveRate >= 1 {
		return nil, errors.New("false positive rate must be between 0 and 1")
	}
	return &BigBloom{
		n:                    0,
		k:                    calcKFromAcc(64, maxFalsePositiveRate),
		bs:                   make([]byte, 64),
		len:                  64,
		maxFalsePositiveRate: nil,
		cap:                  nil,
		isLoaded:             false,
	}, nil
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

// constains bloom from not adding more than cap insertions
func (b *Bloom) AddCapacityConstraint(cap int) error {
	if cap < 1 {
		return errors.New("capacity cannot be less than 1")
	}
	if b.cap != nil {
		// check if contraints capacity and maxFalsePositiveRate are compatible together with this size bloom filter
		if !constraintsCompatible(b.len, cap, b.k, *b.maxFalsePositiveRate) {
			return errors.New("false positive rate will be higher at full capacity than the maxFalsePositiveRate provided")
		}
	}
	b.cap = &cap
	return nil
}

// constains bloom from not adding more insertions that cause accuracy to be worse than maxFalsePositiveRate
func (b *Bloom) AddAccuracyConstraint(maxFalsePositiveRate float64) error {
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

// used when both constraints are set to check compatability
func constraintsCompatible(len, cap, k int, allowedMaxFalsePositiveRate float64) bool {
	// check if contraints capacity and maxFalsePositiveRate are compatible together with this size bloom filter
	calcMaxFalsePositiveRate := falsePositiveRate(len, cap, k)
	if calcMaxFalsePositiveRate > allowedMaxFalsePositiveRate {
		// if the maximum calculated false positive rate is greater than the inputed allowed false positive rate, fail.
		// this is more readable than returning calcMaxFalsePositiveRate <= allowedMaxFalsePositiveRate
		return false
	}
	return true
}

// calculate k from len of filter and capacity
func calcKFromCap(len, n int) int {
	m := len * 8
	// k = ln(2) * m/n
	kFloat := math.Log(2) * float64(m) / float64(n)
	// round to nearest int
	var k int
	if kFloat < 1 {
		k = 1
	} else {
		k = int(math.Round(kFloat))
	}
	return k
}

// calculate k from len of filter and accuracy
func calcKFromAcc(len int, acc float64) int {
	m := len * 8

	// eq1: k = ln(2) * m/n
	// rearranged: n = ln(2) * m/k
	// eq2: acc = (1 - (1 - 1/m)^-kn)^k
	// replace n in eq2 with e1 ...
	// acc = (1 - (1 - 1/m)^(-ln(2)m))^k
	// let base2 = 1 - (1 - 1/m)^(-ln(2)m)
	// solve for k ...
	// k = logbase2(acc)
	// change of base ...
	// k = ln(acc) / ln(base2)
	// expand b ...
	// k = ln(acc) / ln( 1-(1 - 1/m)^(-ln(2)m) )
	base1 := 1 - float64(1)/float64(m)
	expCalc := math.Pow(base1, math.Log(2)*float64(m))
	base2 := 1 - expCalc
	kFloat := math.Log(acc) / math.Log(base2)
	var k int
	if kFloat < 1 {
		// k can't be less than 1
		k = 1
	} else {
		// k must be an int
		k = int(math.Round(kFloat))
	}
	return k
}
