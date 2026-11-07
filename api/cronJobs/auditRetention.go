package cronJobs

import (
	"context"
	"database/sql"
	"librequenda/audit"
	"librequenda/emaillog"
	"librequenda/model"
	"log"
	"os"
	"time"

	"github.com/angelbarreiros/Penguin/scheduler"
)

const auditRetentionInterval = 24 * time.Hour

type auditRetentionJob struct {
	db *sql.DB
}

// registerAuditRetentionCronJob: GDPR storage limitation. The audit log keeps
// entries for AUDIT_RETENTION_DAYS (two years by default); older ones go every day.
func registerAuditRetentionCronJob(db *sql.DB, cronScheduler *scheduler.Scheduler) {
	var _, err = cronScheduler.ScheduleIntervalJob(
		auditRetentionInterval,
		scheduler.JobFunction(&auditRetentionJob{db: db}),
		scheduler.JobOptions{
			WaitForCompletion: true,
		},
	)
	if err != nil {
		log.Printf("failed to schedule audit retention cron job: %v", err)
	}
}

func (j *auditRetentionJob) Execute() []any {
	var ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var purged, err = model.New(j.db).PurgeAuditLog(ctx, int32(audit.Retention(os.Getenv("AUDIT_RETENTION_DAYS"))))
	if err != nil {
		log.Printf("failed to purge the audit log: %v", err)
		return []any{err}
	}
	if purged > 0 {
		log.Printf("purged %d audit log entries past the retention period", purged)
	}
	// the email log too: EMAIL_LOG_RETENTION_DAYS (a year by default)
	var emails int64
	if emails, err = model.New(j.db).PurgeEmailLog(ctx, int32(emaillog.Retention(os.Getenv("EMAIL_LOG_RETENTION_DAYS")))); err != nil {
		log.Printf("failed to purge the email log: %v", err)
		return []any{err}
	}
	if emails > 0 {
		log.Printf("purged %d email log entries past the retention period", emails)
	}
	return []any{purged, emails}
}
