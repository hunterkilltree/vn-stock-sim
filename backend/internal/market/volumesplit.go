package market

// withVolumeSplit fills BuyVolume/SellVolume on every bar that doesn't
// already carry a split. Neither the mock generator nor Vietcap's OHLCV
// endpoint reports aggressor-side volume (that needs tick-level trade
// data), so this is an *estimate*: the share of a bar's volume attributed
// to buyers is where the close sits inside the bar's range (close-
// location value). A bar closing at its high is all buying, at its low all
// selling, and a flat bar splits 50/50. BuyVolume + SellVolume always
// equals Volume. Replace with real tick data if a licensed feed lands.
func withVolumeSplit(bars []Bar) []Bar {
	for i := range bars {
		b := &bars[i]
		if b.BuyVolume != 0 || b.SellVolume != 0 {
			continue
		}
		share := 0.5
		if b.High > b.Low {
			share = (b.Close - b.Low) / (b.High - b.Low)
		}
		if share < 0 {
			share = 0
		} else if share > 1 {
			share = 1
		}
		b.BuyVolume = int64(float64(b.Volume)*share + 0.5)
		b.SellVolume = b.Volume - b.BuyVolume
	}
	return bars
}
