package storage

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"image/png"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// The longest side an image is kept at: a facility's photo fills a phone or a
// tablet's header at most; a profile photo is a small circle.
const (
	MaxPhotoSide  = 1280
	MaxAvatarSide = 512
	jpegQuality   = 80
	// a phone's photo is 12-50 MP; past this, decoding alone would take too much memory
	maxPixels = 60_000_000
)

// MaxSide is how large the images of a folder are kept.
func MaxSide(folder string) int {
	if folder == "users" {
		return MaxAvatarSide
	}
	return MaxPhotoSide
}

// Shrink fits an uploaded image to what the screens show: no side longer than
// maxSide, a JPEG re-encoded without its metadata (a phone's photo carries
// where it was taken) and turned as the camera said. JPEG and PNG keep their
// format; a WebP that has to be scaled becomes a JPEG (a PNG with transparency),
// as Go writes no WebP. Anything it can't read comes back as it was.
func Shrink(data []byte, contentType string, maxSide int) ([]byte, string) {
	var config, format, err = image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width*config.Height > maxPixels {
		return data, contentType
	}
	var isJPEG = format == "jpeg"
	var scale = max(config.Width, config.Height) > maxSide
	if !scale && !isJPEG {
		return data, contentType
	}
	var img image.Image
	if img, _, err = image.Decode(bytes.NewReader(data)); err != nil {
		return data, contentType
	}

	var orientation, hasMetadata = 1, false
	if isJPEG {
		orientation, hasMetadata = jpegMetadata(data)
	}
	if scale {
		var w, h = config.Width, config.Height
		if w >= h {
			w, h = maxSide, max(1, h*maxSide/w)
		} else {
			w, h = max(1, w*maxSide/h), maxSide
		}
		var scaled = image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, img.Bounds(), draw.Src, nil)
		img = scaled
	}
	img = orient(img, orientation)

	var out bytes.Buffer
	var outType = "image/png"
	if isJPEG || (format == "webp" && opaque(img)) {
		outType = "image/jpeg"
		err = jpeg.Encode(&out, img, &jpeg.Options{Quality: jpegQuality})
	} else {
		var encoder = png.Encoder{CompressionLevel: png.BestCompression}
		err = encoder.Encode(&out, img)
	}
	if err != nil {
		return data, contentType
	}
	// a small JPEG already compressed harder, with nothing to strip or turn, stays as it was
	if !scale && !hasMetadata && orientation == 1 && out.Len() >= len(data) {
		return data, contentType
	}
	return out.Bytes(), outType
}

// ExtensionFor is the file extension of the types Shrink returns.
func ExtensionFor(contentType string) string {
	return map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}[contentType]
}

// SameType says whether a reference's extension still fits a content type.
func SameType(ref string, contentType string) bool {
	var ext = strings.ToLower(ref[strings.LastIndex(ref, "."):])
	return ext == ExtensionFor(contentType) || (ext == ".jpeg" && contentType == "image/jpeg")
}

func opaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	return false
}

// jpegMetadata reads a JPEG's EXIF orientation (1 when it has none) and whether
// it carries any EXIF or XMP at all.
func jpegMetadata(data []byte) (orientation int, hasMetadata bool) {
	orientation = 1
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return
	}
	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return
		}
		var marker = data[i+1]
		if marker == 0xDA || marker == 0xD9 { // the image data starts: no more metadata
			return
		}
		var length = int(binary.BigEndian.Uint16(data[i+2:]))
		if length < 2 || i+2+length > len(data) {
			return
		}
		var segment = data[i+4 : i+2+length]
		if marker == 0xE1 { // APP1: EXIF or XMP
			hasMetadata = true
			if bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
				if o := exifOrientation(segment[6:]); o >= 1 && o <= 8 {
					orientation = o
				}
			}
		}
		i += 2 + length
	}
	return
}

// exifOrientation finds tag 0x0112 in the first IFD of a TIFF block.
func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 0
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	var ifd = int(order.Uint32(tiff[4:]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 0
	}
	var count = int(order.Uint16(tiff[ifd:]))
	for n := 0; n < count; n++ {
		var entry = ifd + 2 + n*12
		if entry+12 > len(tiff) {
			return 0
		}
		if order.Uint16(tiff[entry:]) == 0x0112 {
			return int(order.Uint16(tiff[entry+8:]))
		}
	}
	return 0
}

// orient turns an image as EXIF orientation 2-8 says, so it shows upright
// once the tag is gone.
func orient(src image.Image, orientation int) image.Image {
	if orientation < 2 || orientation > 8 {
		return src
	}
	var b = src.Bounds()
	var w, h = b.Dx(), b.Dy()
	var swap = orientation >= 5
	var dst *image.RGBA
	if swap {
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
	} else {
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch orientation {
			case 2: // mirrored
				dx, dy = w-1-x, y
			case 3: // upside down
				dx, dy = w-1-x, h-1-y
			case 4: // upside down, mirrored
				dx, dy = x, h-1-y
			case 5: // mirrored, a quarter turn left
				dx, dy = y, x
			case 6: // a quarter turn right
				dx, dy = h-1-y, x
			case 7: // mirrored, a quarter turn right
				dx, dy = h-1-y, w-1-x
			case 8: // a quarter turn left
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
