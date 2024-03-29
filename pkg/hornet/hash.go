package hornet

import (
	"fmt"

	"github.com/iotaledger/iota.go/consts"
	"github.com/iotaledger/iota.go/encoding/t5b1"
	"github.com/iotaledger/iota.go/trinary"
)

var (
	hashBytesSize = t5b1.EncodedLen(consts.HashTrinarySize)
	tagBytesSize  = t5b1.EncodedLen(consts.TagTrinarySize)
)

const (
	addressTrytesSize = hashTrytesSize + consts.AddressChecksumTrytesSize/consts.TritsPerTryte
	hashTrytesSize    = consts.HashTrytesSize
	tagTrytesSize     = consts.TagTrinarySize / consts.TritsPerTryte
	HashSize          = 49
)

// Hash is the binary representation of a trinary Hash.
type Hash []byte

// HashFromAddressTrytes returns the binary representation of the given address trytes.
// It panics when trytes hash invalid length.
func HashFromAddressTrytes(trytes trinary.Trytes) Hash {
	if len(trytes) != hashTrytesSize && len(trytes) != addressTrytesSize {
		panic("invalid address length")
	}

	return t5b1.EncodeTrytes(trytes[:hashTrytesSize])
}

// HashFromHashTrytes returns the binary representation of the given hash trytes.
// It panics when trytes hash invalid length.
func HashFromHashTrytes(trytes trinary.Trytes) Hash {
	if len(trytes) != hashTrytesSize {
		panic("invalid hash length")
	}

	return t5b1.EncodeTrytes(trytes)
}

// HashFromTagTrytes returns the binary representation of the given tag trytes.
// It panics when trytes hash invalid length.
func HashFromTagTrytes(trytes trinary.Trytes) Hash {
	if len(trytes) != tagTrytesSize {
		panic("invalid tag length")
	}

	return t5b1.EncodeTrytes(trytes)
}

// Trytes converts the binary Hash to its tryte representation.
// It panics when the binary encoding is invalid.
func (h Hash) Trytes() trinary.Trytes {
	switch len(h) {
	case hashBytesSize:
		return mustDecodeToTrytes(h)[:hashTrytesSize]
	case tagBytesSize:
		return mustDecodeToTrytes(h)[:tagTrytesSize]
	default:
		panic("invalid hash length")
	}
}

func mustDecodeToTrytes(src []byte) trinary.Trytes {
	dst, err := t5b1.DecodeToTrytes(src)
	if err != nil {
		panic(fmt.Sprintf("invalid hash bytes: %v", err))
	}

	return dst
}

// Hashes is a slice of Hash.
type Hashes []Hash
