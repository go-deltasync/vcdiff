package vcdiff

import (
	"encoding/binary"
	"io"
)

const (
	minMatch     = 4  // shortest COPY the encoder will emit (and the seed length)
	runThreshold = 4  // shortest byte run emitted as a RUN instead of an ADD
	maxChain     = 32 // cap on hash-chain candidates inspected per position
)

// EncodeBytes returns a VCDIFF delta that reconstructs target from source.
// A nil/empty source produces a self-referential ("pure compression") delta.
func EncodeBytes(source, target []byte) []byte {
	out := appendHeader(nil)
	if len(target) > 0 {
		out = append(out, encodeWindow(source, target)...)
	}
	return out
}

// Encode writes EncodeBytes(source, target) to out.
func Encode(source, target []byte, out io.Writer) error {
	_, err := out.Write(EncodeBytes(source, target))
	return err
}

// encodeWindow builds a single VCDIFF window covering the whole target. The
// address space U = source ‖ target is searched with a greedy longest-match
// hash index; matches become COPY instructions, byte runs become RUN, and the
// rest becomes ADD literals.
func encodeWindow(source, target []byte) []byte {
	u := make([]byte, 0, len(source)+len(target))
	u = append(u, source...)
	u = append(u, target...)
	sLen := len(source)
	n := len(u)

	index := make(map[uint32][]int)
	indexAt := func(pos int) {
		if pos+minMatch <= n {
			h := binary.LittleEndian.Uint32(u[pos : pos+minMatch])
			index[h] = append(index[h], pos)
		}
	}
	for p := 0; p < sLen; p++ {
		indexAt(p)
	}

	cache := newAddressCache(defaultNearCache, defaultSameCache)
	var dataS, instS, addrS []byte

	litStart := sLen
	flush := func(end int) {
		if end > litStart {
			instS = append(instS, opcodeADD0)
			instS = appendVarint(instS, uint64(end-litStart))
			dataS = append(dataS, u[litStart:end]...)
		}
	}

	i := sLen
	for i < n {
		if mLen, mPos := findMatch(u, index, i, n); mLen >= minMatch {
			flush(i)
			enc := cache.encode(mPos, i)
			instS = append(instS, opcodeCOPY0(enc.mode))
			instS = appendVarint(instS, uint64(mLen))
			addrS = appendVarint(addrS, enc.val)
			for q := i; q < i+mLen; q++ {
				indexAt(q)
			}
			i += mLen
			litStart = i
			continue
		}
		if r := runLength(u, i, n); r >= runThreshold {
			flush(i)
			instS = append(instS, opcodeRUN0)
			instS = appendVarint(instS, uint64(r))
			dataS = append(dataS, u[i])
			for q := i; q < i+r; q++ {
				indexAt(q)
			}
			i += r
			litStart = i
			continue
		}
		indexAt(i)
		i++
	}
	flush(n)

	return assembleWindow(sLen, len(target), dataS, instS, addrS)
}

// findMatch returns the longest match (length, U-position) for the seed at i,
// or (0, -1) if none reaches minMatch's seed length. Candidates are inspected
// newest-first, capped at maxChain.
func findMatch(u []byte, index map[uint32][]int, i, n int) (int, int) {
	if i+minMatch > n {
		return 0, -1
	}
	chain := index[binary.LittleEndian.Uint32(u[i:i+minMatch])]
	bestLen, bestPos := 0, -1
	for c := len(chain) - 1; c >= 0 && len(chain)-1-c < maxChain; c-- {
		if l := matchLen(u, chain[c], i, n); l > bestLen {
			bestLen, bestPos = l, chain[c]
		}
	}
	return bestLen, bestPos
}

// matchLen returns how many bytes of u match starting at p versus i (p < i),
// stopping at the end of u. Forward overlap is intentional.
func matchLen(u []byte, p, i, n int) int {
	l := 0
	for i+l < n && u[p+l] == u[i+l] {
		l++
	}
	return l
}

// runLength returns the count of bytes equal to u[i] starting at i.
func runLength(u []byte, i, n int) int {
	r := 1
	for i+r < n && u[i+r] == u[i] {
		r++
	}
	return r
}

// assembleWindow serializes a VCD_SOURCE (or sourceless) window from the three
// prepared sections.
func assembleWindow(sLen, targetLen int, dataS, instS, addrS []byte) []byte {
	// The delta-encoding body: target-window size, Delta_Indicator, the three
	// section lengths, then the three sections.
	body := appendVarint(nil, uint64(targetLen))
	body = append(body, 0x00) // Delta_Indicator: no per-section compression
	body = appendVarint(body, uint64(len(dataS)))
	body = appendVarint(body, uint64(len(instS)))
	body = appendVarint(body, uint64(len(addrS)))
	body = append(body, dataS...)
	body = append(body, instS...)
	body = append(body, addrS...)

	var win []byte
	if sLen > 0 {
		win = append(win, winSource)
		win = appendVarint(win, uint64(sLen)) // source segment length
		win = appendVarint(win, 0)            // source segment position
	} else {
		win = append(win, 0x00)
	}
	win = appendVarint(win, uint64(len(body))) // length of the delta encoding
	win = append(win, body...)
	return win
}
