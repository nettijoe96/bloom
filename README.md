# Bloom Filter Package
**Read the docs [here](https://pkg.go.dev/github.com/nettijoe96/bloom)**

A [Bloom filter](https://dl.acm.org/doi/10.1145/362686.362692) is a space-efficient probabilistic data strucuture invented by Burton Bloom. They provide probabilistic set membership where elements can be checked if they exist in the bloom filter. There are no false negatives, but there are false positives that are dependent on size of filter, number of entries, and number of hashes. The math is described [here](https://brilliant.org/wiki/bloom-filter/). Inserting an element is: filter = hash(ele) OR filter. Checking for existance is: filter == hash(ele) OR filter. Bloom filters have many applications including cache filtering and P2P networks. For example, SPV nodes [use](https://github.com/bitcoin/bitcoin/blob/master/src/leveldb/util/bloom.cc) bloom filters to help conceal their addresses. They do this by constructing a filter that will match transactions that are associated to their addresses and other transactions that are not associated to their addresses. Then they send this bloom filter to connected full nodes and recieve the transactions that match it. Full nodes can not confident in which transactions are related to addresses of the SPV node.

A [mock-up](https://github.com/nettijoe96/spv-bloom) of the SPV functionality using this package!

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
1. Apply $k$ hashes according to formula $k=\text{ln}(2)\times(\frac{m}{n})$
2. Possibly merge Bloom and BigBloom into one type
