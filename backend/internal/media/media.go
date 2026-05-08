// Package media handles media upload, processing, and delivery
package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	// MaxMediaSize is the maximum allowed media file size (2GB as per spec)
	MaxMediaSize = 2 * 1024 * 1024 * 1024 // 2GB
	
	// MediaTypeImage for image files
	MediaTypeImage = "image"
	// MediaTypeVideo for video files
	MediaTypeVideo = "video"
	// MediaTypeAudio for audio/voice files
	MediaTypeAudio = "audio"
	// MediaTypeFile for document files
	MediaTypeFile = "file"
	
	// ThumbnailWidth for generated thumbnails
	ThumbnailWidth = 320
	// ThumbnailHeight for generated thumbnails
	ThumbnailHeight = 320
	
	// Supported image extensions
	ImageExtension = ".jpg,.jpeg,.png,.gif,.webp,.heic"
	// Supported video extensions
	VideoExtension = ".mp4,.mov,.avi,.mkv,.webm"
	// Supported audio extensions
	AudioExtension = ".mp3,.wav,.ogg,.m4a,.opus"
)

// MediaMetadata holds metadata about uploaded media
type MediaMetadata struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // image, video, audio, file
	FileName    string            `json:"file_name"`
	FileSize    int64             `json:"file_size"`
	MimeType    string            `json:"mime_type"`
	URL         string            `json:"url"`
	ThumbnailURL string           `json:"thumbnail_url,omitempty"`
	Width       int               `json:"width,omitempty"`
	Height      int               `json:"height,omitempty"`
	Duration    int               `json:"duration,omitempty"` // seconds
	Hash        string            `json:"hash"` // SHA-256 hash
	UploadedBy  string            `json:"uploaded_by"`
	UploadedAt  time.Time         `json:"uploaded_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"` // For temporary media
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// UploadRequest represents a media upload request
type UploadRequest struct {
	File      multipart.File `json:"-"`
	Header    *multipart.FileHeader `json:"-"`
	UserID    string         `json:"user_id"`
	ChatID    string         `json:"chat_id,omitempty"`
	MediaType string         `json:"media_type"`
}

// TranscriptionResult holds voice message transcription
type TranscriptionResult struct {
	Text      string    `json:"text"`
	Language  string    `json:"language,omitempty"`
	Confidence float32  `json:"confidence"`
	Duration  int       `json:"duration"`
}

// Service handles media operations
type Service struct {
	config      Config
	logger      *zap.Logger
	mu          sync.RWMutex
	uploads     map[string]*MediaMetadata
	tempDir     string
	storagePath string
}

// Config holds media service configuration
type Config struct {
	StoragePath       string        `json:"storage_path"`
	TempPath          string        `json:"temp_path"`
	CDNBaseURL        string        `json:"cdn_base_url"`
	MaxFileSize       int64         `json:"max_file_size"`
	EnableTranscription bool        `json:"enable_transcription"`
	AutoCompressImages  bool        `json:"auto_compress_images"`
	ImageQuality        int         `json:"image_quality"` // 1-100
	GenerateThumbnails  bool        `json:"generate_thumbnails"`
	RetentionDays       int         `json:"retention_days"` // 0 = forever
}

