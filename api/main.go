package main

import (
	"context"
	"database/sql"
	"os"
	_ "time/tzdata" // council timezones without relying on the image

	"librequenda/auth"
	"librequenda/core/bookings"
	"librequenda/core/counciladmin"
	"librequenda/core/health"
	"librequenda/core/invites"
	"librequenda/core/me"
	"librequenda/core/media"
	"librequenda/core/notifications"
	"librequenda/core/platform"
	"librequenda/core/reader"
	"librequenda/core/tenants"
	"librequenda/cronJobs"
	"librequenda/db"
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
	auth.InitializeAuth()
	var router = router.InitRouter()
	cronJobs.InitializeCronJobs(dbConexion)
	health.HealthRoutes(dbConexion, router).CreateHealthRoutes()
	me.MeRoutes(dbConexion, router).RegisterRoutes()
	tenants.TenantsRoutes(dbConexion, router).RegisterRoutes()
	bookings.BookingsRoutes(dbConexion, router).RegisterRoutes()
	invites.InvitesRoutes(dbConexion, router).RegisterRoutes()
	notifications.NotificationsRoutes(dbConexion, router).RegisterRoutes()
	platform.PlatformRoutes(dbConexion, router).RegisterRoutes()
	media.MediaRoutes(dbConexion, router).RegisterRoutes()
	counciladmin.CounciladminRoutes(dbConexion, router).RegisterRoutes()
	reader.ReaderRoutes(dbConexion, router).RegisterRoutes()
	router.StartServer(":8000")
}
