package cronJobs

import (
	"database/sql"
	"log"

	"github.com/angelbarreiros/Penguin/scheduler"
)

func InitializeCronJobs(db *sql.DB) {
	var cronScheduler *scheduler.Scheduler = scheduler.StartScheduler()

	registerBookingReminderCronJob(db, cronScheduler)
	registerNotificationEmailCronJob(db, cronScheduler)
	log.Println("Cron jobs initialized")
}
