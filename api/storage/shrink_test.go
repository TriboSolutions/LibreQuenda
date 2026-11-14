package storage

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// photo is a w×h JPEG, its top-left corner red, the rest blue.
func photo(t *testing.T, w, h int) []byte {
	var img = image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var c = color.RGBA{0, 0, 255, 255}
			if x < w/4 && y < h/4 {
				c = color.RGBA{255, 0, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// withOrientation puts an EXIF block with the orientation tag right after the JPEG's SOI.
func withOrientation(data []byte, orientation uint16) []byte {
	var tiff = []byte("MM\x00\x2a\x00\x00\x00\x08\x00\x01")
	tiff = binary.BigEndian.AppendUint16(tiff, 0x0112)
	tiff = binary.BigEndian.AppendUint16(tiff, 3) // SHORT
	tiff = binary.BigEndian.AppendUint32(tiff, 1)
	tiff = binary.BigEndian.AppendUint16(tiff, orientation)
	tiff = append(tiff, 0, 0, 0, 0, 0, 0) // padding, next IFD
	var segment = append([]byte("Exif\x00\x00"), tiff...)
	var app1 = []byte{0xFF, 0xE1}
	app1 = binary.BigEndian.AppendUint16(app1, uint16(len(segment)+2))
	app1 = append(app1, segment...)
	return append(append(append([]byte{}, data[:2]...), app1...), data[2:]...)
}

func size(t *testing.T, data []byte) (int, int) {
	var config, _, err = image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return config.Width, config.Height
}

func TestShrinkScalesALargePhoto(t *testing.T) {
	var data, contentType = Shrink(photo(t, 2400, 1600), "image/jpeg", MaxPhotoSide)
	if contentType != "image/jpeg" {
		t.Fatalf("type = %s", contentType)
	}
	if w, h := size(t, data); w != 1280 || h != 853 {
		t.Fatalf("size = %dx%d, want 1280x853", w, h)
	}
	var avatar, _ = Shrink(photo(t, 600, 2400), "image/jpeg", MaxSide("users"))
	if w, h := size(t, avatar); w != 128 || h != 512 {
		t.Fatalf("avatar = %dx%d, want 128x512", w, h)
	}
}

func TestShrinkTurnsAsTheCameraSaid(t *testing.T) {
	// taken in portrait: stored 800×400, tagged «turn a quarter right»
	var data, _ = Shrink(withOrientation(photo(t, 800, 400), 6), "image/jpeg", MaxPhotoSide)
	if w, h := size(t, data); w != 400 || h != 800 {
		t.Fatalf("size = %dx%d, want 400x800", w, h)
	}
	if o, meta := jpegMetadata(data); o != 1 || meta {
		t.Fatalf("metadata left: orientation %d, %v", o, meta)
	}
	// the red corner, top-left before, is top-right now
	var img, _ = jpeg.Decode(bytes.NewReader(data))
	if r, _, b, _ := img.At(380, 20).RGBA(); r < b {
		t.Fatal("top-right should be red")
	}
	if r, _, b, _ := img.At(20, 20).RGBA(); r > b {
		t.Fatal("top-left should be blue")
	}
}

func TestShrinkLeavesSmallImagesAlone(t *testing.T) {
	var img = image.NewNRGBA(image.Rect(0, 0, 300, 200))
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	var logo = buf.Bytes()
	if data, contentType := Shrink(logo, "image/png", MaxPhotoSide); !bytes.Equal(data, logo) || contentType != "image/png" {
		t.Fatal("a small PNG should stay as it was")
	}
	var garbage = []byte("not an image at all")
	if data, contentType := Shrink(garbage, "image/jpeg", MaxPhotoSide); !bytes.Equal(data, garbage) || contentType != "image/jpeg" {
		t.Fatal("what can't be read should stay as it was")
	}
}

func TestShrinkKeepsTransparencyInPNG(t *testing.T) {
	var img = image.NewNRGBA(image.Rect(0, 0, 2000, 1000)) // fully transparent
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	var data, contentType = Shrink(buf.Bytes(), "image/png", MaxPhotoSide)
	if contentType != "image/png" {
		t.Fatalf("type = %s", contentType)
	}
	if w, h := size(t, data); w != 1280 || h != 640 {
		t.Fatalf("size = %dx%d", w, h)
	}
}

func TestSameType(t *testing.T) {
	if !SameType("x/facilities/a.jpg", "image/jpeg") || !SameType("users/a.JPEG", "image/jpeg") || SameType("users/a.png", "image/jpeg") {
		t.Fatal("SameType")
	}
}
