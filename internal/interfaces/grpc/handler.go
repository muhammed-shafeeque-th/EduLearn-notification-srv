package grpc_interface

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/services"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/metrics"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/proto/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler implements the NotificationServiceServer interface.
type Handler struct {
	proto.UnimplementedNotificationServiceServer
	notificationService   *services.NotificationService
	emailOtpService       *services.EmailOTPService
	forgotPasswordService *services.ForgotPasswordService
	logger                *zap.Logger
	validator             *validator.Validate
}

// NewHandler initializes and returns a new Handler.
func NewHandler(
	notificationService *services.NotificationService,
	emailOtpService *services.EmailOTPService,
	forgotPasswordService *services.ForgotPasswordService,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		notificationService:   notificationService,
		emailOtpService:       emailOtpService,
		forgotPasswordService: forgotPasswordService,
		logger:                logger,
		validator:             validator.New(),
	}
}

// validateRequest validates the incoming request using the validator package.
func (h *Handler) validateRequest(req interface{}) error {
	if err := h.validator.Struct(req); err != nil {
		h.logger.Warn("Request validation failed", zap.Error(err))
		return status.Errorf(codes.InvalidArgument, "Invalid request data: %v", err)
	}
	return nil
}

// createErrorResponse constructs a NotificationResponse for errors.
func (h *Handler) createErrorResponse(code codes.Code, message string, details ...string) *proto.NotificationResponse {
	errorDetails := make([]*proto.ErrorDetail, len(details))
	for i, d := range details {
		errorDetails[i] = &proto.ErrorDetail{Message: d}
	}
	return &proto.NotificationResponse{
		Result: &proto.NotificationResponse_Error{
			Error: &proto.Error{
				Code:    code.String(),
				Message: message,
				Details: errorDetails,
			},
		},
	}
}

// createSuccessResponse creates a NotificationResponse for successes.
func (h *Handler) createSuccessResponse(message string) *proto.NotificationResponse {
	return &proto.NotificationResponse{
		Result: &proto.NotificationResponse_Success{
			Success: &proto.NotificationSuccess{
				Success: true,
				Message: message,
			},
		},
	}
}

// SendOTP handles the sending of OTPs.
func (h *Handler) SendOTP(ctx context.Context, req *proto.OTPRequest) (*proto.NotificationResponse, error) {
	if err := h.validateRequest(req); err != nil {
		return h.createErrorResponse(codes.InvalidArgument, "Invalid request data", err.Error()), nil
	}
	if err := h.emailOtpService.SendOTP(ctx, req.UserId, req.Email, req.Username); err != nil {

		h.logger.Error("SendOTP failed", zap.String("userId", req.UserId), zap.String("email", req.Email), zap.Error(err))
		return h.createErrorResponse(codes.Internal, "Failed to send OTP", err.Error()), nil
	}
	metrics.OTPSentTotal.Inc()
	return h.createSuccessResponse("OTP sent successfully"), nil
}

// VerifyOTP verifies an OTP for the user.
func (h *Handler) VerifyOTP(ctx context.Context, req *proto.VerifyOTPRequest) (*proto.NotificationResponse, error) {
	if err := h.validateRequest(req); err != nil {
		return h.createErrorResponse(codes.InvalidArgument, "Invalid request data", err.Error()), nil
	}
	isValid, err := h.emailOtpService.VerifyOTP(ctx, req.Email, req.Otp)
	if err != nil {
		h.logger.Error("VerifyOTP failed", zap.String("email", req.Email), zap.Error(err))
		return h.createErrorResponse(codes.Internal, "Failed to verify OTP", err.Error()), nil
	}
	if !isValid {
		return h.createErrorResponse(codes.InvalidArgument, "Invalid OTP"), nil
	}
	h.logger.Info("OTP verified", zap.String("email", req.Email))
	return h.createSuccessResponse("OTP verified successfully"), nil
}

// ForgotPassword sends a password reset email.
func (h *Handler) ForgotPassword(ctx context.Context, req *proto.ForgotPasswordRequest) (*proto.NotificationResponse, error) {
	if err := h.validateRequest(req); err != nil {
		return h.createErrorResponse(codes.InvalidArgument, "Invalid request data", err.Error()), nil
	}
	if err := h.forgotPasswordService.Send(ctx, req.UserId, req.Username, req.Email, req.ResetLink); err != nil {
		h.logger.Error("ForgotPassword email failed", zap.String("userId", req.UserId), zap.String("email", req.Email), zap.Error(err))
		return h.createErrorResponse(codes.Internal, "Failed to send password reset email", err.Error()), nil
	}
	h.logger.Info("Password reset email sent", zap.String("email", req.Email))
	return h.createSuccessResponse("Password reset email sent successfully"), nil
}

