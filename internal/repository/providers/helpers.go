package providers

import (
	"fmt"
	"strings"
)

// defaultCurrency returns the currency code, defaulting to IDR when the
// provider omits it.
func defaultCurrency(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "IDR"
	}
	return strings.ToUpper(code)
}

// optionalString returns a pointer to a trimmed string, or nil when empty so
// that missing optional fields serialize as JSON null.
func optionalString(value string) *string {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	return &v
}

// lowerAll lowercases and trims every entry of a list.
func lowerAll(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, strings.ToLower(v))
		}
	}
	return out
}

// formatBagCount renders a numeric baggage allowance as human readable text.
func formatBagCount(n int) string {
	switch {
	case n <= 0:
		return ""
	case n == 1:
		return "1 bag"
	default:
		return fmt.Sprintf("%d bags", n)
	}
}

// cleanBagPart removes the given keywords from a baggage description.
func cleanBagPart(part string, keywords ...string) string {
	out := part
	for _, kw := range keywords {
		out = strings.ReplaceAll(out, " "+kw, "")
		out = strings.ReplaceAll(out, kw+" ", "")
		out = strings.ReplaceAll(out, kw, "")
	}
	return strings.TrimSpace(out)
}
