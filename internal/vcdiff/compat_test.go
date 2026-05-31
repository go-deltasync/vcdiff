//go:build compat

// Cross-implementation interoperability tests against Google's open-vcdiff
// `vcdiff` CLI. Run with: go test -tags=compat ./internal/vcdiff/...
// The whole file is skipped if `vcdiff` is not on PATH.
//
// open-vcdiff is invoked in its default mode, which emits the standard RFC 3284
// layout (separate data/instructions/addresses sections, no secondary
// compression) — exactly the wire subset this package targets. We deliberately
// avoid -interleaved and -checksum, which select open-vcdiff's format
// extensions.
package vcdiff

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireVcdiff(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("vcdiff"); err != nil {
		t.Skip("open-vcdiff `vcdiff` not found on PATH; skipping cross-impl compat")
	}
}

func runVcdiff(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("vcdiff", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("vcdiff %v failed: %v\n%s", args, err, out)
	}
}

// makeSourceAndTarget writes a dictionary (source) and a derived, edited target
// to dir and returns their paths and contents.
func makeSourceAndTarget(t *testing.T, dir string) (srcPath, tgtPath string, source, target []byte) {
	t.Helper()
	source = bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog.\n"), 600)
	target = append([]byte{}, source[:12_000]...)
	target = append(target, []byte("--cross-impl-inserted-bytes--")...)
	target = append(target, source[15_000:]...)

	srcPath = filepath.Join(dir, "source.bin")
	tgtPath = filepath.Join(dir, "target.bin")
	if err := os.WriteFile(srcPath, source, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tgtPath, target, 0o644); err != nil {
		t.Fatal(err)
	}
	return
}

// TestGoEncodeCDecode: Go computes the delta, open-vcdiff applies it. Exercises
// the C decoder reading a delta our encoder produced.
func TestGoEncodeCDecode(t *testing.T) {
	requireVcdiff(t)
	dir := t.TempDir()
	srcPath, _, source, target := makeSourceAndTarget(t, dir)

	deltaPath := filepath.Join(dir, "go.vcdiff")
	df, err := os.Create(deltaPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := Encode(source, target, df); err != nil {
		t.Fatal(err)
	}
	df.Close()

	outPath := filepath.Join(dir, "c.out")
	runVcdiff(t, "decode", "-dictionary", srcPath, "-delta", deltaPath, "-target", outPath)

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, target) {
		t.Fatal("Go-encode / C-decode reconstruction mismatch")
	}
}

// TestCEncodeGoDecode: open-vcdiff computes the delta, Go applies it. Exercises
// our decoder reading open-vcdiff's standard output.
func TestCEncodeGoDecode(t *testing.T) {
	requireVcdiff(t)
	dir := t.TempDir()
	srcPath, tgtPath, source, target := makeSourceAndTarget(t, dir)

	deltaPath := filepath.Join(dir, "c.vcdiff")
	runVcdiff(t, "encode", "-dictionary", srcPath, "-target", tgtPath, "-delta", deltaPath)

	delta, err := os.Open(deltaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer delta.Close()

	var out bytes.Buffer
	if _, err := Decode(source, delta, &out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(out.Bytes(), target) {
		t.Fatal("C-encode / Go-decode reconstruction mismatch")
	}
}