// GetANotification fetches a single notification for the user.
func (h *Handler) GetNotification(ctx context.Context, req *proto.GetNotificationRequest) (*proto.GetNotificationResponse, error) {
	if err := h.validateRequest(req); err != nil {
		return &proto.GetNotificationResponse{
			Result: &proto.GetNotificationResponse_Error{
				Error: &proto.Error{
					Code:    codes.InvalidArgument.String(),
					Message: "Invalid request data",
					Details: []*proto.ErrorDetail{{Message: err.Error()}},
				},
			},
		}, nil
	}
	notification, err := h.notificationService.GetNotification(ctx, req.NotificationId, req.UserId)
	if err != nil {
		h.logger.Error("GetANotification failed", zap.String("userId", req.UserId), zap.String("notificationId", req.NotificationId), zap.Error(err))
		return &proto.GetNotificationResponse{
			Result: &proto.GetNotificationResponse_Error{
				Error: &proto.Error{
					Code:    codes.Internal.String(),
					Message: "Failed to get notification",
					Details: []*proto.ErrorDetail{{Message: err.Error()}},
				},
			},
		}, nil
	}
	return &proto.GetNotificationResponse{
		Result: &proto.GetNotificationResponse_Notification{
			Notification: &proto.NotificationData{
				Id:        notification.ID,
				UserId:    notification.UserId,
				Type:      string(notification.Type),
				Subject:   notification.Subject,
				Body:      notification.Body,
				Recipient: notification.Recipient,
				IsRead:    notification.IsRead,
				CreatedAt: notification.CreatedAt.Format(time.RFC3339),
				Priority:  notification.Priority,
				ActionUrl: notification.ActionURL,
				Category:  string(notification.Category),
				Metadata:  notification.Metadata,
			},
		},
	}, nil
}

// GetNotifications returns a paginated list of the user's notifications.
func (h *Handler) GetNotifications(ctx context.Context, req *proto.GetNotificationsRequest) (*proto.GetNotificationsResponse, error) {
	if err := h.validateRequest(req); err != nil {
		return &proto.GetNotificationsResponse{
			Result: &proto.GetNotificationsResponse_Error{
				Error: &proto.Error{
					Code:    codes.InvalidArgument.String(),
					Message: "Invalid request data",
					Details: []*proto.ErrorDetail{{Message: err.Error()}},
				},
			},
		}, nil
	}

	page := int(req.Params.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.Params.PageSize)
	if pageSize < 1 {
		pageSize = 10
	}

	var isRead *bool
	if req.Params.IsRead {
		val := req.Params.IsRead
		isRead = &val
	}
	var categoryType *string
	if req.Params.Category != "" {
		val := req.Params.Category
		categoryType = &val
	}

	var notificationCategory *entity.NotificationCategory
	if categoryType != nil {
		nt := entity.MapToCategory(*categoryType)
		notificationCategory = &nt
	}

	notifications, total, err := h.notificationService.ListNotifications(ctx, req.UserId, page, pageSize, isRead, notificationCategory)
	if err != nil {
		h.logger.Error("GetNotifications failed", zap.String("userId", req.UserId), zap.Error(err))
		return &proto.GetNotificationsResponse{
			Result: &proto.GetNotificationsResponse_Error{
				Error: &proto.Error{
					Code:    codes.Internal.String(),
					Message: "Failed to get notifications",
					Details: []*proto.ErrorDetail{{Message: err.Error()}},
				},
			},
		}, nil
	}

	protoNotifications := make([]*proto.NotificationData, 0, len(notifications))
	for _, n := range notifications {
		protoNotifications = append(protoNotifications, &proto.NotificationData{
			Id:        n.ID,
			UserId:    n.UserId,
			Type:      string(n.Type),
			Subject:   n.Subject,
			Body:      n.Body,
			Recipient: n.Recipient,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt.Format(time.RFC3339),
			Priority:  n.Priority,
			ActionUrl: n.ActionURL,
			Category:  string(n.Category),
			Metadata:  n.Metadata,
		})
	}

	return &proto.GetNotificationsResponse{
		Result: &proto.GetNotificationsResponse_Success{
			Success: &proto.GetAllNotificationsSuccess{
				Notifications: protoNotifications,
				Total:         int32(total),
			},
		},
	}, nil
}

