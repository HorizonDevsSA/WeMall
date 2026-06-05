package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/wemall/pkg/auth"
	"github.com/wemall/media-service/internal/service"
)

type MediaRestHandler struct {
	svc     *service.MediaService
	authMgr *auth.Manager
}

func NewMediaRestHandler(svc *service.MediaService, authMgr *auth.Manager) *MediaRestHandler {
	return &MediaRestHandler{
		svc:     svc,
		authMgr: authMgr,
	}
}

func (h *MediaRestHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/media/upload", h.HandleUpload)
	mux.HandleFunc("/api/v1/media", h.HandleListOrDetail)
	mux.HandleFunc("/api/v1/media/mock-s3-upload", h.HandleMockS3Upload)
	mux.HandleFunc("/uploads/", h.HandleServeMockUploadedFiles)
}

func (h *MediaRestHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Authenticate Request
	claims, err := h.authenticate(r)
	var ownerID string
	if err != nil {
		// In development fallback, allow bypass
		ownerID = uuid.Nil.String()
	} else {
		ownerID = claims.UserID
	}

	// Max 15MB file sizes parsing
	err = r.ParseMultipartForm(15 << 20)
	if err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing 'file' input parameter", http.StatusBadRequest)
		return
	}
	defer file.Close()

	scope := r.FormValue("scope")
	if scope == "" {
		scope = "product-image" // default
	}

	isPrivateStr := r.FormValue("is_private")
	isPrivate := false
	if strings.ToLower(isPrivateStr) == "true" {
		isPrivate = true
	}

	// Call Direct Upload Service
	asset, err := h.svc.UploadDirect(r.Context(), file, header.Filename, header.Header.Get("Content-Type"), scope, isPrivate, header.Size)
	if err != nil {
		http.Error(w, fmt.Sprintf("upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Associate original owner ID from auth if possible
	if ownerID != uuid.Nil.String() && ownerID != "" {
		asset.OwnerId = ownerID
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(asset)
}

func (h *MediaRestHandler) HandleListOrDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if specific ID is requested (e.g. via ?id=)
	idStr := r.URL.Query().Get("id")
	if idStr != "" {
		h.handleDetail(w, r, idStr)
		return
	}

	// Else perform Listing
	claims, err := h.authenticate(r)
	var ownerID string
	if err != nil {
		// Fallback for easy API testing
		ownerID = uuid.Nil.String()
	} else {
		ownerID = claims.UserID
	}

	scope := r.URL.Query().Get("scope")
	mimeFilter := r.URL.Query().Get("mime_type")
	statusFilter := r.URL.Query().Get("status")

	limitStr := r.URL.Query().Get("limit")
	limit := int32(20)
	if limitVal, err := strconv.Atoi(limitStr); err == nil && limitVal > 0 {
		limit = int32(limitVal)
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := int32(0)
	if offsetVal, err := strconv.Atoi(offsetStr); err == nil && offsetVal >= 0 {
		offset = int32(offsetVal)
	}

	assets, total, err := h.svc.ListMediaAssets(r.Context(), ownerID, scope, mimeFilter, statusFilter, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list assets: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"assets":      assets,
		"total_count": total,
		"limit":       limit,
		"offset":      offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MediaRestHandler) handleDetail(w http.ResponseWriter, r *http.Request, mediaID string) {
	asset, err := h.svc.GetMediaAsset(r.Context(), mediaID, "")
	if err != nil {
		http.Error(w, fmt.Sprintf("media details not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

func (h *MediaRestHandler) HandleMockS3Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing s3 raw path key", http.StatusBadRequest)
		return
	}

	localPath := filepath.Join("./.tmp/media", key)
	os.MkdirAll(filepath.Dir(localPath), 0755)

	out, err := os.Create(localPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to write local raw file: %v", err), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, r.Body)
	if err != nil {
		http.Error(w, "failed to save raw streams", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("mock upload completed successfully"))
}

func (h *MediaRestHandler) HandleServeMockUploadedFiles(w http.ResponseWriter, r *http.Request) {
	// Serve static resized variant files from local storage mock folders
	relPath := strings.TrimPrefix(r.URL.Path, "/uploads/")
	
	// Check public folder first
	pubPath := filepath.Join("./.tmp/media/public", relPath)
	if _, err := os.Stat(pubPath); err == nil {
		http.ServeFile(w, r, pubPath)
		return
	}

	// Check private folder
	privPath := filepath.Join("./.tmp/media/private", relPath)
	if _, err := os.Stat(privPath); err == nil {
		// Mock policy checking: check if ?signature= is present
		sig := r.URL.Query().Get("signature")
		if sig == "" {
			http.Error(w, "access denied: missing signature", http.StatusForbidden)
			return
		}
		http.ServeFile(w, r, privPath)
		return
	}

	http.NotFound(w, r)
}

func (h *MediaRestHandler) authenticate(r *http.Request) (*auth.Claims, error) {
	if h.authMgr == nil {
		return nil, fmt.Errorf("auth manager is not initialized")
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("authorization bearer token missing")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := h.authMgr.ValidateAccessToken(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}
