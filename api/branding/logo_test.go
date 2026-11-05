package branding

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestEmbeddedLogosScaleForEmails(t *testing.T) {
	var lockup, mark = lockupImage(), markImage()
	if lockup.Height != emailLogoHeight || lockup.Width <= lockup.Height || len(lockup.PNG) == 0 {
		t.Errorf("lockup: %dx%d, %d bytes", lockup.Width, lockup.Height, len(lockup.PNG))
	}
	if mark.Height != emailLogoHeight || mark.Width != emailLogoHeight {
		t.Errorf("mark: %dx%d", mark.Width, mark.Height)
	}
}

func TestScaleKeepsProportionsAndNeverEnlarges(t *testing.T) {
	var source = image.NewNRGBA(image.Rect(0, 0, 300, 100))
	source.Set(0, 0, color.Black)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	var scaled, err = Scale(bytes.NewReader(encoded.Bytes()), 50)
	if err != nil || scaled.Width != 150 || scaled.Height != 50 {
		t.Fatalf("scaled down: %dx%d, %v", scaled.Width, scaled.Height, err)
	}
	if decoded, decodeErr := png.Decode(bytes.NewReader(scaled.PNG)); decodeErr != nil || decoded.Bounds().Dx() != 150 {
		t.Errorf("not a PNG of the new size: %v", decodeErr)
	}
	if kept, _ := Scale(bytes.NewReader(encoded.Bytes()), 400); kept.Width != 300 || kept.Height != 100 {
		t.Errorf("small enough already: %dx%d", kept.Width, kept.Height)
	}
	if _, err = Scale(bytes.NewReader([]byte("<svg/>")), 50); err == nil {
		t.Error("not an image, yet decoded")
	}
}
