package platform

import (
	"context"
	"librequenda/branding"
	"librequenda/model"
	"librequenda/shared"
	"librequenda/storage"
	"net/http"
)

func brandingDto(b branding.Branding) *BrandingDto {
	return &BrandingDto{Name: b.Name(), CustomName: b.CustomName, LogoRef: b.LogoRef, LogoUrl: storage.URL(b.LogoRef), SupportEmail: shared.SUPPORT_EMAIL, SupportPhone: shared.SUPPORT_PHONE}
}

// Branding is the installation's name and logo: public, the web app shows them on every page.
func (s *platformService) Branding(ctx context.Context) (*BrandingDto, *shared.ErrorResponse) {
	var b, err = branding.Load(ctx, model.New(s.db))
	if err != nil {
		return nil, failed("errors.branding_load_failed")
	}
	return brandingDto(b), nil
}

func (s *platformService) UpdateBranding(ctx context.Context, input BrandingInput) (*BrandingDto, *shared.ErrorResponse) {
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var err = model.New(s.db).SaveAppSettings(ctx, model.SaveAppSettingsParams{
		AppName: nullString(input.Name),
		LogoRef: nullString(input.LogoRef),
	})
	if err != nil {
		return nil, failed("errors.branding_save_failed")
	}
	return brandingDto(branding.Branding{CustomName: input.Name, LogoRef: input.LogoRef}), nil
}
