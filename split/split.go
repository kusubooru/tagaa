package split

// Bytes splits a slice of bytes b into subslices every n-th byte and returns a
// slice of those subslices.
func Bytes(b []byte, n int) [][]byte {
	var out [][]byte
	if len(b) == 0 {
		return make([][]byte, 0)
	}
	if n <= 0 {
		n = 1
	}
	if n >= len(b) {
		out = append(out, append(b))
		return out
	}
	rows := len(b) / n
	if len(b)%n != 0 {
		rows++
	}
	a := 0
	z := a + n
	for i := 0; i < rows; i++ {
		if z > len(b)-1 {
			out = append(out, append(b[a:]))
			break
		}
		out = append(out, append(b[a:z]))
		a = z
		z = a + n
	}
	return out
}
