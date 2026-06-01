// Package vcdiff is a pure-Go, cgo-free, xdelta3-interoperable implementation of
// the VCDIFF (RFC 3284) delta format. It encodes a delta from a source
// ("dictionary") to a target and decodes it back, using the standard wire
// layout (default code table, separate sections, no secondary compression) so
// deltas interoperate with xdelta3 and other RFC 3284 tools.
//
//	delta := vcdiff.EncodeBytes(source, target)
//	var out bytes.Buffer
//	_, _ = vcdiff.Decode(source, bytes.NewReader(delta), &out) // out == target
//
// For batch workloads, reuse an Encoder to amortize buffer allocation:
//
//	enc := vcdiff.NewEncoder()
//	for _, p := range pairs {
//		delta := enc.EncodeBytes(p.source, p.target)
//		// ...
//	}
package vcdiff

import (
	"io"

	impl "github.com/go-deltasync/vcdiff/internal/vcdiff"
)

// Encoder is a reusable VCDIFF encoder. Reusing one across many Encode calls in
// a batch or server workload avoids re-allocating its internal buffers on every
// call. An Encoder must not be used concurrently.
type Encoder = impl.Encoder

// NewEncoder returns a reusable Encoder.
func NewEncoder() *Encoder { return impl.NewEncoder() }

// Encode writes a VCDIFF delta that reconstructs target from source to out.
// A nil or empty source produces a self-referential ("pure compression") delta.
func Encode(source, target []byte, out io.Writer) error {
	return impl.Encode(source, target, out)
}

// EncodeBytes returns a VCDIFF delta that reconstructs target from source.
// A nil or empty source produces a self-referential ("pure compression") delta.
func EncodeBytes(source, target []byte) []byte {
	return impl.EncodeBytes(source, target)
}

// Decode applies a VCDIFF delta to source, writes the reconstructed target to
// out, and returns the number of target bytes written. The delta may use any
// standard RFC 3284 construct, so deltas produced by xdelta3 are accepted.
func Decode(source []byte, delta io.Reader, out io.Writer) (int64, error) {
	return impl.Decode(source, delta, out)
}
