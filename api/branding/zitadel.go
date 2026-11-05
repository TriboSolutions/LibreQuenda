package branding

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"librequenda/model"
	"librequenda/shared"
	"librequenda/zitadel"
)

// zitadelLogoHeight: big enough for the sign-in page's header, small for its emails.
const zitadelLogoHeight = 256

var syncing sync.Mutex // one copy at a time: a save while the start-up one runs waits

// SyncZitadel copies the installation's name and logo to Zitadel (see
// zitadel.SyncBranding), in the background: on start, and after every save.
// A failure is logged; the next save or start tries again.
func SyncZitadel(db *sql.DB) {
	go func() {
		syncing.Lock()
		defer syncing.Unlock()
		var ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		var b, err = Load(ctx, model.New(db))
		if err != nil {
			shared.Logger.Error("branding: zitadel: %v", err)
			return
		}
		var logo []byte
		if b.LogoRef != "" {
			var image, logoErr = StoredLogo(ctx, b.LogoRef, zitadelLogoHeight)
			if logoErr != nil {
				shared.Logger.Error("branding: zitadel: logo %s: %v", b.LogoRef, logoErr)
				return
			}
			logo = image.PNG
		}
		err = zitadel.SyncBranding(ctx, b.Name(), logo)
		if errors.Is(err, zitadel.ErrUnavailable) {
			return // no service account (tests, or Zitadel not set up): nothing to copy to
		}
		if err != nil {
			shared.Logger.Error("branding: zitadel: %v", err)
			return
		}
		shared.Logger.Info("branding: zitadel has the name %q (own logo: %t)", b.Name(), logo != nil)
	}()
}
