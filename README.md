# vcdiff

A pure-Go, **xdelta3-interoperable** implementation of the VCDIFF
([RFC 3284](https://www.rfc-editor.org/rfc/rfc3284)) delta format. Part of the
[`go-deltasync`](https://github.com/go-deltasync) family of delta-sync tools.

```
encode  [-s SOURCE] TARGET DELTA   compute the delta from SOURCE to TARGET
decode  [-s SOURCE] DELTA  OUTPUT  apply the delta to SOURCE, producing OUTPUT
```

VCDIFF is the lingua franca of delta encoding — the `vcdiff` HTTP
content-encoding, Chrome's SDCH, and xdelta3 all speak it. The bytes this tool
emits use the standard wire layout (file version `0x00`, the RFC default code
table, separate data/instructions/addresses sections, no secondary
compression), so deltas are interchangeable with
[xdelta3](https://github.com/jmacd/xdelta) — the actively maintained reference
implementation. Cross-impl interop is verified both directions against
`xdelta3 -S` (secondary compression disabled) under `-tags=compat`, alongside a
comparative performance test (`go test -tags=compat -run Perf`) that reports
encode/decode throughput and delta size against xdelta3.

## Install

```sh
go install github.com/go-deltasync/vcdiff/cmd/vcdiff@latest
```

## Usage

```sh
vcdiff encode -s old.bin new.bin patch.vcdiff   # delta against a dictionary
vcdiff decode -s old.bin patch.vcdiff out.bin
cmp new.bin out.bin                             # identical

vcdiff encode new.bin patch.vcdiff              # no source: pure compression
vcdiff decode patch.vcdiff out.bin
```

`-` means stdin/stdout for the streamable TARGET/DELTA/OUTPUT argument. The
optional source (dictionary) is always read fully into memory because COPY
instructions reference it at arbitrary offsets.

## How it works

The encoder searches the address space `U = SOURCE ‖ TARGET` with a greedy
longest-match index (4-byte seed hash chains) and turns matches into COPY
instructions, repeated bytes into RUN, and everything else into ADD literals.
It emits a single window per call using only the always-standard SELF / HERE /
near address modes. A COPY that begins in the source window is kept within it
(matches that would cross into the target window are capped), so the delta works
with decoders — like xdelta3 — that read a source-window copy straight from the
source buffer; target-window self-references may still overlap freely.

The decoder is a full RFC 3284 reader: it handles every default-table opcode
(including the double ADD+COPY / COPY+ADD forms and size-baked opcodes), all
nine address modes (SELF, HERE, four near-cache, three same-cache), `VCD_SOURCE`
and `VCD_TARGET` windows, the application header, and the optional `VCD_CHECKSUM`
Adler-32. It rejects secondary compression (`VCD_DECOMPRESS`) and custom code
tables (`VCD_CODETABLE`) with explicit errors.

### Current limitations

- **Single window at encode time.** The encoder loads SOURCE and TARGET fully
  into memory and emits one window covering the whole target. (The decoder
  accepts arbitrary multi-window deltas.)
- **No secondary compression or custom code tables on output**, and no
  interleaved-format output (open-vcdiff's `0x53` extension). The decoder still
  reads the standard layout these tools produce by default.

## Performance

The encoder indexes the address space with zlib-style `head`/`prev` hash chains
(flat slices, no per-position allocation), capped at a fixed chain depth.

### Protocol

`TestPerfVsXdelta3` (under `-tags=compat`) builds the Go CLI and invokes it and
`xdelta3 -S` as **subprocesses on identical files** (apples-to-apples; each pays
process startup + file I/O). The input is an 8 MiB incompressible source with 32
scattered edits plus an insertion as the target, so COPY matching dominates.
Delta size is exact; wall-clock is the best of 3 runs. Reproduce with
`go test -tags=compat -v -run Perf ./internal/vcdiff/`.

### Results

Measured on an Apple M4 Max, Go 1.26, xdelta3 3.1.0:

| impl        | encode    | decode    | delta bytes |
|-------------|-----------|-----------|-------------|
| go-vcdiff   | 105 MB/s  | 382 MB/s  | 452         |
| xdelta3 -S  | 226 MB/s  | 628 MB/s  | 450         |

Delta size matches the reference to **2 bytes** (equivalent match quality);
throughput is within ~2× — reasonable for a pure-Go, cgo-free encoder versus
the C reference. Numbers are machine-dependent and indicative.

## License

BSD-3-Clause. See [LICENSE](LICENSE).
