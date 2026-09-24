//go:build windows

package windows

import (
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"corvin101/internal/data/enemies"
)

type schoolIcon struct {
	width  int
	height int
	pixels []uint32
}

var schoolIcons struct {
	once sync.Once

	icons map[enemies.School]*schoolIcon
}

func loadSchoolIcons() {
	schoolIcons.once.Do(
		func() {
			schoolIcons.icons =
				make(
					map[enemies.School]*schoolIcon,
				)

			paths := map[enemies.School]string{
				enemies.SchoolFire:    "fire.png",
				enemies.SchoolIce:     "ice.png",
				enemies.SchoolStorm:   "storm.png",
				enemies.SchoolMyth:    "myth.png",
				enemies.SchoolLife:    "life.png",
				enemies.SchoolDeath:   "death.png",
				enemies.SchoolBalance: "balance.png",
				enemies.SchoolSun:     "sun.png",
				enemies.SchoolMoon:    "moon.png",
				enemies.SchoolStar:    "star.png",
				enemies.SchoolShadow:  "shadow.png",
			}

			for school, filename := range paths {

				icon, err :=
					readSchoolIcon(
						filepath.Join(
							"assets",
							"schools",
							filename,
						),
					)

				if err != nil {
					continue
				}

				schoolIcons.icons[school] =
					icon
			}
		},
	)
}

func readSchoolIcon(
	path string,
) (*schoolIcon, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	source, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := source.Bounds()

	width := bounds.Dx()
	height := bounds.Dy()

	pixels := make(
		[]uint32,
		width*height,
	)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			red, green, blue, alpha :=
				source.At(
					bounds.Min.X+x,
					bounds.Min.Y+y,
				).RGBA()

			a := uint32(alpha >> 8)
			r := uint32(red >> 8)
			g := uint32(green >> 8)
			b := uint32(blue >> 8)

			// Unser DIB-Buffer ist BGRA.
			pixels[y*width+x] =
				a<<24 |
					r<<16 |
					g<<8 |
					b
		}
	}

	return &schoolIcon{
		width:  width,
		height: height,
		pixels: pixels,
	}, nil
}

func getSchoolIcon(
	school enemies.School,
) *schoolIcon {
	loadSchoolIcons()

	return schoolIcons.icons[school]
}

func drawSchoolIcon(
	target unsafe.Pointer,
	targetWidth int32,
	targetHeight int32,
	school enemies.School,
	centerX int32,
	top int32,
	size int32,
) {
	if target == nil ||
		targetWidth <= 0 ||
		targetHeight <= 0 ||
		size <= 0 {

		return
	}

	icon := getSchoolIcon(
		school,
	)

	if icon == nil ||
		icon.width <= 0 ||
		icon.height <= 0 {

		return
	}

	buffer := unsafe.Slice(
		(*uint32)(target),
		int(targetWidth*targetHeight),
	)

	left :=
		centerX -
			size/2

	for destinationY :=
		int32(0); destinationY < size; destinationY++ {

		y := top + destinationY

		if y < 0 ||
			y >= targetHeight {

			continue
		}

		sourceY :=
			int(destinationY) *
				icon.height /
				int(size)

		for destinationX :=
			int32(0); destinationX < size; destinationX++ {

			x := left + destinationX

			if x < 0 ||
				x >= targetWidth {

				continue
			}

			sourceX :=
				int(destinationX) *
					icon.width /
					int(size)

			sourcePixel :=
				icon.pixels[sourceY*icon.width+sourceX]

			sourceAlpha :=
				byte(
					sourcePixel >> 24,
				)

			if sourceAlpha == 0 {
				continue
			}

			index :=
				int(y*targetWidth + x)

			buffer[index] =
				alphaBlendPixel(
					buffer[index],
					sourcePixel,
				)
		}
	}
}

func alphaBlendPixel(
	destination uint32,
	source uint32,
) uint32 {
	sourceAlpha :=
		uint32(
			byte(source >> 24),
		)

	if sourceAlpha == 0 {
		return destination
	}

	if sourceAlpha == 255 {
		return source
	}

	destinationAlpha :=
		uint32(
			byte(destination >> 24),
		)

	sourceRed :=
		(source >> 16) & 0xFF

	sourceGreen :=
		(source >> 8) & 0xFF

	sourceBlue :=
		source & 0xFF

	destinationRed :=
		(destination >> 16) & 0xFF

	destinationGreen :=
		(destination >> 8) & 0xFF

	destinationBlue :=
		destination & 0xFF

	inverseAlpha :=
		uint32(255) -
			sourceAlpha

	outAlpha :=
		sourceAlpha +
			destinationAlpha*
				inverseAlpha/
				255

	if outAlpha == 0 {
		return 0
	}

	outRed :=
		(sourceRed*sourceAlpha +
			destinationRed*
				inverseAlpha) /
			255

	outGreen :=
		(sourceGreen*sourceAlpha +
			destinationGreen*
				inverseAlpha) /
			255

	outBlue :=
		(sourceBlue*sourceAlpha +
			destinationBlue*
				inverseAlpha) /
			255

	return outAlpha<<24 |
		outRed<<16 |
		outGreen<<8 |
		outBlue
}
