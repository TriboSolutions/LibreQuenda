package cronJobs

import (
	"context"
	"database/sql"
	"librequenda/model"
	"log"
	"time"

	"github.com/angelbarreiros/Penguin/scheduler"
)

const forgetBookingHoldersInterval = time.Hour

type forgetBookingHoldersJob struct {
	db *sql.DB
}

// registerForgetBookingHoldersCronJob: RXPD. The name and phone that staff take
// for someone without an account are removed 30 days after the booking ends.
func registerForgetBookingHoldersCronJob(db *sql.DB, cronScheduler *scheduler.Scheduler) {
	var _, err = cronScheduler.ScheduleIntervalJob(
		forgetBookingHoldersInterval,
		scheduler.JobFunction(&forgetBookingHoldersJob{db: db}),
		scheduler.JobOptions{
			WaitForCompletion: true,
		},
	)
	if err != nil {
		log.Printf("failed to schedule forget booking holders cron job: %v", err)
	}
}

func (j *forgetBookingHoldersJob) Execute() []any {
	var ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var forgotten, err = model.New(j.db).ForgetBookingHolders(ctx)
	if err != nil {
		log.Printf("failed to forget booking holders: %v", err)
		return []any{err}
	}
	if forgotten > 0 {
		log.Printf("forgot the name and phone of %d bookings", forgotten)
	}
	return []any{forgotten}
}
