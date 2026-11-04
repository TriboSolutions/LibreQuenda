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

// owner says whose file an upload is: the council (uuid.Nil: the installation)
// and whether this person may store it there.
//   - users/ (their own photo): anyone signed in, the installation's
//   - issues/ (a photo of a problem): anyone signed in, the council of the
//     active facility in "facility" (the report itself checks the booking)
//   - councils/, facilities/: a superadministrator or an administrator of the
//     council in "council" (its slug)
//   - branding/ (the installation's logo): a superadministrator
func (h *mediaHandler) owner(r *http.Request, folder string) (council uuid.UUID, allowed bool, err error) {
	var queries = model.New(h.db)
	switch folder {
	case "users":
		return uuid.Nil, true, nil
	case "issues":
		var facilityId, parseErr = uuid.Parse(r.FormValue("facility"))
		if parseErr != nil {
			return uuid.Nil, false, nil
		}
		var facility, getErr = queries.GetFacilityForIssue(r.Context(), facilityId)
		if errors.Is(getErr, sql.ErrNoRows) {
			return uuid.Nil, false, nil
		}
		return facility.TenantID, getErr == nil, getErr
	}
	var session, _ = auth.SessionFromContext(r.Context())
	var userId, userErr = me.CurrentUserId(r.Context(), h.db, session)
	if userErr != nil {
		return uuid.Nil, false, errors.New(userErr.Message)
	}
	var isSuperadmin bool
	if isSuperadmin, err = queries.IsPlatformAdmin(r.Context(), userId); err != nil {
		return uuid.Nil, false, err
	}
	if folder == "branding" {
		return uuid.Nil, isSuperadmin, nil
	}
	var tenant, getErr = queries.GetTenantIdBySlug(r.Context(), r.FormValue("council"))
	if errors.Is(getErr, sql.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if getErr != nil {
		return uuid.Nil, false, getErr
	}
	if isSuperadmin {
		return tenant.ID, true, nil
	}
	var roles []model.GetUserTenantRolesRow
	if roles, err = queries.GetUserTenantRoles(r.Context(), userId); err != nil {
		return uuid.Nil, false, err
	}
	for _, role := range roles {
		if role.TenantID == tenant.ID && role.Role == model.TenantRoleAdmin {
			return tenant.ID, true, nil
		}
	}
	return uuid.Nil, false, nil
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
	if !storage.CouncilFolders[folder] && !storage.PlatformFolders[folder] {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadRequest, "errors.upload_invalid")
		return
	}
	var council, allowed, ownerErr = h.owner(r, folder)
	err = ownerErr
	if err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusInternalServerError, "errors.upload_failed")
		return
	}
	if !allowed {
		helpers.SendI18NErrorResponse(w, r, http.StatusForbidden, "errors.forbidden")
		return
	}
	// images for facilities, councils or the installation are admin work
	if session, _ := auth.SessionFromContext(r.Context()); folder != "users" && folder != "issues" && !me.AdminSecondFactor(w, r, session) {
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

	var name = uuid.Must(uuid.NewV7()).String() + ext
	var ref = folder + "/" + name
	if council != uuid.Nil {
		ref = storage.CouncilRef(council, folder, name)
	}
	var body = io.MultiReader(bytes.NewReader(head[:n]), file)
	if err = storage.Put(r.Context(), ref, body, header.Size, contentType); err != nil {
		helpers.SendI18NErrorResponse(w, r, http.StatusBadGateway, "errors.upload_failed")
		return
	}
	helpers.SendSuccessResponse(w, UploadResponse{Ref: ref, Url: storage.URL(ref)})
}
