package counciladmin

import (
	"context"
	"database/sql"
	"errors"
	"librequenda/audit"
	"librequenda/db"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// The council's open data: its public DCAT-AP-ES catalog for datos.gob.es
// (api/core/opendata serves it). The council turns it on and names itself
// with its DIR3 code, the publisher datos.gob.es knows it by.
type OpenDataInput struct {
	Enabled  bool   `json:"enabled"`
	Dir3Code string `json:"dir3_code"`
}

type OpenDataDto struct {
	Enabled   bool       `json:"enabled"`
	Dir3Code  string     `json:"dir3_code"`
	IssuedAt  *time.Time `json:"issued_at"`  // first turned on
	UpdatedAt *time.Time `json:"updated_at"` // nil: never saved
}

// DIR3: a letter and eight letters or digits, e.g. L01150301 (a council: L01 and its INE code)
var dir3Pattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{8}$`)

func (in *OpenDataInput) validate() string {
	in.Dir3Code = strings.ToUpper(strings.TrimSpace(in.Dir3Code))
	if !dir3Pattern.MatchString(in.Dir3Code) {
		return "errors.invalid_dir3_code"
	}
	return ""
}

func (s *counciladminService) GetOpenData(ctx context.Context, tenantId uuid.UUID) (*OpenDataDto, *shared.ErrorResponse) {
	var row, err = model.New(s.db).GetTenantOpenData(ctx, tenantId)
	if errors.Is(err, sql.ErrNoRows) {
		return &OpenDataDto{}, nil
	}
	if err != nil {
		return nil, failed("errors.open_data_load_failed")
	}
	var dto = &OpenDataDto{Enabled: row.Enabled, Dir3Code: row.Dir3Code, UpdatedAt: &row.UpdatedAt}
	if row.IssuedAt.Valid {
		dto.IssuedAt = &row.IssuedAt.Time
	}
	return dto, nil
}

func (s *counciladminService) SaveOpenData(ctx context.Context, tenantId uuid.UUID, input OpenDataInput) (*OpenDataDto, *shared.ErrorResponse) {
	if key := input.validate(); key != "" {
		return nil, &shared.ErrorResponse{Code: http.StatusUnprocessableEntity, Message: key}
	}
	var before, _ = s.GetOpenData(ctx, tenantId)
	var err = db.WithTenant(ctx, s.db, tenantId, func(queries *model.Queries) error {
		return queries.UpsertTenantOpenData(ctx, model.UpsertTenantOpenDataParams{TenantID: tenantId, Enabled: input.Enabled, Dir3Code: input.Dir3Code})
	})
	if err != nil {
		return nil, failed("errors.open_data_save_failed")
	}
	if before != nil {
		audit.Changed(ctx, "published", before.Enabled, input.Enabled)
		audit.Changed(ctx, "DIR3", before.Dir3Code, input.Dir3Code)
	}
	return s.GetOpenData(ctx, tenantId)
}

func (h *counciladminHandler) GetOpenData(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var response, err = GetCounciladminService(h.db).GetOpenData(r.Context(), tenantId)
	send(w, r, response, err)
}

func (h *counciladminHandler) SaveOpenData(w http.ResponseWriter, r *http.Request) {
	var tenantId, ok = h.councilAdmin(w, r)
	if !ok {
		return
	}
	var input OpenDataInput
	if !decode(w, r, &input) {
		return
	}
	var response, err = GetCounciladminService(h.db).SaveOpenData(r.Context(), tenantId, input)
	send(w, r, response, err)
}
