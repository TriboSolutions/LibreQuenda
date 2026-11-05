package branding

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"image"
	_ "image/jpeg" // logos are PNG, JPEG or WebP (core/media)
	"image/png"
	"io"
	"sync"

	"librequenda/model"
	"librequenda/shared"
	"librequenda/storage"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// LibreQuenda's own images, for an installation with no logo of its own: the
// lockup (mark, name and tagline) and the mark alone, the same as zitadel/brand.
var (
	//go:embed librequenda-logo.png
	defaultLockup []byte
	//go:embed librequenda-simbolo.png
	defaultMark []byte
)

// Image is a logo ready to send: always PNG (email clients and Zitadel take it,
// WebP not everywhere), no taller than asked.
type Image struct {
	PNG           []byte
	Width, Height int
}

// Email is what an email's header shows, like the web's top bar: LibreQuenda's
// lockup, or with a name or logo of the installation's own, its mark (the
// installation's logo, else LibreQuenda's) and its name beside it.
type Email struct {
	Name   string
	Lockup bool // Logo is LibreQuenda's lockup, with the name in it
	Logo   Image
}

// emailLogoHeight: twice the 32 px it is shown at, sharp on high-density screens.
const emailLogoHeight = 64

// ForEmail is the installation's name and logo for the emails. A logo that can't
// be read falls back to LibreQuenda's mark rather than stopping what is sent.
func ForEmail(ctx context.Context, db *sql.DB) Email {
	var b, err = Load(ctx, model.New(db))
	if err != nil {
		shared.Logger.Error("branding: %v", err)
	}
	var email = Email{Name: b.Name()}
	if b.CustomName == "" && b.LogoRef == "" {
		email.Lockup = true
		email.Logo = lockupImage()
		return email
	}
	if b.LogoRef != "" {
		var logo, logoErr = StoredLogo(ctx, b.LogoRef, emailLogoHeight)
		if logoErr == nil {
			email.Logo = logo
			return email
		}
		shared.Logger.Error("branding: email logo %s: %v", b.LogoRef, logoErr)
	}
	email.Logo = markImage()
	return email
}

// the embedded images, scaled once
var (
	lockupImage = sync.OnceValue(func() Image { return mustScale(defaultLockup) })
	markImage   = sync.OnceValue(func() Image { return mustScale(defaultMark) })
)

func mustScale(source []byte) Image {
	var img, err = Scale(bytes.NewReader(source), emailLogoHeight)
	if err != nil {
		panic(fmt.Sprintf("branding: embedded logo: %v", err)) // fixed at build time and covered by the tests
	}
	return img
}

// the stored logo, scaled, kept while its ref stays the same (a new upload gets a new ref)
var stored struct {
	sync.Mutex
	images map[string]Image
}

// StoredLogo reads the installation's logo from the storage as a PNG no taller than maxHeight.
func StoredLogo(ctx context.Context, ref string, maxHeight int) (Image, error) {
	var key = fmt.Sprintf("%s@%d", ref, maxHeight)
	stored.Lock()
	var img, ok = stored.images[key]
	stored.Unlock()
	if ok {
		return img, nil
	}
	var object, err = storage.Get(ctx, ref)
	if err != nil {
		return Image{}, err
	}
	defer object.Reader.Close()
	if img, err = Scale(object.Reader, maxHeight); err != nil {
		return Image{}, err
	}
	stored.Lock()
	if stored.images == nil {
		stored.images = map[string]Image{}
	}
	stored.images[key] = img
	stored.Unlock()
	return img, nil
}

// Scale decodes a PNG, JPEG or WebP image and encodes it as PNG, scaled down to
// maxHeight when taller (keeping its proportions; never scaled up).
func Scale(reader io.Reader, maxHeight int) (Image, error) {
	var source, _, err = image.Decode(reader)
	if err != nil {
		return Image{}, err
	}
	var bounds = source.Bounds()
	var width, height = bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return Image{}, fmt.Errorf("empty image")
	}
	var out = source
	if height > maxHeight {
		width = max(1, width*maxHeight/height)
		height = maxHeight
		var scaled = image.NewNRGBA(image.Rect(0, 0, width, height))
		draw.CatmullRom.Scale(scaled, scaled.Bounds(), source, bounds, draw.Src, nil)
		out = scaled
	}
	var buffer bytes.Buffer
	if err = png.Encode(&buffer, out); err != nil {
		return Image{}, err
	}
	return Image{PNG: buffer.Bytes(), Width: width, Height: height}, nil
}
