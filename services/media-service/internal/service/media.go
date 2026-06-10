package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cfsign "github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

	mediav1 "github.com/wemall/gen/media/v1"
	"github.com/wemall/media-service/internal/config"
	"github.com/wemall/media-service/internal/db"
)

type MediaService struct {
	cfg           *config.Config
	log           zerolog.Logger
	q             *db.Queries
	pool          *pgxpool.Pool
	nc            *nats.Conn
	s3Client      *s3.Client
	presignClient *s3.PresignClient
	cfPrivateKey  *rsa.PrivateKey
	cfSigner      *cfsign.URLSigner
	mu            sync.RWMutex
	isMockMode    bool
}

func NewMediaService(cfg *config.Config, log zerolog.Logger, q *db.Queries, pool *pgxpool.Pool, nc *nats.Conn) *MediaService {
	svc := &MediaService{
		cfg:  cfg,
		log:  log,
		q:    q,
		pool: pool,
		nc:   nc,
	}

	// Determine if we are in local development mock mode
	if cfg.Environment == "development" && os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		svc.log.Warn().Msg("AWS credentials or buckets missing; starting in LOCAL DEVELOPMENT MOCK MODE")
		svc.isMockMode = true
		// Ensure local temp directories exist
		os.MkdirAll("./.tmp/media/raw", 0755)
		os.MkdirAll("./.tmp/media/public", 0755)
		os.MkdirAll("./.tmp/media/private", 0755)
	} else {
		// Initialize Real S3 Clients
		ctx := context.Background()
		awsCfg, err := s3config.LoadDefaultConfig(ctx, s3config.WithRegion(cfg.AWSRegion))
		if err != nil {
			svc.log.Error().Err(err).Msg("failed to load AWS configuration, falling back to mock mode")
			svc.isMockMode = true
		} else {
			svc.s3Client = s3.NewFromConfig(awsCfg)
			svc.presignClient = s3.NewPresignClient(svc.s3Client)
			svc.log.Info().Msg("S3 client and presigner initialized successfully")
		}

		// Initialize CloudFront URL Signer if private key exists
		if cfg.CloudFrontKeyPairID != "" {
			err := svc.initCloudFrontSigner()
			if err != nil {
				svc.log.Warn().Err(err).Msg("CloudFront signer could not be initialized, private assets signature generation will be mock/bypassed")
			}
		}
	}

	return svc
}

func (s *MediaService) initCloudFrontSigner() error {
	// Attempt to load private key from local or environment (e.g. for developer conveniences)
	keyPem := os.Getenv("AWS_CLOUDFRONT_PRIVATE_KEY_PEM")
	if keyPem == "" {
		return errors.New("AWS_CLOUDFRONT_PRIVATE_KEY_PEM environment variable is empty")
	}

	// Support single-line escaped PEM key formats (e.g. for Docker/Compose compatibility)
	keyPem = strings.ReplaceAll(keyPem, "\\n", "\n")

	block, _ := pem.Decode([]byte(keyPem))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return errors.New("failed to decode PEM block containing RSA private key")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse PKCS1 private key: %w", err)
	}

	s.cfPrivateKey = privKey
	s.cfSigner = cfsign.NewURLSigner(s.cfg.CloudFrontKeyPairID, privKey)
	s.log.Info().Msg("CloudFront Private Key loaded, URL Signer initialized")
	return nil
}

// ── Service API Implementations ──────────────────────────────────────────────

