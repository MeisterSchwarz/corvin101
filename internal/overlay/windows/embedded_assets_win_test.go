//go:build windows

package windows

import (
	"corvin101/internal/data/enemies"
	"testing"
)

func TestEmbeddedOverlayAssetsWithoutAssetDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	raven := loadLoadingRaven()
	if raven == nil || len(raven.pixels) == 0 {
		t.Fatal("embedded loading raven missing")
	}
	visible := false
	for _, pixel := range raven.pixels {
		if pixel>>24 != 0 {
			visible = true
			break
		}
	}
	if !visible {
		t.Fatal("loading raven is fully transparent")
	}
	loadOverlayFonts()
	if overlayFontHandle == 0 {
		t.Fatal("embedded font was not registered")
	}
	for _, school := range []enemies.School{
		enemies.SchoolFire, enemies.SchoolIce, enemies.SchoolStorm, enemies.SchoolMyth,
		enemies.SchoolLife, enemies.SchoolDeath, enemies.SchoolBalance, enemies.SchoolSun,
		enemies.SchoolMoon, enemies.SchoolStar, enemies.SchoolShadow, enemies.SchoolAny,
	} {
		icon := getSchoolIcon(school)
		if icon == nil || icon.width == 0 || icon.height == 0 || len(icon.pixels) != icon.width*icon.height {
			t.Fatalf("embedded school icon missing: %s", school)
		}
	}
}
