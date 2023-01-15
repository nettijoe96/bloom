# Bloom Filter Package
[Bloom filters](https://dl.acm.org/doi/10.1145/362686.362692) are a space-efficient probabilistic data strucuture invented by Burton Bloom. They provide probabilistic set membership where elements can be checked if they exist in the bloom filter. There are no false negatives, but there are false positives that are dependent on size of filter, number of entries, and number of hashes. The math is described [here](https://brilliant.org/wiki/bloom-filter/). Inserting an element is: filter = hash(ele) OR filter. Checking for existance is: filter == hash(ele) OR filter. Bloom filters have many applications including cache filtering and P2P networks. For example, [cryptocurrency full nodes](https://github.com/bitcoin/bitcoin/blob/master/src/leveldb/util/bloom.cc) use bloom filters to determine if they need to recieve a transaction from a peer.
## Example
```
bloom := NewBloom()
bloom.PutStr("hello")
exists, acc := bloom.ExistsStr("hello")
fmt.Printf("'hello' exists: %t, chance false positive: %f\n", exists, acc)
```
## Bloom type

### NewBloom()
constructs 512-bit bloom filter with no constaints

### NewBloomConstrain(cap *int, maxFalsePositiveRate *float64)
constructs 512-bit bloom filter with constraints. If cap is provided, bloom filter will not allow for more that number of unique elements in bloom filter. If maxFalsePositiveRate is provided then the false positive rate of the filter will not be allowed to increase beyond that amount. 

## BigBloom type

### NewBigBloom(len int)
constructs len-byte bloom filter with no constraints.

### NewBigBloomAlloc(cap int, maxFalsePositiveRate float64)
constructs bloom filter with the minimum length to both satisify constraints. 

### NewBigBloomConstrain(len int, cap *int, maxFalsePositiveRate *float64)
Equivalent to NewBloomConstain. Constructs 512-bit bloom filter with constraints. If cap is provided, bloom filter will not allow for more that number of unique elements in bloom filter. If maxFalsePositiveRate is provided then the false positive rate of the filter will not be allowed to increase beyond that amount.

## future improvements needed
#### - Applying many hashes (k) according to formula k=ln(2)*(m/n)
#### - Parallelism in BigBloom. Each hash can be calculated and applied to filter in it's own go routine.