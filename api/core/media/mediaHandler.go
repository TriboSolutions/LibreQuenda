package media

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"librequenda/auth"
	"librequenda/core/me"
	"librequenda/model"
	"librequenda/storage"
	"net/http"
	"strconv"
	"sync"

	"github.com/angelbarreiros/Penguin/router/helpers"
	"github.com/google/uuid"
)

const maxImageSize int64 = 5 * 1024 * 1024

// Detected from the file's bytes, never from its name. No SVG: it can carry scripts.
var allowedImages = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type mediaHandler struct {
	db *sql.DB
}

var hMediaOnce *sync.Once = new(sync.Once)
var hMediaInstance *mediaHandler

func MediaHandler(db *sql.DB) *mediaHandler {
	hMediaOnce.Do(func() {
		hMediaInstance = &mediaHandler{db: db}
	})
	return hMediaInstance
}

type UploadResponse struct {
	Ref string `json:"ref"` // what forms save
	Url string `json:"url"` // what screens show
}

// canUpload: anyone signed in may upload their own photo (users/) or a photo of a
// problem they report (issues/); council and facility
// images (councils/, facilities/) need a superadministrator or a council administrator,
// and the installation's logo (branding/) a superadministrator.
func (h *mediaHandler) canUpload(r *http.Request, folder string) (bool, error) {
	if folder == "users" || folder == "issues" {
		return true, nil
	}
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		return false, errors.New(userErr.Message)
	}
	var queries = model.New(h.db)
	var isSuperadmin, err = queries.IsPlatformAdmin(r.Context(), userId)
	if err != nil || isSuperadmin || folder == "branding" {
		return isSuperadmin, err
	}
	var roles []model.GetUserTenantRolesRow
	roles, err = queries.GetUserTenantRoles(r.Context(), userId)
	return len(roles) > 0, err
}

func (h *mediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	var err error
	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize+64*1024)
	if err = r.ParseMultipartForm(maxImageSize); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			helpers.SendI18NErrorResponse(w, r, http.StatusRequestEntityTooLarge, "errors.upload_too_large")
			return
		}
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.upload_invalid")
		return
	}
	defer r.MultipartForm.RemoveAll()

	var folder = r.FormValue("folder")
	if !storage.Folders[folder] {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.upload_invalid")
		return
	}
	var allowed bool
	allowed, err = h.canUpload(r, folder)
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusInternalServerError, "errors.upload_failed")
		return
	}
	if !allowed {
		helpers.SendI18NErrorResponse(w, r, http.StatusForbidden, "errors.forbidden")
		return
	}
	// images for facilities, councils or the installation are admin work
	if session, _ := auth.SessionFromContext(r.Context()); folder != "users" && !me.AdminSecondFactor(w, r, session) {
		return
	}
	var file, header, fileErr = r.FormFile("file")
	if fileErr != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.upload_invalid")
		return
	}
	defer file.Close()
	if header.Size > maxImageSize {
		helpers.SendI18NErrorResponse(w, r, http.StatusRequestEntityTooLarge, "errors.upload_too_large")
		return
	}

	var head = make([]byte, 512)
	var n, _ = io.ReadFull(file, head)
	var contentType = http.DetectContentType(head[:n])
	var ext, ok = allowedImages[contentType]
	if !ok {
		helpers.SendI18NErrorResponse(w, r, http.StatusUnsupportedMediaType, "errors.upload_type_not_allowed")
		return
	}

	var ref = folder + "/" + uuid.Must(uuid.NewV7()).String() + ext
	var body = io.MultiReader(bytes.NewReader(head[:n]), file)
	if err = storage.Put(r.Context(), ref, body, header.Size, contentType); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadGateway, "errors.upload_failed")
		return
	}
	helpers.SendSuccessResponse(w, UploadResponse{Ref: ref, Url: storage.URL(ref)})
}

func (h *mediaHandler) Serve(w http.ResponseWriter, r *http.Request) {
	var ref = r.PathValue("ref")
	if !storage.ValidRef(ref) {
		http.NotFound(w, r)
		return
	}
	var object, err = storage.Get(r.Context(), ref)
	if errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	defer object.Reader.Close()

	var etag = `"` + object.ETag + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // a reference never changes content
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", object.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(object.Size, 10))
	if r.Method == http.MethodHead {
		return
	}
	io.Copy(w, object.Reader)
}
