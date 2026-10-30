package issue

import (
	"librequenda/model"
	"strings"
	"testing"
	"time"
)

func TestReportClean(t *testing.T) {
	var valid = func() Report {
		return Report{Category: "lighting", Description: "  The lights of court 2 do not turn on. "}
	}
	var cases = []struct {
		name   string
		change func(*Report)
		want   string
	}{
		{"valid", func(*Report) {}, ""},
		{"unknown category", func(r *Report) { r.Category = "noise" }, "errors.invalid_issue_category"},
		{"no description", func(r *Report) { r.Description = "   " }, "errors.invalid_issue_description"},
		{"description too long", func(r *Report) { r.Description = strings.Repeat("a", MaxDescription+1) }, "errors.invalid_issue_description"},
		{"photo of a facility", func(r *Report) { r.PhotoRef = "facilities/a.png" }, "errors.invalid_image"},
		{"photo of the issue", func(r *Report) { r.PhotoRef = "issues/a.png" }, ""},
	}
	for _, c := range cases {
		var r = valid()
		c.change(&r)
		if got := r.Clean(); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
	var r = valid()
	r.Clean()
	if r.Description != "The lights of court 2 do not turn on." {
		t.Errorf("description not trimmed: %q", r.Description)
	}
}

func TestReportable(t *testing.T) {
	var ended = time.Date(2026, 11, 2, 19, 0, 0, 0, time.UTC)
	if !Reportable(ended, ended.Add(6*24*time.Hour)) || Reportable(ended, ended.Add(8*24*time.Hour)) {
		t.Error("a booking can be reported from for a week after it ends")
	}
}

func TestChangeAndNotice(t *testing.T) {
	if got := Change(model.IssueStatusRejected, " "); got != "errors.issue_reject_reason" {
		t.Errorf("rejecting needs a reason, got %q", got)
	}
	if got := Change(model.IssueStatus("closed"), ""); got != "errors.invalid_issue_status" {
		t.Errorf("unknown status, got %q", got)
	}
	if got := Change(model.IssueStatusSolved, ""); got != "" {
		t.Errorf("solving needs no note, got %q", got)
	}
	var cases = []struct {
		from, to model.IssueStatus
		want     model.NotificationType
	}{
		{model.IssueStatusNew, model.IssueStatusInProgress, ""},
		{model.IssueStatusInProgress, model.IssueStatusSolved, model.NotificationTypeIssueSolved},
		{model.IssueStatusNew, model.IssueStatusRejected, model.NotificationTypeIssueRejected},
		{model.IssueStatusSolved, model.IssueStatusSolved, ""}, // saving the note again tells nobody twice
		{model.IssueStatusSolved, model.IssueStatusInProgress, ""},
	}
	for _, c := range cases {
		if got := Notice(c.from, c.to); got != c.want {
			t.Errorf("%s -> %s: got %q, want %q", c.from, c.to, got, c.want)
		}
	}
}
