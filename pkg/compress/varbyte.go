package compress

func AppendUvarint(dst []byte, v uint64) []byte {
	for v >= 0x80 {
		dst = append(dst, byte(v)&0x7f)
		v >>= 7
	}
	return append(dst, byte(v)|0x80)
}

func EncodeGaps(positions []int) []byte {
	out := make([]byte, 0, len(positions)+1)
	out = AppendUvarint(out, uint64(len(positions)))
	prev := 0
	for _, p := range positions {
		out = AppendUvarint(out, uint64(p-prev))
		prev = p
	}
	return out
}

func DecodeGaps(data []byte) []int {
	n, i := readUvarint(data, 0)
	positions := make([]int, 0, n)
	prev := 0
	for k := uint64(0); k < n; k++ {
		var gap uint64
		gap, i = readUvarint(data, i)
		prev += int(gap)
		positions = append(positions, prev)
	}
	return positions
}

func readUvarint(data []byte, i int) (uint64, int) {
	var v uint64
	var shift uint
	for i < len(data) {
		b := data[i]
		i++
		v |= uint64(b&0x7f) << shift
		if b&0x80 != 0 {
			return v, i
		}
		shift += 7
	}
	return v, i
}
