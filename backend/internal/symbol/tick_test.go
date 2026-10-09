package symbol

import "testing"

func TestTickFor(t *testing.T) {
	cases := []struct {
		exchange string
		price    float64
		want     float64
	}{
		{"HOSE", 9_990, 10}, {"HOSE", 10_000, 50}, {"HOSE", 49_950, 50}, {"HOSE", 50_000, 100},
		{"HNX", 9_000, 100}, {"UPCOM", 60_000, 100},
	}
	for _, c := range cases {
		if got := TickFor(c.exchange, c.price); got != c.want {
			t.Errorf("TickFor(%s, %v) = %v, want %v", c.exchange, c.price, got, c.want)
		}
	}
}

func TestBandRoundingStaysInside(t *testing.T) {
	// HOSE, reference 23,400: +7% = 25,038 -> ceiling 25,000; -7% = 21,762 -> floor 21,800.
	if got := FloorToTick(23_400*1.07, "HOSE"); got != 25_000 {
		t.Errorf("ceiling = %v, want 25000", got)
	}
	if got := CeilToTick(23_400*0.93, "HOSE"); got != 21_800 {
		t.Errorf("floor = %v, want 21800", got)
	}
	// Already on the grid stays put.
	if got := FloorToTick(23_450, "HOSE"); got != 23_450 {
		t.Errorf("on-grid = %v", got)
	}
}