func (s *MediaService) RequestUploadUrl(ctx context.Context, ownerID, scope, filename, mimeType string, sizeBytes int64) (*mediav1.RequestUploadUrlResponse, error) {
	mediaID := uuid.New()
	rawKey := fmt.Sprintf("raw/%s/%s", mediaID.String(), filename)
	isPrivate := s.determinePrivacy(scope)

	// Register pending record in Database
	_, err := s.q.CreateMediaAsset(ctx, db.CreateMediaAssetParams{
		OwnerID:      uuid.MustParse(ownerID),
		ServiceScope: scope,
		OriginalName: filename,
		MimeType:     mimeType,
		SizeBytes:    sizeBytes,
		RawS3Key:     rawKey,
		IsPrivate:    isPrivate,
		Status:       "pending_upload",
	})
	if err != nil {
		s.log.Error().Err(err).Msg("failed to write media metadata in DB")
		return nil, fmt.Errorf("database transaction failed: %w", err)
	}

	var uploadURL string
	headers := make(map[string]string)

	if s.isMockMode {
		// Return local server upload endpoint
		uploadURL = fmt.Sprintf("http://localhost:%s/api/v1/media/mock-s3-upload?key=%s&id=%s", s.cfg.HTTPPort, rawKey, mediaID.String())
	} else {
		// Generate actual AWS S3 Presigned PUT URL
		presignedReq, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(s.cfg.AWSS3RawBucket),
			Key:         aws.String(rawKey),
			ContentType: aws.String(mimeType),
		}, s3.WithPresignExpires(15*time.Minute))
		if err != nil {
			s.log.Error().Err(err).Msg("failed to generate S3 presigned URL")
			return nil, fmt.Errorf("s3 presign failed: %w", err)
		}
		uploadURL = presignedReq.URL
		for k, v := range presignedReq.SignedHeader {
			headers[k] = strings.Join(v, ", ")
		}
	}

	s.log.Info().Str("media_id", mediaID.String()).Msg("Presigned upload URL generated successfully")
	return &mediav1.RequestUploadUrlResponse{
		MediaId:         mediaID.String(),
		UploadUrl:       uploadURL,
		RequiredHeaders: headers,
	}, nil
}

func (s *MediaService) ConfirmUpload(ctx context.Context, mediaIDStr string) (*mediav1.ConfirmUploadResponse, error) {
	mediaID, err := uuid.Parse(mediaIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid media id format: %w", err)
	}

	asset, err := s.q.GetMediaAsset(ctx, mediaID)
	if err != nil {
		return nil, fmt.Errorf("media asset not found: %w", err)
	}

	if asset.Status != "pending_upload" {
		return &mediav1.ConfirmUploadResponse{
			MediaId: asset.ID.String(),
			Status:  string(asset.Status),
		}, nil
	}

	// Verify S3 Object Exists
	if s.isMockMode {
		localPath := filepath.Join("./.tmp/media", asset.RawS3Key)
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("uploaded file not found on mock storage")
		}
	} else {
		_, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(s.cfg.AWSS3RawBucket),
			Key:    aws.String(asset.RawS3Key),
		})
		if err != nil {
			s.log.Error().Err(err).Str("key", asset.RawS3Key).Msg("failed to locate file in S3 bucket")
			return nil, fmt.Errorf("uploaded file could not be verified in S3 raw bucket: %w", err)
		}
	}

	// Update Status to Uploaded (which signals lambda processors to begin)
	updated, err := s.q.UpdateMediaStatus(ctx, db.UpdateMediaStatusParams{
		ID:     asset.ID,
		Status: "uploaded",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	// Publish JetStream event
	if s.nc != nil {
		eventPayload, _ := json.Marshal(map[string]string{
			"media_id":      asset.ID.String(),
			"raw_s3_key":    asset.RawS3Key,
			"mime_type":     asset.MimeType,
			"service_scope": asset.ServiceScope,
		})
		_ = s.nc.Publish("wemall.media.uploaded", eventPayload)
	}

	if s.isMockMode {
		// Spin up background goroutine to mock Lambda optimization
		go s.mockBackgroundLambdaProcess(asset)
	}

	return &mediav1.ConfirmUploadResponse{
		MediaId: updated.ID.String(),
		Status:  string(updated.Status),
	}, nil
}

func (s *MediaService) UploadDirect(ctx context.Context, r io.Reader, filename, mimeType, scope string, isPrivate bool, sizeBytes int64) (*mediav1.MediaAsset, error) {
	mediaID := uuid.New()
	rawKey := fmt.Sprintf("raw/%s/%s", mediaID.String(), filename)

	// Save to DB as pending_upload
	asset, err := s.q.CreateMediaAsset(ctx, db.CreateMediaAssetParams{
		OwnerID:      uuid.Nil, // Direct upload default, or parse from context auth if accessible
		ServiceScope: scope,
		OriginalName: filename,
		MimeType:     mimeType,
		SizeBytes:    sizeBytes,
		RawS3Key:     rawKey,
		IsPrivate:    isPrivate,
		Status:       "pending_upload",
	})
	if err != nil {
		return nil, fmt.Errorf("database insert failed: %w", err)
	}

	// Stream directly to landing zone
	if s.isMockMode {
		localPath := filepath.Join("./.tmp/media", rawKey)
		os.MkdirAll(filepath.Dir(localPath), 0755)
		out, err := os.Create(localPath)
		if err != nil {
			return nil, fmt.Errorf("failed to write mock local raw file: %w", err)
		}
		defer out.Close()
		_, err = io.Copy(out, r)
		if err != nil {
			return nil, fmt.Errorf("failed to write local raw file streams: %w", err)
		}
	} else {
		_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(s.cfg.AWSS3RawBucket),
			Key:         aws.String(rawKey),
			Body:        r,
			ContentType: aws.String(mimeType),
		})
		if err != nil {
			s.log.Error().Err(err).Msg("failed to upload bytes directly to S3")
			return nil, fmt.Errorf("failed to stream upload to S3 raw: %w", err)
		}
	}

	// Auto confirm
	_, err = s.ConfirmUpload(ctx, asset.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to auto-confirm uploaded file: %w", err)
	}

	// Refetch updated asset
	updatedAsset, err := s.GetMediaAsset(ctx, asset.ID.String(), "")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve finalized asset: %w", err)
	}

	return updatedAsset, nil
}

