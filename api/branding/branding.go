// Package branding is how this installation presents itself: its name and logo,
// set by a superadministrator. Without them it is LibreQuenda with its own logo.
package branding

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/model"
	"librequenda/shared"
)

const DefaultName = "LibreQuenda"

type Branding struct {
	CustomName string // "" = DefaultName
	LogoRef    string // "" = the LibreQuenda logo
}

// Name is what the app is called: the custom name, or DefaultName.
func (b Branding) Name() string {
	if b.CustomName == "" {
		return DefaultName
	}
	return b.CustomName
}

// Load reads the saved branding; with nothing saved it is the default one.
func Load(ctx context.Context, queries *model.Queries) (Branding, error) {
	var row, err = queries.GetAppSettings(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return Branding{}, nil
	}
	if err != nil {
		return Branding{}, err
	}
	return Branding{CustomName: row.AppName.String, LogoRef: row.LogoRef.String}, nil
}

// Name is the app's name for emails and files: an error reading it falls back to
// DefaultName rather than stopping what is being sent.
func Name(ctx context.Context, db *sql.DB) string {
	var b, err = Load(ctx, model.New(db))
	if err != nil {
		shared.Logger.Error("branding: %v", err)
	}
	return b.Name()
}
