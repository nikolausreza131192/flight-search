package currency

import "testing"

func TestFormat(t *testing.T) {
	cases := []struct {
		amount int64
		code   string
		want   string
	}{
		{485000, "IDR", "Rp 485.000"},
		{1250000, "IDR", "Rp 1.250.000"},
		{950, "", "Rp 950"},
		{100, "USD", "USD 100"},
		{1000000, "USD", "USD 1,000,000"},
		{0, "IDR", "Rp 0"},
	}
	for _, c := range cases {
		if got := Format(c.amount, c.code); got != c.want {
			t.Errorf("Format(%d, %q) = %q, want %q", c.amount, c.code, got, c.want)
		}
	}
}
