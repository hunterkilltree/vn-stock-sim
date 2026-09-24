package crypto

import "strings"

// Categories in design/screens/Crypto-Main.dc.html's order.
const (
	CatLayer1   = "Layer 1"
	CatDeFi     = "DeFi"
	CatExchange = "Sàn & hạ tầng"
	CatVietnam  = "Dự án Việt Nam"
)

var categoryOrder = []string{CatLayer1, CatDeFi, CatExchange, CatVietnam}

// universe is Crypto-Main.dc.html's own 24 coins. Supply and all-time
// high are hand-seeded approximations (phase-i.md decision 3); the
// service still raises the all-time high to the highest price it has
// actually seen, so "distance from ATH" can't go positive. seedPrice only
// anchors the mock generator.
var universe = []Pair{
	p("BTC", "Bitcoin", CatLayer1, 19_780_000, 21_000_000, 64_820.5, 73_750, true),
	p("ETH", "Ethereum", CatLayer1, 120_200_000, 0, 3_412.8, 4_878, true),
	p("SOL", "Solana", CatLayer1, 470_000_000, 0, 184.2, 260, true),
	p("AVAX", "Avalanche", CatLayer1, 407_000_000, 720_000_000, 36.1, 146, true),
	p("ADA", "Cardano", CatLayer1, 35_900_000_000, 45_000_000_000, 0.482, 3.10, true),
	p("DOT", "Polkadot", CatLayer1, 1_500_000_000, 0, 7.12, 55, true),
	p("UNI", "Uniswap", CatDeFi, 600_000_000, 1_000_000_000, 10.4, 45, true),
	p("AAVE", "Aave", CatDeFi, 15_000_000, 16_000_000, 162, 666, true),
	p("LINK", "Chainlink", CatDeFi, 626_000_000, 1_000_000_000, 17.5, 52.9, true),
	p("MKR", "Maker", CatDeFi, 880_000, 1_005_577, 2_184, 6_339, true),
	p("CRV", "Curve DAO", CatDeFi, 1_280_000_000, 3_030_000_000, 0.684, 15.4, true),
	p("LDO", "Lido DAO", CatDeFi, 895_000_000, 1_000_000_000, 2.21, 7.3, true),
	p("BNB", "BNB", CatExchange, 146_000_000, 200_000_000, 580, 720, true),
	p("OKB", "OKB", CatExchange, 60_000_000, 0, 48.2, 73, false),
	p("ATOM", "Cosmos", CatExchange, 390_000_000, 0, 7.42, 44.7, true),
	p("FIL", "Filecoin", CatExchange, 590_000_000, 1_960_000_000, 5.9, 237, true),
	p("ARB", "Arbitrum", CatExchange, 4_000_000_000, 10_000_000_000, 0.92, 2.39, true),
	p("OP", "Optimism", CatExchange, 1_260_000_000, 4_294_967_296, 2.14, 4.84, true),
	p("AXS", "Axie Infinity", CatVietnam, 150_000_000, 270_000_000, 6.84, 165, true),
	p("RON", "Ronin", CatVietnam, 360_000_000, 1_000_000_000, 1.842, 4.46, true),
	p("KNC", "Kyber Network", CatVietnam, 180_000_000, 0, 0.62, 5.8, true),
	p("C98", "Coin98", CatVietnam, 790_000_000, 1_000_000_000, 0.18, 6.1, true),
	p("SLP", "Smooth Love Potion", CatVietnam, 40_000_000_000, 0, 0.00412, 0.4, true),
	p("VNDC", "VNDC", CatVietnam, 38_000_000_000, 0, 0.0000392, 0.0000425, false),
}

func p(base, name, category string, circ, max, price, ath float64, listed bool) Pair {
	return Pair{
		Symbol: base + "USDT", Base: base, Quote: "USDT", Name: name, Category: category,
		CirculatingSupply: circ, MaxSupply: max, listed: listed, seedPrice: price, seedATH: ath,
	}
}

var bySymbol = func() map[string]Pair {
	m := make(map[string]Pair, len(universe))
	for _, pr := range universe {
		m[pr.Symbol] = pr
	}
	return m
}()

// Lookup accepts "BTCUSDT", "BTC/USDT", "btc-usdt" or "BTC".
func Lookup(raw string) (Pair, bool) {
	s := strings.ToUpper(strings.NewReplacer("/", "", "-", "", "_", "", " ", "").Replace(raw))
	if pr, ok := bySymbol[s]; ok {
		return pr, true
	}
	pr, ok := bySymbol[s+"USDT"]
	return pr, ok
}
