package cronJobs

import (
	"context"
	"database/sql"
	"librequenda/model"
	"log"
	"time"

	"github.com/angelbarreiros/Penguin/scheduler"
)

// every minute: reminders go out 15 minutes before the start
const bookingReminderInterval = time.Minute

type bookingReminderJob struct {
	db *sql.DB
}

// registerBookingReminderCronJob creates a "booking_reminder" notification
// for everyone in a confirmed booking that starts within the next 15 minutes.
func registerBookingReminderCronJob(db *sql.DB, cronScheduler *scheduler.Scheduler) {
	var err error

	_, err = cronScheduler.ScheduleIntervalJob(
		bookingReminderInterval,
		scheduler.JobFunction(&bookingReminderJob{db: db}),
		scheduler.JobOptions{
			WaitForCompletion: true,
		},
	)
	if err != nil {
		log.Printf("failed to schedule booking reminder cron job: %v", err)
	}
}

func (j *bookingReminderJob) Execute() []any {
	var ctx context.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var createdReminders int64
	var err error
	createdReminders, err = CreateBookingReminders(ctx, j.db)
	if err != nil {
		log.Printf("failed to create booking reminders: %v", err)
		return []any{err}
	}

	if createdReminders > 0 {
		log.Printf("created %d booking reminders", createdReminders)
	}

	return []any{createdReminders}
}

func CreateBookingReminders(ctx context.Context, db *sql.DB) (int64, error) {
	var queries *model.Queries = model.New(db)
	var createdReminders int64
	var err error
	createdReminders, err = queries.CreateBookingReminders(ctx)
	if err != nil {
		return 0, err
	}

	return createdReminders, nil
}
