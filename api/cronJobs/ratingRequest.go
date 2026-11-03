package cronJobs

import (
	"context"
	"database/sql"
	"librequenda/model"
	"log"
	"time"

	"github.com/angelbarreiros/Penguin/scheduler"
)

// every minute: the request arrives soon after the booking ends
const ratingRequestInterval = time.Minute

type ratingRequestJob struct {
	db *sql.DB
}

// registerRatingRequestCronJob asks everyone with an account in a booking that
// just ended for the facility's stars (and, after a public one, the others').
func registerRatingRequestCronJob(db *sql.DB, cronScheduler *scheduler.Scheduler) {
	var _, err = cronScheduler.ScheduleIntervalJob(
		ratingRequestInterval,
		scheduler.JobFunction(&ratingRequestJob{db: db}),
		scheduler.JobOptions{
			WaitForCompletion: true,
		},
	)
	if err != nil {
		log.Printf("failed to schedule rating request cron job: %v", err)
	}
}

func (j *ratingRequestJob) Execute() []any {
	var ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var created, err = model.New(j.db).CreateRatingRequests(ctx)
	if err != nil {
		log.Printf("failed to create rating requests: %v", err)
		return []any{err}
	}
	if created > 0 {
		log.Printf("created %d rating requests", created)
	}
	return []any{created}
}
