package tenants

import (
	"context"
	"librequenda/model"
	"math"
	"time"

	"github.com/google/uuid"
)

// PublicNoticeDto: one of the council's announcements, on its pages until it expires.
type PublicNoticeDto struct {
	Id           string    `json:"id"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	FacilityName string    `json:"facility_name"` // "" when it is for the whole council
	FacilitySlug string    `json:"facility_slug"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// CouncilPageDto: the council and its current announcements.
type CouncilPageDto struct {
	TenantDto
	Notices []PublicNoticeDto `json:"notices"`
}

// KnownIssueDto: problems reported and not fixed yet, by category, without
// what anyone wrote or who reported them.
type KnownIssueDto struct {
	Category        string    `json:"category"`
	Reports         int32     `json:"reports"`
	FirstReportedAt time.Time `json:"first_reported_at"`
	InProgress      bool      `json:"in_progress"` // the council is on it
	Urgent          bool      `json:"urgent"`
}

type SolvedIssueDto struct {
	Category string    `json:"category"`
	Reports  int32     `json:"reports"`
	SolvedAt time.Time `json:"solved_at"`
}

// IssuesSummaryDto: what is known to be broken at a facility, what was fixed
// in the last 30 days, and how long the council takes, on average, to fix
// what residents report (over 180 days; null before it solved any).
type IssuesSummaryDto struct {
	Open               []KnownIssueDto  `json:"open"`
	RecentlySolved     []SolvedIssueDto `json:"recently_solved"`
	AverageDaysToSolve *float64         `json:"average_days_to_solve"`
	SolvedCount        int32            `json:"solved_count"`
}

func (s *tenantsService) facilityNotices(ctx context.Context, tenantId, facilityId uuid.UUID) ([]PublicNoticeDto, error) {
	var rows, err = model.New(s.db).ListActiveFacilityNotices(ctx, model.ListActiveFacilityNoticesParams{TenantID: tenantId, FacilityID: uuid.NullUUID{UUID: facilityId, Valid: true}})
	if err != nil {
		return nil, err
	}
	var out = make([]PublicNoticeDto, 0, len(rows))
	for _, row := range rows {
		out = append(out, PublicNoticeDto{Id: row.ID.String(), Title: row.Title, Body: row.Body, ExpiresAt: row.ExpiresAt, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (s *tenantsService) councilNotices(ctx context.Context, tenantId uuid.UUID) ([]PublicNoticeDto, error) {
	var rows, err = model.New(s.db).ListActiveCouncilNotices(ctx, tenantId)
	if err != nil {
		return nil, err
	}
	var out = make([]PublicNoticeDto, 0, len(rows))
	for _, row := range rows {
		out = append(out, PublicNoticeDto{
			Id: row.ID.String(), Title: row.Title, Body: row.Body, FacilityName: row.FacilityName, FacilitySlug: row.FacilitySlug,
			ExpiresAt: row.ExpiresAt, CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (s *tenantsService) issuesSummary(ctx context.Context, tenantId, facilityId uuid.UUID) (*IssuesSummaryDto, error) {
	var queries = model.New(s.db)
	var open, err = queries.ListOpenFacilityIssues(ctx, facilityId)
	if err != nil {
		return nil, err
	}
	var solved, solvedErr = queries.ListRecentlySolvedFacilityIssues(ctx, facilityId)
	if solvedErr != nil {
		return nil, solvedErr
	}
	var resolution, resolutionErr = queries.CouncilIssueResolution(ctx, tenantId)
	if resolutionErr != nil {
		return nil, resolutionErr
	}
	var summary = &IssuesSummaryDto{Open: make([]KnownIssueDto, 0, len(open)), RecentlySolved: make([]SolvedIssueDto, 0, len(solved)), SolvedCount: resolution.Solved}
	for _, row := range open {
		summary.Open = append(summary.Open, KnownIssueDto{Category: string(row.Category), Reports: row.Reports, FirstReportedAt: row.FirstReportedAt, InProgress: row.InProgress, Urgent: row.Urgent})
	}
	for _, row := range solved {
		summary.RecentlySolved = append(summary.RecentlySolved, SolvedIssueDto{Category: string(row.Category), Reports: row.Reports, SolvedAt: row.SolvedAt})
	}
	if resolution.Solved > 0 {
		summary.AverageDaysToSolve = averageDays(resolution.AverageDays)
	}
	return summary, nil
}

// averageDays rounds to one decimal, and never says less than 0.1 days.
func averageDays(days float64) *float64 {
	var rounded = math.Max(math.Round(days*10)/10, 0.1)
	return &rounded
}
