// Package issue holds the rules for the problems residents report at a
// facility (broken net, no light…) and the council's maintenance list.
package issue

import (
	"librequenda/model"
	"librequenda/storage"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxDescription = 1000 // characters, for the description and the staff note
	MaxOpenReports = 5    // new or in progress, per person
	// ReportWindow: how long after a booking ends it can still be reported from
	ReportWindow = 7 * 24 * time.Hour
)

var categories = map[model.IssueCategory]bool{
	model.IssueCategoryEquipment: true, model.IssueCategoryLighting: true, model.IssueCategoryCleaning: true,
	model.IssueCategoryChangingRooms: true, model.IssueCategoryAccessibility: true, model.IssueCategorySafety: true,
	model.IssueCategoryOther: true,
}

// Report: what a resident sends.
type Report struct {
	FacilityID   string `json:"facility_id"`
	BookingID    string `json:"booking_id"` // optional: the booking it is reported from
	Category     string `json:"category"`
	Description  string `json:"description"`
	PhotoRef     string `json:"photo_ref"` // optional, a storage reference under issues/
	Urgent       bool   `json:"urgent"`
	ShareContact bool   `json:"share_contact"` // the council's staff may see their email
}

// Clean trims the report and checks it; the error is an i18n key.
func (r *Report) Clean() string {
	r.Description = strings.TrimSpace(r.Description)
	r.PhotoRef = strings.TrimSpace(r.PhotoRef)
	switch {
	case !categories[model.IssueCategory(r.Category)]:
		return "errors.invalid_issue_category"
	case r.Description == "" || utf8.RuneCountInString(r.Description) > MaxDescription:
		return "errors.invalid_issue_description"
	case r.PhotoRef != "" && (!storage.ValidRef(r.PhotoRef) || !strings.HasPrefix(r.PhotoRef, "issues/")):
		return "errors.invalid_image"
	}
	return ""
}

// Reportable: a booking can be reported from until ReportWindow after it ends.
func Reportable(bookingEnds, now time.Time) bool {
	return now.Before(bookingEnds.Add(ReportWindow))
}

// Final: solved or rejected, the reporter is told.
func Final(status model.IssueStatus) bool {
	return status == model.IssueStatusSolved || status == model.IssueStatusRejected
}

// Change checks a status change by the council's staff; the error is an i18n
// key. Any status may follow any other (a solved problem can come back), but
// rejecting needs a reason for the reporter.
func Change(to model.IssueStatus, note string) string {
	switch {
	case to != model.IssueStatusNew && to != model.IssueStatusInProgress && !Final(to):
		return "errors.invalid_issue_status"
	case utf8.RuneCountInString(note) > MaxDescription:
		return "errors.invalid_issue_note"
	case to == model.IssueStatusRejected && strings.TrimSpace(note) == "":
		return "errors.issue_reject_reason"
	}
	return ""
}

// Notice: the reporter's notification type when a change closes the issue,
// "" when it does not (it was already closed the same way, or it stays open).
func Notice(from, to model.IssueStatus) model.NotificationType {
	if from == to || !Final(to) {
		return ""
	}
	if to == model.IssueStatusSolved {
		return model.NotificationTypeIssueSolved
	}
	return model.NotificationTypeIssueRejected
}
