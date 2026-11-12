// Package currency formats monetary amounts for display. Indonesian Rupiah is
// formatted with a dot as the thousands separator (Rp 1.250.000), matching the
// local convention.
package currency

import (
	"strconv"
	"strings"
)

// Format renders an amount with its currency. IDR uses dot-grouped thousands
// (Rp 485.000); any other currency falls back to a generic grouped form.
func Format(amount int64, code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	switch code {
	case "", "IDR":
		return "Rp " + group(amount, '.')
	default:
		return code + " " + group(amount, ',')
	}
}

// group inserts a separator every three digits.
func group(n int64, sep byte) string {
	neg := n < 0
	if neg {
		n = -n
	}
	digits := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(sep)
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
