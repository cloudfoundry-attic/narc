package narc

import (
	"encoding/binary"
)

func parseWindowChange(s []byte) (width, height uint16, ok bool) {
	width32, s, ok := parseUint32(s)
	if !ok {
		return
	}

	height32, _, ok := parseUint32(s)
	if !ok {
		return
	}

	width = uint16(width32)
	height = uint16(height32)
	if width < 1 {
		ok = false
	}
	if height < 1 {
		ok = false
	}

	return
}

func parsePtyRequest(s []byte) (width, height uint16, ok bool) {
	_, s, ok = parseString(s)
	if !ok {
		return
	}

	width32, s, ok := parseUint32(s)
	if !ok {
		return
	}

	height32, _, ok := parseUint32(s)
	if !ok {
		return
	}

	width = uint16(width32)
	height = uint16(height32)

	return width, height, width >= 1 && height >= 1
}

func parseString(in []byte) (out, rest []byte, ok bool) {
	if len(in) < 4 {
		return
	}

	length := binary.BigEndian.Uint32(in)
	if uint32(len(in)) < 4+length {
		return
	}

	out = in[4 : 4+length]
	rest = in[4+length:]

	ok = true

	return
}

func parseUint32(in []byte) (uint32, []byte, bool) {
	if len(in) < 4 {
		return 0, nil, false
	}

	return binary.BigEndian.Uint32(in), in[4:], true
}
