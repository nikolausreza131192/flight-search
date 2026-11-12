package providers

import (
	"os"
	"path/filepath"
	"testing"

	"bookcabin/config"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "template", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestNormalizeCounts(t *testing.T) {
	cases := []struct {
		file      string
		normalize func([]byte) (int, error)
		want      int
	}{
		{"garuda_indonesia_search_response.json", countGaruda, 3},
		{"lion_air_search_response.json", countLion, 3},
		{"batik_air_search_response.json", countBatik, 3},
		{"airasia_search_response.json", countAirAsia, 4},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			got, err := c.normalize(readFixture(t, c.file))
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			if got != c.want {
				t.Errorf("got %d flights, want %d", got, c.want)
			}
		})
	}
}

func countGaruda(data []byte) (int, error) {
	f, err := NewGaruda(config.Config{}).normalize(data)
	return len(f), err
}
func countLion(data []byte) (int, error) {
	f, err := NewLion(config.Config{}).normalize(data)
	return len(f), err
}
func countBatik(data []byte) (int, error) {
	f, err := NewBatik(config.Config{}).normalize(data)
	return len(f), err
}
func countAirAsia(data []byte) (int, error) {
	f, err := NewAirAsia(config.Config{}).normalize(data)
	return len(f), err
}
