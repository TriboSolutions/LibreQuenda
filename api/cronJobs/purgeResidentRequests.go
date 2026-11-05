package cronJobs

import (
	"context"
	"database/sql"
	"librequenda/core/me"
	"librequenda/model"
	"log"
	"time"

	"github.com/angelbarreiros/Penguin/scheduler"
)

const purgeResidentRequestsInterval = time.Hour

type purgeResidentRequestsJob struct {
	db *sql.DB
}

// registerPurgeResidentRequestsCronJob: GDPR storage limitation. A request to
// be verified as a resident carries a DNI and an address; once the council
// decides it is deleted at once, and one nobody decided on goes after
// me.ResidentRequestRetentionDays (checked every hour).
func registerPurgeResidentRequestsCronJob(db *sql.DB, cronScheduler *scheduler.Scheduler) {
	var _, err = cronScheduler.ScheduleIntervalJob(
		purgeResidentRequestsInterval,
		scheduler.JobFunction(&purgeResidentRequestsJob{db: db}),
		scheduler.JobOptions{
			WaitForCompletion: true,
		},
	)
	if err != nil {
		log.Printf("failed to schedule resident request purge cron job: %v", err)
	}
}

func (j *purgeResidentRequestsJob) Execute() []any {
	var ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var purged, err = model.New(j.db).PurgeResidentRequests(ctx, me.ResidentRequestRetentionDays)
	if err != nil {
		log.Printf("failed to purge resident requests: %v", err)
		return []any{err}
	}
	if purged > 0 {
		log.Printf("purged %d resident requests past %d days", purged, me.ResidentRequestRetentionDays)
	}
	return []any{purged}
}
