package fade

import (
	"fmt"
	"image"
	"testing"

	"github.com/charmbracelet/colorprofile"
)

func TestWorld_SumImage(t *testing.T) {
	assets, err := LoadAssets()
	if err != nil {
		t.Fatal(err)
	}
	world := NewWorld(assets)
	r := image.Rect(1, 1, 5, 5)
	sum := world.SumImage(r)
	if sum.Bounds().Dx() != r.Dx()*16 {
		t.Errorf("SumImage returned incorrect width: got %d, want %d", sum.Bounds().Dx(), r.Dx()*16)
	}
	if sum.Bounds().Dy() != r.Dy()*16 {
		t.Errorf("SumImage returned incorrect height: got %d, want %d", sum.Bounds().Dy(), r.Dy()*16)
	}
	fmt.Fprint(t.Output(), imageAsString(sum, colorprofile.ANSI))
}
