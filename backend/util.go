package backend

var hexDigit = []byte("0123456789abcdef")

func picku32(a, b uint32) uint32 {
	if a > 0 {
		return a
	}
	return b
}

func picku64(a, b uint64) uint64 {
	if a > 0 {
		return a
	}
	return b
}

func pickStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func stringRev(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
