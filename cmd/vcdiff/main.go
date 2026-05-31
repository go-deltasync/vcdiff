// vcdiff is a pure-Go, open-vcdiff-interoperable implementation of the VCDIFF
// (RFC 3284) delta format:
//
//	vcdiff encode [-s SOURCE] TARGET DELTA
//	vcdiff decode [-s SOURCE] DELTA  OUTPUT
//
// A single dash ("-") may be used for the streamable input/output of each
// command. The optional source (dictionary) is always read fully into memory
// because COPY instructions reference it at arbitrary offsets.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/go-deltasync/vcdiff/internal/vcdiff"
	"github.com/spf13/cobra"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:           "vcdiff",
		Short:         "Pure-Go, open-vcdiff-compatible VCDIFF (RFC 3284) encode/decode",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(encodeCmd(), decodeCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "vcdiff: %v\n", err)
		os.Exit(1)
	}
}

func encodeCmd() *cobra.Command {
	var sourcePath string
	cmd := &cobra.Command{
		Use:   "encode [flags] TARGET DELTA",
		Short: "Compute the VCDIFF delta from SOURCE to TARGET",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			source, err := readSource(sourcePath)
			if err != nil {
				return err
			}
			target, err := readAll(args[0])
			if err != nil {
				return err
			}
			out, closeOut, err := createOutput(args[1])
			if err != nil {
				return err
			}
			defer closeOut()
			return vcdiff.Encode(source, target, out)
		},
	}
	cmd.Flags().StringVarP(&sourcePath, "source", "s", "", "source (dictionary) file; empty for self-referential compression")
	return cmd
}

func decodeCmd() *cobra.Command {
	var sourcePath string
	cmd := &cobra.Command{
		Use:   "decode [flags] DELTA OUTPUT",
		Short: "Apply a VCDIFF delta to SOURCE, producing OUTPUT",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			source, err := readSource(sourcePath)
			if err != nil {
				return err
			}
			delta, closeDelta, err := openInput(args[0])
			if err != nil {
				return err
			}
			defer closeDelta()
			out, closeOut, err := createOutput(args[1])
			if err != nil {
				return err
			}
			defer closeOut()
			_, err = vcdiff.Decode(source, delta, out)
			return err
		},
	}
	cmd.Flags().StringVarP(&sourcePath, "source", "s", "", "source (dictionary) file; empty if the delta is self-contained")
	return cmd
}

// readSource reads the optional source file fully; an empty path yields nil.
func readSource(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	return readAll(path)
}

// readAll reads path fully ("-" => stdin).
func readAll(path string) ([]byte, error) {
	if path == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return b, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return b, nil
}

// openInput returns a reader for path ("-" => stdin) and a close function.
func openInput(path string) (io.Reader, func(), error) {
	if path == "-" {
		return os.Stdin, func() {}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	return f, func() { _ = f.Close() }, nil
}

// createOutput returns a writer for path ("-" => stdout) and a close function.
func createOutput(path string) (io.Writer, func(), error) {
	if path == "-" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("create %s: %w", path, err)
	}
	return f, func() { _ = f.Close() }, nil
}