func (s *MediaService) GetMediaAsset(ctx context.Context, mediaIDStr, requestedByID string) (*mediav1.MediaAsset, error) {
	mediaID, err := uuid.Parse(mediaIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid media id format: %w", err)
	}

	row, err := s.q.GetMediaAsset(ctx, mediaID)
	if err != nil {
		return nil, fmt.Errorf("media asset not found: %w", err)
	}

	return s.mapRowToProto(row), nil
}

func (s *MediaService) BatchGetMediaAssets(ctx context.Context, mediaIDs []string, requestedByID string) ([]*mediav1.MediaAsset, error) {
	uuids := make([]uuid.UUID, 0, len(mediaIDs))
	for _, id := range mediaIDs {
		if u, err := uuid.Parse(id); err == nil {
			uuids = append(uuids, u)
		}
	}

	rows, err := s.q.BatchGetMediaAssets(ctx, uuids)
	if err != nil {
		return nil, fmt.Errorf("batch database fetch failed: %w", err)
	}

	// Maintain order of original array
	rowMap := make(map[string]db.MediaAsset)
	for _, row := range rows {
		rowMap[row.ID.String()] = row
	}

	assets := make([]*mediav1.MediaAsset, 0, len(mediaIDs))
	for _, id := range mediaIDs {
		if row, found := rowMap[id]; found {
			assets = append(assets, s.mapRowToProto(row))
		}
	}

	return assets, nil
}

