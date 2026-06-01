package vcdiff_test

import (
	"bytes"
	"testing"

	"github.com/go-deltasync/vcdiff"
)

// TestFacadeRoundTrip exercises the public API end to end: encode a delta from a
// source to a target and decode it back, via both the package-level functions
// and a reusable Encoder.
func TestFacadeRoundTrip(t *testing.T) {
	source := bytes.Repeat([]byte("the quick brown fox\n"), 200)
	target := append(append([]byte{}, source[:1500]...), append([]byte("EDIT"), source[1600:]...)...)

	// Package-level EncodeBytes + Decode.
	delta := vcdiff.EncodeBytes(source, target)
	var out bytes.Buffer
	n, err := vcdiff.Decode(source, bytes.NewReader(delta), &out)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if int(n) != len(target) || !bytes.Equal(out.Bytes(), target) {
		t.Fatal("package-level round-trip mismatch")
	}

	// Streaming Encode.
	var deltaBuf bytes.Buffer
	if err := vcdiff.Encode(source, target, &deltaBuf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if !bytes.Equal(deltaBuf.Bytes(), delta) {
		t.Fatal("Encode and EncodeBytes disagree")
	}

	// Reusable Encoder produces identical output.
	enc := vcdiff.NewEncoder()
	if got := enc.EncodeBytes(source, target); !bytes.Equal(got, delta) {
		t.Fatal("Encoder output differs from package-level EncodeBytes")
	}
}
