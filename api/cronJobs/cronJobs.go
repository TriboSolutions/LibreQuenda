package cronJobs

import (
	"database/sql"
	"log"

	"github.com/angelbarreiros/Penguin/scheduler"
)

func InitializeCronJobs(db *sql.DB) {
	var cronScheduler *scheduler.Scheduler = scheduler.StartScheduler()

	registerBookingReminderCronJob(db, cronScheduler)
	registerRatingRequestCronJob(db, cronScheduler)
	registerNotificationEmailCronJob(db, cronScheduler)
	registerForgetBookingHoldersCronJob(db, cronScheduler)
	registerAuditRetentionCronJob(db, cronScheduler)
	log.Println("Cron jobs initialized")
}
