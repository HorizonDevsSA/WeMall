package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/wemall/notification-service/internal/db"
	"github.com/wemall/notification-service/internal/providers/email"
	"github.com/wemall/notification-service/internal/providers/push"
)

type Worker struct {
	server *asynq.Server
	dbPool *pgxpool.Pool
	q      *db.Queries
	smtp   *email.SMTPProvider
	fcm    *push.FCMProvider
	logger zerolog.Logger
}

func NewWorker(redisAddr string, concurrency int, dbPool *pgxpool.Pool, smtp *email.SMTPProvider, fcm *push.FCMProvider, logger zerolog.Logger) *Worker {
	var opt asynq.RedisConnOpt
	var err error
	if len(redisAddr) >= 8 && (redisAddr[:8] == "redis://" || redisAddr[:9] == "rediss://") {
		opt, err = asynq.ParseRedisURI(redisAddr)
		if err != nil {
			logger.Error().Err(err).Msg("failed to parse Redis URI, using default localhost:6379")
			opt = asynq.RedisClientOpt{Addr: "localhost:6379"}
		}
	} else {
		opt = asynq.RedisClientOpt{Addr: redisAddr}
	}

	server := asynq.NewServer(
		opt,
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.Error().Err(err).Str("task", task.Type()).Msg("Task processing failed")
			}),
		},
	)

	return &Worker{
		server: server,
		dbPool: dbPool,
		q:      db.New(dbPool),
		smtp:   smtp,
		fcm:    fcm,
		logger: logger,
	}
}

func (w *Worker) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypeEmailSend, w.handleEmailSendTask)
	mux.HandleFunc(TypePushSend, w.handlePushSendTask)
	mux.HandleFunc(TypePushMulticast, w.handlePushMulticastTask)

	w.logger.Info().Msg("Starting Asynq worker server...")
	return w.server.Run(mux)
}

func (w *Worker) Shutdown() {
	w.logger.Info().Msg("Shutting down Asynq worker server...")
	w.server.Shutdown()
}

