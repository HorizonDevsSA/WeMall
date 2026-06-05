package handler

import (
	"context"

	mediav1 "github.com/wemall/gen/media/v1"
	"github.com/wemall/media-service/internal/service"
)

type MediaGrpcHandler struct {
	svc *service.MediaService
}

func NewMediaGrpcHandler(svc *service.MediaService) *MediaGrpcHandler {
	return &MediaGrpcHandler{svc: svc}
}

func (h *MediaGrpcHandler) RequestUploadUrl(ctx context.Context, req *mediav1.RequestUploadUrlRequest) (*mediav1.RequestUploadUrlResponse, error) {
	return h.svc.RequestUploadUrl(ctx, req.OwnerId, req.ServiceScope, req.Filename, req.MimeType, req.SizeBytes)
}

func (h *MediaGrpcHandler) ConfirmUpload(ctx context.Context, req *mediav1.ConfirmUploadRequest) (*mediav1.ConfirmUploadResponse, error) {
	return h.svc.ConfirmUpload(ctx, req.MediaId)
}

func (h *MediaGrpcHandler) GetMediaAsset(ctx context.Context, req *mediav1.GetMediaAssetRequest) (*mediav1.GetMediaAssetResponse, error) {
	asset, err := h.svc.GetMediaAsset(ctx, req.MediaId, req.RequestedByUserId)
	if err != nil {
		return nil, err
	}
	return &mediav1.GetMediaAssetResponse{Asset: asset}, nil
}

func (h *MediaGrpcHandler) BatchGetMediaAssets(ctx context.Context, req *mediav1.BatchGetMediaAssetsRequest) (*mediav1.BatchGetMediaAssetsResponse, error) {
	assets, err := h.svc.BatchGetMediaAssets(ctx, req.MediaIds, req.RequestedByUserId)
	if err != nil {
		return nil, err
	}
	return &mediav1.BatchGetMediaAssetsResponse{Assets: assets}, nil
}

func (h *MediaGrpcHandler) ListMediaAssets(ctx context.Context, req *mediav1.ListMediaAssetsRequest) (*mediav1.ListMediaAssetsResponse, error) {
	assets, total, err := h.svc.ListMediaAssets(ctx, req.OwnerId, req.ServiceScope, req.MimeType, req.Status, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	return &mediav1.ListMediaAssetsResponse{
		Assets:     assets,
		TotalCount: total,
		Limit:      req.Limit,
		Offset:     req.Offset,
	}, nil
}
