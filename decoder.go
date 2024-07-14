package bottledlightning

import (
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"sync"
)

// Inspired by [encoding/gob.Decoder] from the Go standard library, a Decoder
// specialises in the receipt of LMDB key-value records transmitted by an
// Encoder counterpart. It is safe for concurrent use by multiple goroutines.
type Decoder struct {
	reader io.Reader
	hasher hash.Hash32
	mutex  sync.Mutex
}

// NewDecoder returns a new Decoder that will receive from the [io.Reader], and
// optionally verify the checksum of every record if the [hash.Hash32] is not
// nil.
func NewDecoder(reader io.Reader, hasher hash.Hash32) (d *Decoder) {
	d = &Decoder{
		reader: reader,
		hasher: hasher,
	}

	return
}

// Decode receives the next record from the input stream and returns two byte
// slices containing the key and value, respectively.
//
// At the end of the stream, Decode returns a wrapped [io.EOF]. See [errors.Is]
// for more information on detecting wrapped errors.
func (d *Decoder) Decode() (key, val []byte, e error) {
	key, val, _, e = d.decode()

	return
}

// DecodeX is a variant of Decode that also interprets extended metadata.
func (d *Decoder) DecodeX() (key, val []byte, xmv xMetaValue, e error) {
	return d.decode()
}

func (d *Decoder) decode() (key, val []byte, xmv xMetaValue, e error) {
	defer errorf("could not decode record", &e)

	var (
		c bool // a trailing 32-bit checksum is present if true
		k int  // key length
		v int  // value length
		x int  // number of bytes representing value length
	)

	d.mutex.Lock()

	defer d.mutex.Unlock()

	x, c, xmv, k, e = d.readXCMK()
	if e != nil {
		return
	}

	v, e = d.readV(x)
	if e != nil {
		return
	}

	key, e = d.readKey(k)
	if e != nil {
		return
	}

	val, e = d.readVal(v)
	if e != nil {
		return
	}

	if !c {
		return
	}

	e = d.verifyChecksum(key, val)
	if e != nil {
		return
	}

	return
}

func (d *Decoder) readXCMK() (x int, c bool, m xMetaValue, k int, e error) {
	// Reads the first two bytes, expecting the following bit fields:
	//   * X: 2 bits to encode the value of x, so that 1 <= x <= 4 represents
	//     len(val),
	//   * C: 1 bit to indicate the presence of a trailing 32-bit checksum,
	//   * M: 4 bits for extended metadata, and
	//   * K: 9 bits to represent len(key).

	var (
		xcmk uint16
	)

	e = binary.Read(d.reader, binary.BigEndian, &xcmk)
	if e != nil {
		return
	}

	x = int(xcmk >> offsetX)

	if x == 0 {
		x = 4
	}

	c = (xcmk>>offsetC)&1 == 1

	m = xMetaValue(xcmk>>offsetM) & XMetaValueF

	k = int(xcmk & lmdbMaxKeyLen)

	return
}

func (d *Decoder) readV(x int) (v int, e error) {
	// Reads x bytes and returns the interpreted len(val).

	var (
		b = make([]byte, maxUintLen32)
	)

	_, e = d.reader.Read(b[maxUintLen32-x:])
	if e != nil {
		return
	}

	v = int(binary.BigEndian.Uint32(b))

	return
}

func (d *Decoder) readKey(k int) (key []byte, e error) {
	// Reads k bytes containing the uninterpreted key.

	key = make([]byte, k)

	_, e = d.reader.Read(key)
	if e != nil {
		return
	}

	return
}

func (d *Decoder) readVal(v int) (val []byte, e error) {
	// Reads v bytes containing the uninterpreted value.

	val = make([]byte, v)

	_, e = d.reader.Read(val)
	if e != nil {
		return
	}

	return
}

func (d *Decoder) verifyChecksum(key, val []byte) (e error) {
	// Reads and verifies a 32-bit checksum of the record if d.hasher is not
	// nil; discards four bytes otherwise.

	var (
		computed uint32
		observed uint32
	)

	if d.hasher == nil {
		_, e = io.CopyN(io.Discard, d.reader, maxUintLen32)

		return
	}

	e = binary.Read(d.reader, binary.BigEndian, &observed)
	if e != nil {
		return
	}

	defer d.hasher.Reset()

	_, e = d.hasher.Write(key)
	if e != nil {
		return
	}

	_, e = d.hasher.Write(val)
	if e != nil {
		return
	}

	computed = d.hasher.Sum32()

	if computed != observed {
		e = fmt.Errorf("computed checksum does not match observed")

		return
	}

	return
}
