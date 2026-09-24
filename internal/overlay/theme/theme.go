package theme

const (
	DisplayFontFamily = "Shermlock"
)

const (
	RoundFontSize = 92.0

	RoundOutlineSize = 4
	RoundShadowX     = 4
	RoundShadowY     = 5
)

type RGB struct {
	R byte
	G byte
	B byte
}

var (
	RoundColor = RGB{
		R: 255,
		G: 211,
		B: 38,
	}

	RoundHighlightColor = RGB{
		R: 255,
		G: 239,
		B: 118,
	}

	RoundOutlineColor = RGB{
		R: 20,
		G: 15,
		B: 6,
	}

	RoundShadowColor = RGB{
		R: 4,
		G: 3,
		B: 2,
	}
)
