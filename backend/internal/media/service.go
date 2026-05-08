package media

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// MediaService handles media storage and retrieval
type MediaService struct {
	storagePath string
	maxSize     int64
}

// MediaInfo contains metadata about uploaded media
type MediaInfo struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	Hash      string    `json:"hash"`
}

// NewMediaService creates a new media service
func NewMediaService(storagePath string, maxSize int64) *MediaService {
	// Ensure storage directory exists
	os.MkdirAll(storagePath, 0755)
	
	return &MediaService{
		storagePath: storagePath,
		maxSize:     maxSize,
	}
}

// Upload uploads a media file
func (m *MediaService) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, userID string) (*MediaInfo, error) {
	// Check file size
	if header.Size > m.maxSize {
		return nil, fmt.Errorf("file too large: max %d bytes", m.maxSize)
	}

	// Generate unique ID
	mediaID := uuid.New().String()

	// Calculate hash for deduplication
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	fileHash := fmt.Sprintf("%x", hash.Sum(nil))

	// Reset file pointer
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	// Create subdirectory based on date to avoid too many files in one dir
	now := time.Now()
	subDir := filepath.Join(
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)
	fullDir := filepath.Join(m.storagePath, subDir)
	
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Determine file extension
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".bin"
	}

	// Save file
	filename := fmt.Sprintf("%s%s", mediaID, ext)
	filepath := filepath.Join(fullDir, filename)

	outFile, err := os.Create(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		os.Remove(filepath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Create media info
	mediaInfo := &MediaInfo{
		ID:        mediaID,
		UserID:    userID,
		Filename:  header.Filename,
		MimeType:  header.Header.Get("Content-Type"),
		Size:      size,
		URL:       fmt.Sprintf("/api/v1/media/%s", mediaID),
		CreatedAt: now,
		Hash:      fileHash,
	}

	return mediaInfo, nil
}

// GetURL returns the URL for a media file
func (m *MediaService) GetURL(ctx context.Context, mediaID string) (string, error) {
	// TODO: Query database to find file path
	// For now, return placeholder
	return fmt.Sprintf("/media/%s", mediaID), nil
}

// Delete deletes a media file
func (m *MediaService) Delete(ctx context.Context, mediaID, userID string) error {
	// TODO: Query database to find file path
	// Verify ownership
	// Delete file
	
	return nil
}

// GetStats returns storage statistics
func (m *MediaService) GetStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	// TODO: Calculate user's storage usage
	return map[string]interface{}{
		"used_bytes":  0,
		"total_bytes": m.maxSize,
		"file_count":  0,
	}, nil
}