func (s *MediaService) ListMediaAssets(ctx context.Context, ownerID string, scope, mimeTypeFilter, statusFilter string, limit, offset int32) ([]*mediav1.MediaAsset, int32, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid owner id: %w", err)
	}

	var scopeArg *string
	if scope != "" {
		scopeArg = &scope
	}

	var mimeArg *string
	if mimeTypeFilter != "" {
		mime := mimeTypeFilter + "%"
		mimeArg = &mime
	}

	var statusArg *db.MediaStatus
	if statusFilter != "" {
		st := db.MediaStatus(statusFilter)
		statusArg = &st
	}

	// Fetch count
	totalCount, err := s.q.CountMediaAssets(ctx, db.CountMediaAssetsParams{
		OwnerID:      ownerUUID,
		ServiceScope: scopeArg,
		MimeType:     mimeArg,
		Status:       statusArg,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count media assets: %w", err)
	}

	// Fetch page
	rows, err := s.q.ListMediaAssets(ctx, db.ListMediaAssetsParams{
		OwnerID:      ownerUUID,
		ServiceScope: scopeArg,
		MimeType:     mimeArg,
		Status:       statusArg,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query media assets page: %w", err)
	}

	assets := make([]*mediav1.MediaAsset, len(rows))
	for i, row := range rows {
		assets[i] = s.mapRowToProto(row)
	}

	return assets, int32(totalCount), nil
}

// ── Private Helper Functions ─────────────────────────────────────────────────

func (s *MediaService) determinePrivacy(scope string) bool {
	switch scope {
	case "seller-kyc", "user-invoice", "seller-tax":
		return true
	default:
		return false
	}
}

func (s *MediaService) mapRowToProto(row db.MediaAsset) *mediav1.MediaAsset {
	asset := &mediav1.MediaAsset{
		Id:           row.ID.String(),
		OwnerId:      row.OwnerID.String(),
		OriginalName: row.OriginalName,
		MimeType:     row.MimeType,
		Status:       string(row.Status),
		IsPrivate:    row.IsPrivate,
	}

	// Parse timestamp using timestamppb
	asset.CreatedAt = timestamppb.New(row.CreatedAt)

	// ALWAYS provide the original URL as a fallback / immediate response
	originalURL := s.resolveVariantURL(row.ID.String(), row.OriginalName, row.IsPrivate)
	asset.Variants = &mediav1.MediaVariants{
		TypeVariants: &mediav1.MediaVariants_Document{
			Document: &mediav1.DocumentVariants{
				OriginalUrl: originalURL,
			},
		},
	}

	// If variations have been populated, construct structured response
	if row.Status == "completed" && len(row.Variants) > 0 {
		var urlMap map[string]string
		_ = json.Unmarshal(row.Variants, &urlMap)

		if strings.HasPrefix(row.MimeType, "image/") {
			asset.Variants = &mediav1.MediaVariants{
				TypeVariants: &mediav1.MediaVariants_Image{
					Image: &mediav1.ImageVariants{
						ThumbnailSmallAvif:  s.resolveVariantURL(row.ID.String(), "thumbnail_small.avif", row.IsPrivate),
						ThumbnailSmallWebp:  s.resolveVariantURL(row.ID.String(), "thumbnail_small.webp", row.IsPrivate),
						ThumbnailLargeAvif:  s.resolveVariantURL(row.ID.String(), "thumbnail_large.avif", row.IsPrivate),
						ThumbnailLargeWebp:  s.resolveVariantURL(row.ID.String(), "thumbnail_large.webp", row.IsPrivate),
						MainMobileAvif:      s.resolveVariantURL(row.ID.String(), "main_mobile.avif", row.IsPrivate),
						MainMobileWebp:      s.resolveVariantURL(row.ID.String(), "main_mobile.webp", row.IsPrivate),
						MainTabletAvif:      s.resolveVariantURL(row.ID.String(), "main_tablet.avif", row.IsPrivate),
						MainTabletWebp:      s.resolveVariantURL(row.ID.String(), "main_tablet.webp", row.IsPrivate),
						MainDesktopAvif:     s.resolveVariantURL(row.ID.String(), "main_desktop.avif", row.IsPrivate),
						MainDesktopWebp:     s.resolveVariantURL(row.ID.String(), "main_desktop.webp", row.IsPrivate),
						MainLargeRetinaAvif: s.resolveVariantURL(row.ID.String(), "main_large_retina.avif", row.IsPrivate),
						MainLargeRetinaWebp: s.resolveVariantURL(row.ID.String(), "main_large_retina.webp", row.IsPrivate),
					},
				},
			}
		} else if strings.HasPrefix(row.MimeType, "video/") {
			asset.Variants = &mediav1.MediaVariants{
				TypeVariants: &mediav1.MediaVariants_Video{
					Video: &mediav1.VideoVariants{
						HlsPlaylistUrl: s.resolveVariantURL(row.ID.String(), "playlist.m3u8", row.IsPrivate),
						DashPlaylistUrl: s.resolveVariantURL(row.ID.String(), "playlist.mpd", row.IsPrivate),
					},
				},
			}
		} else {
			// Document fallback
			asset.Variants = &mediav1.MediaVariants{
				TypeVariants: &mediav1.MediaVariants_Document{
					Document: &mediav1.DocumentVariants{
						OriginalUrl: s.resolveVariantURL(row.ID.String(), row.OriginalName, row.IsPrivate),
					},
				},
			}
		}
	}

	return asset
}


func (s *MediaService) resolveVariantURL(mediaID, filename string, isPrivate bool) string {
	var baseURL string
	if isPrivate {
		baseURL = s.cfg.AWSCloudFrontPrivate
	} else {
		baseURL = s.cfg.AWSCloudFrontPublic
	}

	if s.isMockMode {
		return fmt.Sprintf("%s/uploads/%s/%s", baseURL, mediaID, filename)
	}

	rawURL := fmt.Sprintf("%s/uploads/%s/%s", baseURL, mediaID, filename)

	if isPrivate {
		// Dynamic URL Signing
		if s.cfSigner != nil {
			// Return a permanently signed URL (100 years in the future)
			signedURL, err := s.cfSigner.Sign(rawURL, time.Now().AddDate(100, 0, 0))
			if err == nil {
				return signedURL
			}
			s.log.Error().Err(err).Msg("failed to sign CloudFront URL, returning raw unsignable path")
		}
		// Return standard path with warning suffix for debugging if key not loaded
		return rawURL + "?signature=mock-active-expired-or-missing"
	}

	return rawURL
}

// mockBackgroundLambdaProcess simulates the event-driven AWS Lambda/MediaConvert optimizer locally
func (s *MediaService) mockBackgroundLambdaProcess(asset db.MediaAsset) {
	time.Sleep(1 * time.Second) // simulate latency

	s.log.Info().Str("media_id", asset.ID.String()).Msg("Mock Lambda: processing image resize variations...")

	rawPath := filepath.Join("./.tmp/media", asset.RawS3Key)
	destDir := filepath.Join("./.tmp/media/public/images", asset.ID.String())
	if asset.IsPrivate {
		destDir = filepath.Join("./.tmp/media/private/images", asset.ID.String())
	}
	os.MkdirAll(destDir, 0755)

	// Open raw file
	rawFile, err := os.Open(rawPath)
	if err == nil {
		defer rawFile.Close()
		// Mock generating variations simply by copying files with respective names
		variants := []string{
			"thumbnail_small.avif", "thumbnail_small.webp",
			"thumbnail_large.avif", "thumbnail_large.webp",
			"main_mobile.avif", "main_mobile.webp",
			"main_tablet.avif", "main_tablet.webp",
			"main_desktop.avif", "main_desktop.webp",
			"main_large_retina.avif", "main_large_retina.webp",
		}
		for _, vName := range variants {
			destFile, err := os.Create(filepath.Join(destDir, vName))
			if err == nil {
				rawFile.Seek(0, io.SeekStart)
				io.Copy(destFile, rawFile)
				destFile.Close()
			}
		}
	}

	// Update database with completion status and mocked variants
	varMap := map[string]string{
		"thumbnail_small_avif": fmt.Sprintf("images/%s/thumbnail_small.avif", asset.ID.String()),
		"thumbnail_small_webp": fmt.Sprintf("images/%s/thumbnail_small.webp", asset.ID.String()),
	}
	varBytes, _ := json.Marshal(varMap)

	ctx := context.Background()
	_, err = s.q.UpdateMediaVariants(ctx, db.UpdateMediaVariantsParams{
		ID:       asset.ID,
		Variants: varBytes,
	})
	if err != nil {
		s.log.Error().Err(err).Str("media_id", asset.ID.String()).Msg("failed to update mock variants in db")
		return
	}

	s.log.Info().Str("media_id", asset.ID.String()).Msg("Mock Lambda: finished variant creations. Status: COMPLETED")

	// Broadcast success event
	if s.nc != nil {
		eventPayload, _ := json.Marshal(map[string]string{
			"media_id": asset.ID.String(),
			"status":   "completed",
		})
		_ = s.nc.Publish("wemall.media.processed", eventPayload)
	}
}
