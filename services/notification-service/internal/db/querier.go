package db

import (
	"context"

	"github.com/google/uuid"
)

type Querier interface {
	CreateNotificationLog(ctx context.Context, arg CreateNotificationLogParams) (NotificationLog, error)
	CreatePushNotification(ctx context.Context, arg CreatePushNotificationParams) (PushNotification, error)
	DeleteDeviceToken(ctx context.Context, arg DeleteDeviceTokenParams) error
	GetDeviceTokensByUser(ctx context.Context, userID uuid.UUID) ([]UserDeviceToken, error)
	GetDeviceTokensByUsers(ctx context.Context, userIDs []uuid.UUID) ([]UserDeviceToken, error)
	GetNotificationPreference(ctx context.Context, arg GetNotificationPreferenceParams) (UserNotificationPreference, error)
	GetNotificationPreferences(ctx context.Context, userID uuid.UUID) ([]UserNotificationPreference, error)
	ListNotificationLogs(ctx context.Context, arg ListNotificationLogsParams) ([]NotificationLog, error)
	ListPushNotifications(ctx context.Context, arg ListPushNotificationsParams) ([]PushNotification, error)
	MarkPushNotificationRead(ctx context.Context, arg MarkPushNotificationReadParams) error
	UpdateNotificationLogStatus(ctx context.Context, arg UpdateNotificationLogStatusParams) error
	UpsertDeviceToken(ctx context.Context, arg UpsertDeviceTokenParams) (UserDeviceToken, error)
	UpsertNotificationPreference(ctx context.Context, arg UpsertNotificationPreferenceParams) (UserNotificationPreference, error)
}