func (w *Worker) handleEmailSendTask(ctx context.Context, t *asynq.Task) error {
	var payload EmailSendPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	w.logger.Info().Str("to", payload.Recipient).Str("subject", payload.Subject).Msg("Processing email send task")

	uid, err := uuid.Parse(payload.UserID)
	if err != nil {
		return err
	}

	// 1. Create notification log in DB
	logEntry, err := w.q.CreateNotificationLog(ctx, db.CreateNotificationLogParams{
		UserID:     uid,
		Category:   payload.Category,
		Channel:    "email",
		Recipient:  payload.Recipient,
		Title:      payload.Subject,
		Content:    payload.HTMLBody,
		Status:     db.DeliveryStatusQueued,
		RetryCount: 0,
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to create email notification log")
		return err
	}

	// Get info about retries
	retries, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)

	// 2. Dispatch email
	err = w.smtp.SendEmail(payload.Recipient, payload.Subject, payload.HTMLBody)
	if err != nil {
		// Log failure in DB
		status := db.DeliveryStatusRetrying
		if retries >= maxRetry {
			status = db.DeliveryStatusFailed
		}
		errMsg := err.Error()

		_ = w.q.UpdateNotificationLogStatus(ctx, db.UpdateNotificationLogStatusParams{
			ID:           logEntry.ID,
			Status:       status,
			RetryCount:   int32(retries),
			ErrorMessage: &errMsg,
		})

		w.logger.Error().Err(err).Str("to", payload.Recipient).Msg("Failed to send email")
		return err
	}

	// 3. Log Success
	now := time.Now()
	err = w.q.UpdateNotificationLogStatus(ctx, db.UpdateNotificationLogStatusParams{
		ID:         logEntry.ID,
		Status:     db.DeliveryStatusSent,
		RetryCount: int32(retries),
		SentAt:     pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to update email log status to sent")
	}

	return nil
}

func (w *Worker) handlePushSendTask(ctx context.Context, t *asynq.Task) error {
	var payload PushSendPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	w.logger.Info().Str("token", payload.Token).Str("title", payload.Title).Msg("Processing push send task")

	uid, err := uuid.Parse(payload.UserID)
	if err != nil {
		return err
	}

	// 1. Create notification log in DB
	logEntry, err := w.q.CreateNotificationLog(ctx, db.CreateNotificationLogParams{
		UserID:     uid,
		Category:   payload.Category,
		Channel:    "push",
		Recipient:  payload.Token,
		Title:      payload.Title,
		Content:    payload.Body,
		Status:     db.DeliveryStatusQueued,
		RetryCount: 0,
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to create push notification log")
		return err
	}

	retries, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)

	// 2. Dispatch push
	msgID, err := w.fcm.SendPush(ctx, &push.PushMessage{
		Token:    payload.Token,
		Title:    payload.Title,
		Body:     payload.Body,
		Data:     payload.Data,
		Category: payload.Category,
	})
	if err != nil {
		// Log failure in DB
		status := db.DeliveryStatusRetrying
		if retries >= maxRetry {
			status = db.DeliveryStatusFailed
		}
		errMsg := err.Error()

		_ = w.q.UpdateNotificationLogStatus(ctx, db.UpdateNotificationLogStatusParams{
			ID:           logEntry.ID,
			Status:       status,
			RetryCount:   int32(retries),
			ErrorMessage: &errMsg,
		})

		// Token Pruning on unregistered/invalid errors
		if stringsContainsAny(errMsg, "unregistered", "invalid-argument", "INVALID_ARGUMENT", "UNREGISTERED") {
			w.logger.Warn().Str("token", payload.Token).Msg("Pruning invalid device token from user")
			_ = w.q.DeleteDeviceToken(ctx, db.DeleteDeviceTokenParams{
				UserID: uid,
				Token:  payload.Token,
			})
		}

		w.logger.Error().Err(err).Str("token", payload.Token).Msg("Failed to send push")
		return err
	}

	// 3. Log Success
	now := time.Now()
	_ = w.q.UpdateNotificationLogStatus(ctx, db.UpdateNotificationLogStatusParams{
		ID:         logEntry.ID,
		Status:     db.DeliveryStatusSent,
		RetryCount: int32(retries),
		SentAt:     pgtype.Timestamptz{Time: now, Valid: true},
	})

	// 4. Create Push Notification Inbox entry
	fullPayload := map[string]interface{}{
		"message_id": msgID,
		"token":      payload.Token,
		"title":      payload.Title,
		"body":       payload.Body,
		"data":       payload.Data,
	}
	payloadBytes, _ := json.Marshal(fullPayload)

	_, err = w.q.CreatePushNotification(ctx, db.CreatePushNotificationParams{
		UserID:  uid,
		Token:   payload.Token,
		Title:   payload.Title,
		Body:    payload.Body,
		Payload: payloadBytes,
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to create push_notifications inbox entry")
	}

	return nil
}

func (w *Worker) handlePushMulticastTask(ctx context.Context, t *asynq.Task) error {
	var payload PushMulticastPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	w.logger.Info().Int("tokens", len(payload.Tokens)).Str("title", payload.Title).Msg("Processing multicast push task")

	if len(payload.Tokens) == 0 {
		return nil
	}

	// Pre-create pending logs for audit trail
	logIDs := make([]uuid.UUID, len(payload.Tokens))
	for idx, token := range payload.Tokens {
		uid, err := uuid.Parse(payload.UserIDs[idx])
		if err != nil {
			continue
		}
		logEntry, err := w.q.CreateNotificationLog(ctx, db.CreateNotificationLogParams{
			UserID:     uid,
			Category:   payload.Category,
			Channel:    "push",
			Recipient:  token,
			Title:      payload.Title,
			Content:    payload.Body,
			Status:     db.DeliveryStatusQueued,
			RetryCount: 0,
		})
		if err == nil {
			logIDs[idx] = logEntry.ID
		}
	}

	// Send multicast via FCM
	failedTokens, err := w.fcm.SendMulticast(ctx, payload.Tokens, payload.Title, payload.Body, payload.Data)
	if err != nil {
		w.logger.Error().Err(err).Msg("Multicast send returned error")
		return err
	}

	// Create a map of failed tokens for quick lookups
	failedMap := make(map[string]bool)
	for _, ft := range failedTokens {
		failedMap[ft] = true
	}

	// Update logs and prune invalid tokens
	now := time.Now()
	for idx, token := range payload.Tokens {
		uid, err := uuid.Parse(payload.UserIDs[idx])
		if err != nil {
			continue
		}

		logID := logIDs[idx]
		if logID == uuid.Nil {
			continue
		}

		if failedMap[token] {
			errMsg := "FCM delivery failed (permanent)"
			_ = w.q.UpdateNotificationLogStatus(ctx, db.UpdateNotificationLogStatusParams{
				ID:           logID,
				Status:       db.DeliveryStatusFailed,
				RetryCount:   0,
				ErrorMessage: &errMsg,
			})

			// Prune invalid token
			w.logger.Warn().Str("token", token).Msg("Pruning invalid multicast device token")
			_ = w.q.DeleteDeviceToken(ctx, db.DeleteDeviceTokenParams{
				UserID: uid,
				Token:  token,
			})
		} else {
			_ = w.q.UpdateNotificationLogStatus(ctx, db.UpdateNotificationLogStatusParams{
				ID:         logID,
				Status:     db.DeliveryStatusSent,
				RetryCount: 0,
				SentAt:     pgtype.Timestamptz{Time: now, Valid: true},
			})

			// Add to push notifications inbox history
			fullPayload := map[string]interface{}{
				"token": token,
				"title": payload.Title,
				"body":  payload.Body,
				"data":  payload.Data,
			}
			payloadBytes, _ := json.Marshal(fullPayload)

			_, err = w.q.CreatePushNotification(ctx, db.CreatePushNotificationParams{
				UserID:  uid,
				Token:   token,
				Title:   payload.Title,
				Body:    payload.Body,
				Payload: payloadBytes,
			})
			if err != nil {
				w.logger.Error().Err(err).Msg("Failed to create push_notifications inbox entry")
			}
		}
	}

	return nil
}

func stringsContainsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if stringsContains(s, sub) {
			return true
		}
	}
	return false
}

func stringsContains(s, sub string) bool {
	// Simple case-insensitive contains
	lenS := len(s)
	lenSub := len(sub)
	if lenSub > lenS {
		return false
	}
	for i := 0; i <= lenS-lenSub; i++ {
		match := true
		for j := 0; j < lenSub; j++ {
			c1 := s[i+j]
			c2 := sub[j]
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func fmtStr(format string, args ...interface{}) string {
	// Custom fallback for fmt.Errorf formatting
	return ""
}
