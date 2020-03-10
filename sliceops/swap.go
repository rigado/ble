package sliceops

func SwapBuf(in []byte) []byte {
	a := make([]byte, 0, len(in))
	a = append(a, in...)
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}
