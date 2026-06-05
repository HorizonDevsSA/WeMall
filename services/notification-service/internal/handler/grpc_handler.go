package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	notificationv1 "github.com/wemall/gen/notification/v1"
	"github.com/wemall/notification-service/internal/db"
)

type GRPCHandler struct {
	notificationv1.UnimplementedNotificationServiceServer
	q db.Querier
}

func NewGRPCHandler(queries db.Querier) *GRPCHandler {
	return &GRPCHandler{q: queries}
}

func (h *GRPCHandler) RegisterDeviceToken(ctx context.Context, req *notificationv1.RegisterDeviceTokenRequest) (*emptypb.Empty, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	var deviceName *string
	if req.DeviceName != "" {
		deviceName = &req.DeviceName
	}

	_, err = h.q.UpsertDeviceToken(ctx, db.UpsertDeviceTokenParams{
		UserID:     uid,
		Token:      req.Token,
		Platform:   req.Platform,
		DeviceName: deviceName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register device token: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) DeregisterDeviceToken(ctx context.Context, req *notificationv1.DeregisterDeviceTokenRequest) (*emptypb.Empty, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	err = h.q.DeleteDeviceToken(ctx, db.DeleteDeviceTokenParams{
		UserID: uid,
		Token:  req.Token,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to deregister device token: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (h *GRPCHandler) GetNotificationPreferences(ctx context.Context, req *notificationv1.GetNotificationPreferencesRequest) (*notificationv1.GetNotificationPreferencesResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	prefs, err := h.q.GetNotificationPreferences(ctx, uid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get notification preferences: %v", err)
	}

	// Default categories
	allCategories := map[db.NotificationCategory]bool{
		db.NotificationCategoryTransactional: true,
		db.NotificationCategorySecurity:      true,
		db.NotificationCategoryLowStock:      true,
		db.NotificationCategoryFollows:       true,
		db.NotificationCategoryMarketing:     true,
	}

	responsePrefs := make([]*notificationv1.NotificationPreference, 0, len(allCategories))

	// Map existing preferences
	for _, p := range prefs {
		responsePrefs = append(responsePrefs, &notificationv1.NotificationPreference{
			Category:     mapCategoryToProto(p.Category),
			EmailEnabled: p.EmailEnabled,
			PushEnabled:  p.PushEnabled,
		})
		delete(allCategories, p.Category)
	}

	// Add missing categories as default-enabled
	for cat := range allCategories {
		responsePrefs = append(responsePrefs, &notificationv1.NotificationPreference{
			Category:     mapCategoryToProto(cat),
			EmailEnabled: true,
			PushEnabled:  true,
		})
	}

	return &notificationv1.GetNotificationPreferencesResponse{
		Preferences: responsePrefs,
	}, nil
}

func (h *GRPCHandler) UpdateNotificationPreferences(ctx context.Context, req *notificationv1.UpdateNotificationPreferencesRequest) (*notificationv1.NotificationPreference, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	cat := mapProtoToCategory(req.Category)
	if cat == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid notification category")
	}

	pref, err := h.q.UpsertNotificationPreference(ctx, db.UpsertNotificationPreferenceParams{
		UserID:       uid,
		Category:     cat,
		EmailEnabled: req.EmailEnabled,
		PushEnabled:  req.PushEnabled,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update notification preference: %v", err)
	}

	return &notificationv1.NotificationPreference{
		Category:     mapCategoryToProto(pref.Category),
		EmailEnabled: pref.EmailEnabled,
		PushEnabled:  pref.PushEnabled,
	}, nil
}

func (h *GRPCHandler) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	limit := int32(20)
	if req.Limit > 0 {
		limit = req.Limit
	}
	offset := req.Offset

	logs, err := h.q.ListNotificationLogs(ctx, db.ListNotificationLogsParams{
		UserID: uid,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list notification logs: %v", err)
	}

	protoLogs := make([]*notificationv1.NotificationLog, len(logs))
	for i, l := range logs {
		protoLogs[i] = &notificationv1.NotificationLog{
			Id:        l.ID.String(),
			Category:  l.Category,
			Channel:   l.Channel,
			Title:     l.Title,
			Content:   l.Content,
			Status:    string(l.Status),
			CreatedAt: timestamppb.New(l.CreatedAt),
		}
	}

	return &notificationv1.ListNotificationsResponse{
		Notifications: protoLogs,
	}, nil
}

// ── Mappers ───────────────────────────────────────────────────────────────────

func mapCategoryToProto(c db.NotificationCategory) notificationv1.NotificationCategory {
	switch c {
	case db.NotificationCategoryTransactional:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_TRANSACTIONAL
	case db.NotificationCategorySecurity:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_SECURITY
	case db.NotificationCategoryLowStock:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_LOW_STOCK
	case db.NotificationCategoryFollows:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_FOLLOWS
	case db.NotificationCategoryMarketing:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_MARKETING
	default:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_UNSPECIFIED
	}
}

func mapProtoToCategory(c notificationv1.NotificationCategory) db.NotificationCategory {
	switch c {
	case notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_TRANSACTIONAL:
		return db.NotificationCategoryTransactional
	case notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_SECURITY:
		return db.NotificationCategorySecurity
	case notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_LOW_STOCK:
		return db.NotificationCategoryLowStock
	case notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_FOLLOWS:
		return db.NotificationCategoryFollows
	case notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_MARKETING:
		return db.NotificationCategoryMarketing
	default:
		return ""
	}
}
