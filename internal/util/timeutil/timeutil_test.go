package timeutil

import "testing"

func TestFormatDuration(t *testing.T) {
	cases := map[int]string{
		0:   "0m",
		45:  "45m",
		90:  "1h 30m",
		110: "1h 50m",
		120: "2h",
		260: "4h 20m",
	}
	for minutes, want := range cases {
		if got := FormatDuration(minutes); got != want {
			t.Errorf("FormatDuration(%d) = %q, want %q", minutes, got, want)
		}
	}
}

func TestParseSupportedLayouts(t *testing.T) {
	inputs := []string{
		"2025-12-15T06:00:00+07:00", // Garuda / AirAsia
		"2025-12-15T07:15:00+0700",  // Batik Air
		"2025-12-15T05:30:00",       // Lion Air naive
	}
	for _, in := range inputs {
		if _, err := Parse(in); err != nil {
			t.Errorf("Parse(%q) returned error: %v", in, err)
		}
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse("not-a-date"); err == nil {
		t.Fatal("expected error for invalid timestamp")
	}
}

func TestParseInLocationAppliesTimezone(t *testing.T) {
	jakarta, err := ParseInLocation("2025-12-15T05:30:00", "Asia/Jakarta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, offset := jakarta.Zone(); offset != 7*3600 {
		t.Errorf("expected +7h offset, got %d", offset)
	}

	makassar, err := ParseInLocation("2025-12-15T08:15:00", "Asia/Makassar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, offset := makassar.Zone(); offset != 8*3600 {
		t.Errorf("expected +8h offset, got %d", offset)
	}
}

func TestMinutesOfDay(t *testing.T) {
	got, err := MinutesOfDay("08:30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 8*60+30 {
		t.Errorf("MinutesOfDay(08:30) = %d, want %d", got, 8*60+30)
	}
}
