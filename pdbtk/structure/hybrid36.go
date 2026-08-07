package structure

import (
	"fmt"
	"strconv"
	"strings"
)

// Hybrid-36 is the wwPDB convention for encoding integers too large for a
// fixed-width PDB column: a width-w field counts up in decimal, then switches
// to base-36 with uppercase letters, then to base-36 with lowercase letters.
// See http://cci.lbl.gov/hybrid_36/
//
// For width 5: 1..99999 decimal, A0000..ZZZZZ is 100000..43770015,
// a0000..zzzzz is 43770016..87440031.

const h36Upper = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const h36Lower = "0123456789abcdefghijklmnopqrstuvwxyz"

func pow(base, n int) int {
	v := 1
	for i := 0; i < n; i++ {
		v *= base
	}
	return v
}

func base36(s, digits string) (int, bool) {
	v := 0
	for i := 0; i < len(s); i++ {
		idx := strings.IndexByte(digits, s[i])
		if idx < 0 {
			return 0, false
		}
		v = v*36 + idx
	}
	return v, true
}

// decodeHybrid36 parses a PDB numeric field, accepting both plain decimal
// (including negative values) and hybrid-36 encoded values.
func decodeHybrid36(s string, width int) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty field")
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n, nil
	}
	if len(s) != width || !isAlpha(s[0]) {
		return 0, fmt.Errorf("%q is neither a decimal nor a hybrid-36 value", s)
	}
	offset := pow(10, width) - 10*pow(36, width-1)
	if v, ok := base36(s, h36Upper); ok {
		return v + offset, nil
	}
	if v, ok := base36(s, h36Lower); ok {
		return v + offset + 26*pow(36, width-1), nil
	}
	return 0, fmt.Errorf("%q is neither a decimal nor a hybrid-36 value", s)
}

func encodeBase36(v int, width int, digits string) string {
	if v == 0 {
		return strings.Repeat("0", width)
	}
	var b []byte
	for v > 0 {
		b = append([]byte{digits[v%36]}, b...)
		v /= 36
	}
	for len(b) < width {
		b = append([]byte{'0'}, b...)
	}
	return string(b)
}

// encodeHybrid36 renders v into a field of the given width, falling back to
// hybrid-36 once the value no longer fits in decimal. The second return value
// reports whether v is representable at all.
func encodeHybrid36(v, width int) (string, bool) {
	if v >= 0 && v < pow(10, width) {
		return fmt.Sprintf("%*d", width, v), true
	}
	// Negative values keep the plain decimal form as long as they fit.
	if v < 0 && -v < pow(10, width-1) {
		return fmt.Sprintf("%*d", width, v), true
	}
	offset := pow(10, width) - 10*pow(36, width-1)
	if u := v - offset; u >= 0 && u < pow(36, width) {
		return encodeBase36(u, width, h36Upper), true
	}
	if l := v - offset - 26*pow(36, width-1); l >= 0 && l < pow(36, width) {
		return encodeBase36(l, width, h36Lower), true
	}
	return "", false
}
