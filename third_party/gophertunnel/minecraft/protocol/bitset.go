package protocol

import "math/big"

type Bitset struct {
	size int
	int  *big.Int
}

func NewBitset(size int) Bitset {
	return Bitset{size: size, int: new(big.Int)}
}

func (b Bitset) Set(i int) {
	if i >= b.size {
		panic("index out of bounds")
	}
	b.int.SetBit(b.int, i, 1)
}

func (b Bitset) Unset(i int) {
	if i >= b.size {
		panic("index out of bounds")
	}
	b.int.SetBit(b.int, i, 0)
}

func (b Bitset) Load(i int) bool {
	if i >= b.size {
		panic("index out of bounds")
	}
	return b.int.Bit(i) == 1
}

func (b Bitset) Len() int {
	return b.size
}
