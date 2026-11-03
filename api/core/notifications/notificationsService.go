package notifications

import (
	"context"
	"database/sql"
	"github.com/angelbarreiros/Penguin/router/helpers"
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

// GetNotifications: a page of the bell's notices (?page=&pageSize=, 100 by default), most recent first.
func (s *notificationsService) GetNotifications(ctx context.Context, userId uuid.UUID, page helpers.PaginationParams) (*NotificationsResponse, *shared.ErrorResponse) {
	var failed = &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.notifications_load_failed"}
	var queries = model.New(s.db)
	var rows, err = queries.ListNotificationsByUser(ctx, model.ListNotificationsByUserParams{UserID: userId, MaxRows: int32(page.PageSize), Skip: int32(page.Offset())})
	if err != nil {
		return nil, failed
	}
	var unread, total int64
	if unread, err = queries.CountUnreadNotifications(ctx, userId); err != nil {
		return nil, failed
	}
	if total, err = queries.CountNotificationsByUser(ctx, userId); err != nil {
		return nil, failed
	}
	var response = MapNotifications(rows, unread)
	response.Pagination = shared.NewPageDto(page, total)
	return response, nil
}

func (s *notificationsService) MarkAllRead(ctx context.Context, userId uuid.UUID) *shared.ErrorResponse {
	if err := model.New(s.db).MarkNotificationsRead(ctx, userId); err != nil {
		return &shared.ErrorResponse{Code: http.StatusInternalServerError, Message: "errors.notifications_update_failed"}
	}
	return nil
}
