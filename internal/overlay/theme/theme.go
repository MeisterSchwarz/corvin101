package theme

const (
	DisplayFontFamily = "Shermlock"
	UIFontFamily      = "Segoe UI"
)

const (
	RoundFontSize    = 92.0
	RoundOutlineSize = 4
	RoundShadowX     = 4
	RoundShadowY     = 5

	AffinityFontSize    = 17.0
	AffinityOutlineSize = 2
)

type RGB struct {
	R byte
	G byte
	B byte
}

var (
	RoundColor          = RGB{R: 255, G: 211, B: 38}
	RoundHighlightColor = RGB{R: 255, G: 239, B: 118}
	RoundOutlineColor   = RGB{R: 20, G: 15, B: 6}
	RoundShadowColor    = RGB{R: 4, G: 3, B: 2}

	AffinityResistColor  = RGB{R: 245, G: 220, B: 105}
	AffinityBoostColor   = RGB{R: 235, G: 235, B: 235}
	AffinityOutlineColor = RGB{R: 12, G: 10, B: 7}
)