// MarkAsRead marks a given notification as read for the user.
func (h *Handler) MarkAsRead(ctx context.Context, req *proto.MarkNotificationRequest) (*proto.NotificationResponse, error) {
	if err := h.validateRequest(req); err != nil {
		return h.createErrorResponse(codes.InvalidArgument, "Invalid request data", err.Error()), nil
	}
	if err := h.notificationService.MarkAsRead(ctx, req.NotificationId, req.UserId); err != nil {
		h.logger.Error("MarkAsRead failed", zap.String("notificationId", req.NotificationId), zap.Error(err))
		return h.createErrorResponse(codes.Internal, "Failed to mark notification as read", err.Error()), nil
	}
	return h.createSuccessResponse("Notification marked as read"), nil
}

// DeleteNotification deletes notification with given ID and userID.
func (h *Handler) DeleteNotification(ctx context.Context, req *proto.DeleteNotificationRequest) (*proto.DeleteNotificationResponse, error) {
	if err := h.validateRequest(req); err != nil {
		h.logger.Error("Validation failed", zap.String("notificationId", req.NotificationId), zap.String("userId", req.UserId), zap.Error(err))
		return &proto.DeleteNotificationResponse{
			Result: &proto.DeleteNotificationResponse_Error{
				Error: &proto.Error{
					Code:    codes.InvalidArgument.String(),
					Message: "Validation Failed",
					Details: []*proto.ErrorDetail{{Message: err.Error()}},
				},
			},
		}, nil
	}
	if err := h.notificationService.DeleteNotification(ctx, req.NotificationId, req.UserId); err != nil {
		h.logger.Error("Failed  to delete notification", zap.String("notificationId", req.NotificationId), zap.String("userId", req.UserId), zap.Error(err))
		return &proto.DeleteNotificationResponse{
			Result: &proto.DeleteNotificationResponse_Error{
				Error: &proto.Error{
					Code:    codes.Internal.String(),
					Message: "Failed to delete notification",
					Details: []*proto.ErrorDetail{{Message: err.Error()}},
				},
			},
		}, nil
	}
	h.logger.Debug("Successfully deleted notification", zap.String("notificationId", req.NotificationId), zap.String("userId", req.UserId))
	return &proto.DeleteNotificationResponse{
		Result: &proto.DeleteNotificationResponse_Success{
			Success: &proto.DeleteSuccess{
				Success: true,
			},
		},
	}, nil
}

func (h *Handler) ClearNotifications(ctx context.Context, req *proto.ClearUserNotificationsRequest) (*proto.ClearUserNotificationsResponse, error) {
	if err := h.validateRequest(req); err != nil {
		h.logger.Error("Validation failed", zap.String("userId", req.UserId), zap.Error(err))
		return &proto.ClearUserNotificationsResponse{
			Result: &proto.ClearUserNotificationsResponse_Error{
				Error: &proto.Error{
					Code:    codes.InvalidArgument.String(),
					Message: "Validation Failed",
					Details: []*proto.ErrorDetail{{Message: err.Error()}},
				},
			},
		}, nil
	}
	if err := h.notificationService.ClearNotifications(ctx, req.UserId); err != nil {
		h.logger.Error("Failed  to clear user notifications", zap.String("userId", req.UserId), zap.Error(err))
		return &proto.ClearUserNotificationsResponse{
			Result: &proto.ClearUserNotificationsResponse_Error{
				Error: &proto.Error{
					Code:    codes.Internal.String(),
					Message: "Failed to clear notifications",
					Details: []*proto.ErrorDetail{{Message: err.Error()}},
				},
			},
		}, nil
	}
	h.logger.Debug("Successfully cleared user notifications", zap.String("userId", req.UserId))
	return &proto.ClearUserNotificationsResponse{
		Result: &proto.ClearUserNotificationsResponse_Success{
			Success: &proto.DeleteSuccess{
				Success: true,
			},
		},
	}, nil
}

// MarkAllAsRead marks all notifications as read for the user.
func (h *Handler) MarkAllAsRead(ctx context.Context, req *proto.MarkAllNotificationsRequest) (*proto.NotificationResponse, error) {
	if err := h.validateRequest(req); err != nil {
		return h.createErrorResponse(codes.InvalidArgument, "Invalid request data", err.Error()), nil
	}
	if err := h.notificationService.MarkAllAsRead(ctx, req.UserId); err != nil {
		h.logger.Error("MarkAllAsRead failed", zap.String("userId", req.UserId), zap.Error(err))
		return h.createErrorResponse(codes.Internal, "Failed to mark all notifications as read", err.Error()), nil
	}
	return h.createSuccessResponse("All notifications marked as read"), nil
}
