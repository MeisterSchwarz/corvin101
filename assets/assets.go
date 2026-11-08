// Package assets contains the visual resources bundled into Corvin.
package assets

import "embed"

//go:embed schools/*.png
var Schools embed.FS

//go:embed items/*.png
var Items embed.FS

//go:embed Shermlock.ttf
var DisplayFont []byte
