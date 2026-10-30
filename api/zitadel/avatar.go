package zitadel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"librequenda/shared"
)

// The profile photo lives in LibreQuenda's storage and is copied to the person's
// Zitadel account, so the sign-in pages show the same one. Zitadel only takes a
// photo from its owner, so these calls go with the person's own access token
// (the web asks for Zitadel's audience too), not the service account.

// SetOwnAvatar uploads the photo to the token's own Zitadel account.
func SetOwnAvatar(ctx context.Context, accessToken string, photo io.Reader, contentType string) error {
	var body bytes.Buffer
	var form = multipart.NewWriter(&body)
	var header = make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="avatar"`)
	header.Set("Content-Type", contentType)
	var part, err = form.CreatePart(header)
	if err != nil {
		return err
	}
	if _, err = io.Copy(part, photo); err != nil {
		return err
	}
	if err = form.Close(); err != nil {
		return err
	}
	return own(ctx, accessToken, http.MethodPost, "/assets/v1/users/me/avatar", &body, form.FormDataContentType())
}

// RemoveOwnAvatar takes the photo off the token's own Zitadel account.
func RemoveOwnAvatar(ctx context.Context, accessToken string) error {
	return own(ctx, accessToken, http.MethodDelete, "/auth/v1/users/me/avatar", nil, "")
}

func own(ctx context.Context, accessToken, method, path string, body io.Reader, contentType string) error {
	var req, err = http.NewRequestWithContext(ctx, method, shared.ZITADEL_INTERNAL_URL+path, body)
	if err != nil {
		return err
	}
	req.Host = shared.ZITADEL_DOMAIN // Zitadel resolves the instance from the Host header
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	var res *http.Response
	if res, err = client.Do(req); err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("zitadel: %s %s returned %d", method, path, res.StatusCode)
	}
	return nil
}
