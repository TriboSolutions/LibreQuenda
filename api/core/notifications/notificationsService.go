package notifications

import (
	"context"
	"database/sql"
	"librequenda/model"
	"librequenda/shared"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

type notificationsService struct {
	db *sql.DB
}

var sNotificationsInstance *notificationsService
var sNotificationsOnce *sync.Once = new(sync.Once)

func GetNotificationsService(db *sql.DB) *notificationsService {
	sNotificationsOnce.Do(func() {
		sNotificationsInstance = &notificationsService{db: db}
	})
	return sNotificationsInstance
}

const listLimit = 30

func (s *notificationsService) GetNotifications(ctx context.Context, userId uuid.UUID) (*NotificationsResponse, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.notifications_load_failed"}
	var queries = model.New(s.db)
	var rows, err = queries.ListNotificationsByUser(ctx, model.ListNotificationsByUserParams{UserID: userId, MaxRows: listLimit})
	if err != nil {
		return nil, failed
	}
	var unread int64
	if unread, err = queries.CountUnreadNotifications(ctx, userId); err != nil {
		return nil, failed
	}
	return MapNotifications(rows, unread), nil
}

func (s *notificationsService) MarkAllRead(ctx context.Context, userId uuid.UUID) *shared.ErrorResponse {
	if err := model.New(s.db).MarkNotificationsRead(ctx, userId); err != nil {
		return &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.notifications_update_failed"}
	}
	return nil
}
