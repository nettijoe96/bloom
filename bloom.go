package bloom

type Bloomer interface {
	// put in bloom. Bool if successful
	putStr(string) (bool)
	putBytes([]byte) (bool)

	// checks for existance. Float is accuracy: (1 - probability of false positive).
	existsStr(string) (bool, float64);
	existsBytes([]byte) (bool, float64);

	// returns accuracy: (1 - probability of false positive)
	accuracy() float64;
}

type Bloom struct {
	// current number of entries
	n int
	
	// bloom filter bytes
	bs []byte

	// number of bytes. only 32 or 64 allowed
	size int

	// min_accuracy (default is 0)
	min_accuracy float64

	// optional, maximum number of entries allowed
	cap *int 
}