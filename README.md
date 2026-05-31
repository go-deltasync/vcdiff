# vcdiff

A pure-Go, **open-vcdiff-interoperable** implementation of the VCDIFF
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
compression), so deltas are interchangeable with Google's
[open-vcdiff](https://github.com/google/open-vcdiff) `vcdiff` CLI.

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
near address modes.

The decoder is a full RFC 3284 reader: it handles every default-table opcode
(including the double ADD+COPY / COPY+ADD forms and size-baked opcodes), all
nine address modes (SELF, HERE, four near-cache, three same-cache), `VCD_SOURCE`
and `VCD_TARGET` windows, the application header, and the optional `VCD_CHECKSUM`
Adler-32. It rejects secondary compression (`VCD_DECOMPRESS`) and custom code
tables (`VCD_CODETABLE`) with explicit errors.

### Current limitations

- **Single window at encode time.** The encoder loads SOURCE and TARGET fully
  into memory and emits one window covering the whole target. (The decoder
  accepts arbitrary multi-window deltas, such as open-vcdiff's.)
- **No secondary compression or custom code tables on output**, and no
  interleaved-format output (open-vcdiff's `0x53` extension). The decoder still
  reads the standard layout these tools produce by default.

## License

BSD-3-Clause. See [LICENSE](LICENSE).
