package market

import "testing"

func TestWithVolumeSplit(t *testing.T) {
	bars := withVolumeSplit([]Bar{
		{Low: 10, High: 20, Close: 20, Volume: 1000},                                // closes at high
		{Low: 10, High: 20, Close: 10, Volume: 1000},                                // closes at low
		{Low: 10, High: 20, Close: 15, Volume: 1001},                                // mid
		{Low: 10, High: 10, Close: 10, Volume: 100},                                 // flat
		{Low: 10, High: 20, Close: 20, Volume: 1000, BuyVolume: 1, SellVolume: 999}, // kept
	})
	want := [][2]int64{{1000, 0}, {0, 1000}, {501, 500}, {50, 50}, {1, 999}}
	for i, w := range want {
		if bars[i].BuyVolume != w[0] || bars[i].SellVolume != w[1] {
			t.Errorf("bar %d: got buy=%d sell=%d, want %v", i, bars[i].BuyVolume, bars[i].SellVolume, w)
		}
		if bars[i].BuyVolume+bars[i].SellVolume != bars[i].Volume && i != 4 {
			t.Errorf("bar %d: split does not sum to volume", i)
		}
	}
}
