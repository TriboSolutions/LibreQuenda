package main

import (
	"context"
	"database/sql"
	"librequenda/core/agent"
	"os"
	_ "time/tzdata" // council timezones without relying on the image

	"librequenda/auth"
	"librequenda/branding"
	"librequenda/core/bookings"
	"librequenda/core/calendar"
	"librequenda/core/counciladmin"
	"librequenda/core/entrance"
	"librequenda/core/health"
	"librequenda/core/invites"
	"librequenda/core/issues"
	"librequenda/core/me"
	"librequenda/core/media"
	"librequenda/core/notifications"
	"librequenda/core/platform"
	"librequenda/core/polls"
	"librequenda/core/ratings"
	"librequenda/core/reader"
	"librequenda/core/tenants"
	"librequenda/cronJobs"
	"librequenda/db"
	"librequenda/mailer"
	"librequenda/shared"
	"librequenda/storage"

	"github.com/angelbarreiros/Penguin/router"
	"github.com/angelbarreiros/Penguin/router/helpers"
)

func main() {
	// ./app seed-media <dir>: upload demo images to the storage, then exit
	if len(os.Args) == 3 && os.Args[1] == "seed-media" {
		var uploaded, skipped, err = storage.SeedDirectory(context.Background(), os.Args[2])
		if err != nil {
			shared.Logger.Error("seed media failed: %v", err)
			shared.Logger.Flush() // the console logger writes in the background
			os.Exit(1)
		}
		shared.Logger.Info("seed media: %d uploaded, %d already stored", uploaded, skipped)
		shared.Logger.Flush()
		return
	}

	helpers.GetI18nInstance(helpers.SetDefaultLocale("gl"), helpers.SetDirectory("i18n"))

	var dbConexion *sql.DB = db.EstablishConnection()
	mailer.UseRegistry(dbConexion) // every email is written down in email_log
	auth.InitializeAuth()
	var router = router.InitRouter()
	cronJobs.InitializeCronJobs(dbConexion)
	branding.SyncZitadel(dbConexion) // Zitadel's emails and sign-in pages named and drawn like the app
	health.HealthRoutes(dbConexion, router).CreateHealthRoutes()
	me.MeRoutes(dbConexion, router).RegisterRoutes()
	agent.AgentRoutes(dbConexion, router).RegisterRoutes() // /api/mcp: agents look up, sign in and book
	tenants.TenantsRoutes(dbConexion, router).RegisterRoutes()
	bookings.BookingsRoutes(dbConexion, router).RegisterRoutes()
	invites.InvitesRoutes(dbConexion, router).RegisterRoutes()
	calendar.CalendarRoutes(dbConexion, router).RegisterRoutes()
	notifications.NotificationsRoutes(dbConexion, router).RegisterRoutes()
	platform.PlatformRoutes(dbConexion, router).RegisterRoutes()
	media.MediaRoutes(dbConexion, router).RegisterRoutes()
	counciladmin.CounciladminRoutes(dbConexion, router).RegisterRoutes()
	reader.ReaderRoutes(dbConexion, router).RegisterRoutes()
	entrance.EntranceRoutes(dbConexion, router).RegisterRoutes()
	issues.IssuesRoutes(dbConexion, router).RegisterRoutes()
	ratings.RatingsRoutes(dbConexion, router).RegisterRoutes()
	polls.PollsRoutes(dbConexion, router).RegisterRoutes()
	router.StartServer(":8000")
}
