# Bloom Filter Package
**Read the docs [here](https://pkg.go.dev/github.com/nettijoe96/bloom)**

A [Bloom filter](https://dl.acm.org/doi/10.1145/362686.362692) is a space-efficient probabilistic data strucuture invented by Burton Bloom. They provide probabilistic set membership where elements can be checked if they exist in the bloom filter. There are no false negatives, but there are false positives that are dependent on size of filter, number of entries, and number of hashes. The math is described [here](https://brilliant.org/wiki/bloom-filter/). Inserting an element is: filter = hash(ele) OR filter. Checking for existance is: filter == hash(ele) OR filter. Bloom filters have many applications including cache filtering and P2P networks. For example, [cryptocurrency full nodes](https://github.com/bitcoin/bitcoin/blob/master/src/leveldb/util/bloom.cc) use bloom filters to determine if they need to recieve a transaction from a peer.

## Example
`$ go get "github.com/nettijoe96/bloom"`
```
package main

import (
	"fmt"

	"github.com/nettijoe96/bloom"
)

func main() {
	b := bloom.NewBloom()
	b.PutStr("hello")
	exists, acc := b.ExistsStr("hello")
	fmt.Printf("'hello' exists: %t, accuracy of result: %f\n", exists, acc)
}
```

## Future Improvements Needed
1. Applying many hashes $k$ according to formula $k=\text{ln}(2)\times(\frac{m}{n})$
2. Parallelism in BigBloom. Each hash can be calculated and applied to filter in it's own go routine.
3. Possibly merge Bloom and BigBloom into one type
