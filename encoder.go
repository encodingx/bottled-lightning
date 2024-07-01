package bottledlightning

import (
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"sync"
)

// An Encoder is modelled after encoding/gob.Encoder from the Go standard
// library, but specialises in the transmission of LMDB key-value records.
//
// An LMDB record, consisting of a key no more than 511 bytes long, and a value
// of maximum size 4 GiB, is encoded as follows:
//
//   - 2 bytes to represent the key length k in number of bytes,
//   - 1 <= x <= 4 bytes to represent the value length v in number of bytes,
//   - k bytes to hold the uninterpreted key,
//   - v bytes to hold the uninterpreted value, and
//   - 4 bytes to hold an optional 32-bit checksum of the record.
//
// This incurs an overhead of 3 to 10 bytes per record, and leaves the first
// seven bits free to carry the following metadata:
//
//   - 2 bits to encode the value of x (from the second bullet point above),
//   - 1 bit to indicate the presence of a trailing 32-bit checksum, and
//   - 4 bits for extended metadata---see defined constants.
//
// Encoders are safe for concurrent use by multiple goroutines.
type Encoder struct {
	writer io.Writer
	hasher hash.Hash32
	mutex  sync.Mutex
}

// NewEncoder returns a new encoder that will transmit on the [io.Writer], and
// optionally append a 32-bit checksum to every record if the [hash.Hash32] is
// not nil.
func NewEncoder(writer io.Writer, hasher hash.Hash32) (n *Encoder) {
	n = &Encoder{
		writer: writer,
		hasher: hasher,
	}

	return
}

// Encode transmits a key-value record.
func (n *Encoder) Encode(key, val []byte) error {
	return n.encode(key, val, XMetaValue0)
}

// EncodeX transmits a key-value record with extended metadata.
func (n *Encoder) EncodeX(key, val []byte, xmv xMetaValue) error {
	return n.encode(key, val, xmv)
}

func (n *Encoder) encode(key, val []byte, xmv xMetaValue) (e error) {
	// Transmits a key-value record with extended metadata.

	e = n.validateLens(key, val)
	if e != nil {
		return
	}

	n.mutex.Lock()

	defer n.mutex.Unlock()

	e = n.writeXCMK(key, val, xmv)
	if e != nil {
		return
	}

	e = n.writeV(val)
	if e != nil {
		return
	}

	e = n.writeKey(key)
	if e != nil {
		return
	}

	e = n.writeVal(val)
	if e != nil {
		return
	}

	if n.hasher == nil {
		return
	}

	e = n.writeChecksum(key, val)
	if e != nil {
		return
	}

	return
}

func (n *Encoder) validateLens(key, val []byte) error {
	// Returns a descriptive error if either length of key or val exceeds the
	// respective thresholds set by LMDB, or nil otherwise.

	const (
		lmdbMaxValLen = 1 << 32
	)

	if len(key) > lmdbMaxKeyLen {
		return fmt.Errorf("could not encode record: " +
			"LMDB maximum key length (511 B) exceeded",
		)
	}

	if len(val) > lmdbMaxValLen {
		return fmt.Errorf("could not encode record: " +
			"LMDB maximum value length (4 GiB) exceeded",
		)
	}

	return nil
}

func (n *Encoder) writeXCMK(key, val []byte, xmv xMetaValue) (e error) {
	// Writes the first two bytes, consisting of the following bit fields:
	//   * X: 2 bits to encode the value of x, so that 1 <= x <= 4 represents
	//     len(val),
	//   * C: 1 bit to indicate the presence of a trailing 32-bit checksum,
	//   * M: 4 bits for extended metadata, and
	//   * K: 9 bits to represent len(key).
	//
	//  1           0
	//  5 4 3 2 1 0 9 8 7 6 5 4 3 2 1 0
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// | X |C|   M   |        K        |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

	var (
		x = uint16(findX(val)%4) << offsetX
		// 1: 0b01, 2: 0b10, 3: 0b11, 4: 0b00
		c = uint16(1) << offsetC
		m = uint16(xmv) << offsetM
		k = uint16(len(key))
	)

	if n.hasher == nil {
		c = 0
	}

	e = binary.Write(n.writer, binary.BigEndian, x|c|m|k)
	if e != nil {
		return
	}

	return
}

func (n *Encoder) writeV(val []byte) (e error) {
	// Writes one to four bytes representing len(val).

	var (
		b = make([]byte, maxUintLen32)
	)

	binary.BigEndian.PutUint32(b,
		uint32(len(val)),
	)

	_, e = n.writer.Write(b[maxUintLen32-findX(val):])
	if e != nil {
		return
	}

	return
}

func (n *Encoder) writeKey(key []byte) (e error) {
	// Writes the uninterpreted key.

	_, e = n.writer.Write(key)
	if e != nil {
		return
	}

	return
}

func (n *Encoder) writeVal(val []byte) (e error) {
	// Writes the uninterpreted value.

	_, e = n.writer.Write(val)
	if e != nil {
		return
	}

	return
}

func (n *Encoder) writeChecksum(key, val []byte) (e error) {
	// Writes a 32-bit checksum of the record.

	defer n.hasher.Reset()

	_, e = n.hasher.Write(key)
	if e != nil {
		return
	}

	_, e = n.hasher.Write(val)
	if e != nil {
		return
	}

	_, e = n.writer.Write(
		n.hasher.Sum([]byte{}),
	)
	if e != nil {
		return
	}

	return
}

func findX(s []byte) (x int) {
	// Returns the minimum number of bytes needed to encode an unsigned integer
	// indicating the length of byte slice s.

	var (
		l int = len(s)
	)

	switch {
	case l < 1<<8:
		return 1

	case l < 1<<16:
		return 2

	case l < 1<<24:
		return 3

	case l < 1<<32:
		return 4

	default:
		panic("byte slice s exceeds the maximum LMDB value size")
	}

	return
}
