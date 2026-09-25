package art

// sextant returns the character for a 2x3 dot pattern. Bit i is dot i+1,
// with dots numbered 1 2 / 3 4 / 5 6 from the top left. Unicode's "Block
// Sextant" characters (U+1FB00 to U+1FB3B) cover every pattern except four
// that already existed: empty, left half, right half and full.
func sextant(n int) rune {
	switch n {
	case 0:
		return ' '
	case 0b010101:
		return '▌'
	case 0b101010:
		return '▐'
	case 0b111111:
		return '█'
	}
	r := rune(0x1FB00 + n - 1)
	if n > 0b010101 {
		r-- // the left half has no sextant
	}
	if n > 0b101010 {
		r-- // nor has the right half
	}
	return r
}