// NewService creates a new media service
func NewService(config Config, logger *zap.Logger) (*Service, error) {
	// Create storage directories
	if err := os.MkdirAll(config.StoragePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	if err := os.MkdirAll(config.TempPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &Service{
		config:      config,
		logger:      logger,
		uploads:     make(map[string]*MediaMetadata),
		tempDir:     config.TempPath,
		storagePath: config.StoragePath,
	}, nil
}

// Upload uploads a media file
func (s *Service) Upload(ctx context.Context, req UploadRequest) (*MediaMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate file size
	if req.Header.Size > s.config.MaxFileSize {
		return nil, errors.New("file too large")
	}
	if req.Header.Size > MaxMediaSize {
		return nil, errors.New("file exceeds maximum size of 2GB")
	}

	// Determine media type if not specified
	mediaType := req.MediaType
	if mediaType == "" {
		mediaType = s.detectMediaType(req.Header.Filename)
	}

	// Generate unique ID
	id := generateMediaID()
	ext := strings.ToLower(filepath.Ext(req.Header.Filename))
	if ext == "" {
		ext = s.getExtensionFromMime(req.Header.Header.Get("Content-Type"))
	}

	// Create file path
	filePath := filepath.Join(s.storagePath, id[:2], id[2:4])
	if err := os.MkdirAll(filePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	filename := id + ext
	fullPath := filepath.Join(filePath, filename)

	// Save file
	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)
	
	if _, err := io.Copy(writer, req.File); err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Generate URL
	url := fmt.Sprintf("%s/%s/%s/%s", s.config.CDNBaseURL, id[:2], id[2:4], filename)

	// Create metadata
	metadata := &MediaMetadata{
		ID:         id,
		Type:       mediaType,
		FileName:   req.Header.Filename,
		FileSize:   fileInfo.Size(),
		MimeType:   req.Header.Header.Get("Content-Type"),
		URL:        url,
		Hash:       hex.EncodeToString(hasher.Sum(nil)),
		UploadedBy: req.UserID,
		UploadedAt: time.Now(),
	}

	// Generate thumbnail for images and videos
	if s.config.GenerateThumbnails && (mediaType == MediaTypeImage || mediaType == MediaTypeVideo) {
		thumbnailURL, err := s.generateThumbnail(fullPath, id, mediaType)
		if err != nil {
			s.logger.Warn("Failed to generate thumbnail", zap.Error(err))
		} else {
			metadata.ThumbnailURL = thumbnailURL
		}
	}

	// Extract dimensions for images
	if mediaType == MediaTypeImage {
		width, height, err := s.getImageDimensions(fullPath)
		if err != nil {
			s.logger.Warn("Failed to get image dimensions", zap.Error(err))
		} else {
			metadata.Width = width
			metadata.Height = height
		}
	}

	// Extract duration for audio/video
	if mediaType == MediaTypeAudio || mediaType == MediaTypeVideo {
		duration, err := s.getMediaDuration(fullPath, mediaType)
		if err != nil {
			s.logger.Warn("Failed to get media duration", zap.Error(err))
		} else {
			metadata.Duration = duration
		}
	}

	// Store metadata
	s.uploads[id] = metadata

	s.logger.Info("Media uploaded", 
		zap.String("id", id), 
		zap.String("type", mediaType),
		zap.Int64("size", fileInfo.Size()))

	return metadata, nil
}

// GetMedia retrieves media metadata by ID
func (s *Service) GetMedia(id string) (*MediaMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metadata, exists := s.uploads[id]
	if !exists {
		return nil, errors.New("media not found")
	}
	return metadata, nil
}

// DeleteMedia deletes a media file
func (s *Service) DeleteMedia(id, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata, exists := s.uploads[id]
	if !exists {
		return errors.New("media not found")
	}

	// Check ownership
	if metadata.UploadedBy != userID {
		return errors.New("can only delete your own media")
	}

	// Delete file from disk
	filePath := s.getFilePath(id)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("Failed to delete media file", zap.Error(err))
	}

	// Delete thumbnail if exists
	if metadata.ThumbnailURL != "" {
		thumbnailPath := s.getThumbnailPath(id)
		os.Remove(thumbnailPath)
	}

	delete(s.uploads, id)
	s.logger.Info("Media deleted", zap.String("id", id))
	return nil
}

// TranscribeAudio transcribes an audio file (voice message)
func (s *Service) TranscribeAudio(ctx context.Context, mediaID string) (*TranscriptionResult, error) {
	if !s.config.EnableTranscription {
		return nil, errors.New("transcription is disabled")
	}

	metadata, err := s.GetMedia(mediaID)
	if err != nil {
		return nil, err
	}

	if metadata.Type != MediaTypeAudio {
		return nil, errors.New("can only transcribe audio files")
	}

	// In production, integrate with speech-to-text service
	// For MVP, return placeholder
	s.logger.Info("Transcribing audio", zap.String("id", mediaID))
	
	// Placeholder - implement actual transcription with Whisper/on-device STT
	return &TranscriptionResult{
		Text:       "[Transcription not available in MVP]",
		Language:   "en",
		Confidence: 0.95,
		Duration:   metadata.Duration,
	}, nil
}

// Helper methods

func (s *Service) detectMediaType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	
	if strings.Contains(ImageExtension, ext) {
		return MediaTypeImage
	}
	if strings.Contains(VideoExtension, ext) {
		return MediaTypeVideo
	}
	if strings.Contains(AudioExtension, ext) {
		return MediaTypeAudio
	}
	return MediaTypeFile
}

func (s *Service) getExtensionFromMime(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	default:
		return ""
	}
}

func (s *Service) getFilePath(id string) string {
	return filepath.Join(s.storagePath, id[:2], id[2:4], id)
}

func (s *Service) getThumbnailPath(id string) string {
	return filepath.Join(s.storagePath, id[:2], id[2:4], id+"_thumb.jpg")
}

func (s *Service) generateThumbnail(sourcePath, id, mediaType string) (string, error) {
	// In production, use image processing library (e.g., imglib, vips)
	// For MVP, return placeholder URL
	thumbnailPath := s.getThumbnailPath(id)
	thumbnailURL := fmt.Sprintf("%s/%s/%s/%s_thumb.jpg", s.config.CDNBaseURL, id[:2], id[2:4], id)
	
	// Placeholder - implement actual thumbnail generation
	s.logger.Debug("Thumbnail generation placeholder", zap.String("path", thumbnailPath))
	
	return thumbnailURL, nil
}

func (s *Service) getImageDimensions(path string) (int, int, error) {
	// In production, use image.Decode to get dimensions
	// For MVP, return placeholder
	return 0, 0, nil
}

func (s *Service) getMediaDuration(path, mediaType string) (int, error) {
	// In production, use ffprobe or similar to get duration
	// For MVP, return placeholder
	return 0, nil
}

func generateMediaID() string {
	return time.Now().Format("20060102150405") + "_" + randomString(12)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond) // Ensure different values
	}
	return string(b)
}
