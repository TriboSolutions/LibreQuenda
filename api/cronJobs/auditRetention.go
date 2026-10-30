package cronJobs

import (
	"context"
	"database/sql"
	"librequenda/audit"
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
	return []any{purged}
}
