package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"messenger/internal/auth"
	"messenger/internal/chat"
	"messenger/internal/media"
	"messenger/internal/websocket"
)

// Handler represents the API handler with all services
type Handler struct {
	authService  *auth.AuthService
	chatService  *chat.ChatService
	mediaService *media.MediaService
	hub          *websocket.Hub
	logger       *zap.Logger
}

// NewHandler creates a new API handler
func NewHandler(authService *auth.AuthService, chatService *chat.ChatService, 
	mediaService *media.MediaService, hub *websocket.Hub, logger *zap.Logger) *Handler {
	return &Handler{
		authService:  authService,
		chatService:  chatService,
		mediaService: mediaService,
		hub:          hub,
		logger:       logger,
	}
}

// HealthCheck returns health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "0.1.0-mvp",
	})
}

// RequestOTP sends OTP to phone/email
func (h *Handler) RequestOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone,omitempty"`
		Email string `json:"email,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	identifier := req.Phone
	if identifier == "" {
		identifier = req.Email
	}

	if identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phone or email required"})
		return
	}

	otp, err := h.authService.RequestOTP(c.Request.Context(), identifier)
	if err != nil {
		h.logger.Error("Failed to request OTP", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send OTP"})
		return
	}

	// In production, send OTP via SMS/email
	// For now, return it for testing (REMOVE IN PRODUCTION)
	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent",
		"otp":     otp, // REMOVE IN PRODUCTION
		"expires_in": 300,
	})
}

// VerifyOTP verifies OTP and returns session token
func (h *Handler) VerifyOTP(c *gin.Context) {
	var req struct {
		Identifier string `json:"identifier"`
		OTP        string `json:"otp"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	session, err := h.authService.VerifyOTP(c.Request.Context(), req.Identifier, req.OTP)
	if err != nil {
		h.logger.Error("Failed to verify OTP", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         session.Token,
		"refresh_token": session.RefreshToken,
		"expires_at":    session.ExpiresAt,
		"user_id":       session.UserID,
	})
}

// Enable2FA enables two-factor authentication
func (h *Handler) Enable2FA(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	secret, qrCode, err := h.authService.EnableTOTP(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to enable 2FA", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret": secret,
		"qr_code": qrCode,
		"message": "Scan QR code with authenticator app",
	})
}

// Verify2FA verifies TOTP code
func (h *Handler) Verify2FA(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Code string `json:"code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	valid, err := h.authService.VerifyTOTP(c.Request.Context(), userID, req.Code)
	if err != nil || !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid 2FA code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA verified"})
}

// Disable2FA disables two-factor authentication
func (h *Handler) Disable2FA(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.authService.DisableTOTP(c.Request.Context(), userID); err != nil {
		h.logger.Error("Failed to disable 2FA", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA disabled"})
}

// RegisterPasskey registers a new passkey
func (h *Handler) RegisterPasskey(c *gin.Context) {
	// TODO: Implement WebAuthn passkey registration
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

// LoginWithPasskey logs in using passkey
func (h *Handler) LoginWithPasskey(c *gin.Context) {
	// TODO: Implement WebAuthn passkey authentication
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

// Logout invalidates session
func (h *Handler) Logout(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.authService.InvalidateSession(c.Request.Context(), userID); err != nil {
		h.logger.Error("Failed to logout", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetProfile returns user profile
func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// TODO: Get profile from database
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"name":    "User",
		"phone":   "+1234567890",
		"avatar":  "",
	})
}

// UpdateProfile updates user profile
func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// TODO: Update profile in database
	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}

// GetContacts returns user contacts
func (h *Handler) GetContacts(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// TODO: Get contacts from database
	c.JSON(http.StatusOK, gin.H{
		"contacts": []interface{}{},
	})
}

// SyncContacts syncs contacts with server
func (h *Handler) SyncContacts(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Contacts []struct {
			Phone string `json:"phone"`
			Name  string `json:"name"`
		} `json:"contacts"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// TODO: Sync contacts
	c.JSON(http.StatusOK, gin.H{"message": "Contacts synced"})
}

// ListChats returns list of chats
func (h *Handler) ListChats(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	chats, err := h.chatService.ListChats(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list chats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list chats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chats": chats,
		"total": len(chats),
	})
}

// CreateChat creates a new chat
func (h *Handler) CreateChat(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Type      string   `json:"type"` // "direct", "group", "channel"
		Name      string   `json:"name"`
		MemberIDs []string `json:"member_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	chat, err := h.chatService.CreateChat(c.Request.Context(), userID, req.Type, req.Name, req.MemberIDs)
	if err != nil {
		h.logger.Error("Failed to create chat", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"chat": chat})
}

// GetChat returns chat details
func (h *Handler) GetChat(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	chatID := c.Param("id")
	chat, err := h.chatService.GetChat(c.Request.Context(), chatID)
	if err != nil {
		h.logger.Error("Failed to get chat", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chat": chat})
}

// DeleteChat deletes a chat
func (h *Handler) DeleteChat(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	chatID := c.Param("id")
	if err := h.chatService.DeleteChat(c.Request.Context(), chatID, userID); err != nil {
		h.logger.Error("Failed to delete chat", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat deleted"})
}

// ListMessages returns messages in a chat
func (h *Handler) ListMessages(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	chatID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	before := c.Query("before")

	messages, err := h.chatService.ListMessages(c.Request.Context(), chatID, before, limit)
	if err != nil {
		h.logger.Error("Failed to list messages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"has_more": len(messages) == limit,
	})
}

// SendMessage sends a message to a chat
func (h *Handler) SendMessage(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	chatID := c.Param("id")
	var req struct {
		Content   string            `json:"content"`
		Type      string            `json:"type"` // "text", "image", "video", "voice", "file"
		MediaID   string            `json:"media_id"`
		ReplyTo   string            `json:"reply_to"`
		Metadata  map[string]string `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	message, err := h.chatService.SendMessage(c.Request.Context(), chatID, userID, req.Content, req.Type, req.ReplyTo)
	if err != nil {
		h.logger.Error("Failed to send message", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Broadcast to WebSocket clients
	h.hub.Broadcast <- &websocket.Message{
		Type:    "new_message",
		Payload: message,
		Room:    chatID,
	}

	c.JSON(http.StatusCreated, gin.H{"message": message})
}

// EditMessage edits a message
func (h *Handler) EditMessage(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	chatID := c.Param("id")
	msgID := c.Param("msgId")

	var req struct {
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	message, err := h.chatService.EditMessage(c.Request.Context(), msgID, userID, req.Content)
	if err != nil {
		h.logger.Error("Failed to edit message", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to edit message"})
		return
	}

	// Broadcast update
	h.hub.Broadcast <- &websocket.Message{
		Type:    "message_edited",
		Payload: message,
		Room:    chatID,
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// DeleteMessage deletes a message
func (h *Handler) DeleteMessage(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	chatID := c.Param("id")
	msgID := c.Param("msgId")

	if err := h.chatService.DeleteMessage(c.Request.Context(), msgID, userID); err != nil {
		h.logger.Error("Failed to delete message", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete message"})
		return
	}

	// Broadcast deletion
	h.hub.Broadcast <- &websocket.Message{
		Type:    "message_deleted",
		Payload: gin.H{"message_id": msgID},
		Room:    chatID,
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message deleted"})
}

// AddReaction adds a reaction to a message
func (h *Handler) AddReaction(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	chatID := c.Param("id")
	msgID := c.Param("msgId")

	var req struct {
		Emoji string `json:"emoji"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.chatService.AddReaction(c.Request.Context(), msgID, userID, req.Emoji); err != nil {
		h.logger.Error("Failed to add reaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add reaction"})
		return
	}

	// Broadcast reaction
	h.hub.Broadcast <- &websocket.Message{
		Type:    "reaction_added",
		Payload: gin.H{"message_id": msgID, "user_id": userID, "emoji": req.Emoji},
		Room:    chatID,
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reaction added"})
}

// RemoveReaction removes a reaction from a message
func (h *Handler) RemoveReaction(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	chatID := c.Param("id")
	msgID := c.Param("msgId")

	if err := h.chatService.RemoveReaction(c.Request.Context(), msgID, userID); err != nil {
		h.logger.Error("Failed to remove reaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove reaction"})
		return
	}

	// Broadcast removal
	h.hub.Broadcast <- &websocket.Message{
		Type:    "reaction_removed",
		Payload: gin.H{"message_id": msgID, "user_id": userID},
		Room:    chatID,
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reaction removed"})
}

// SendTypingIndicator sends typing indicator
func (h *Handler) SendTypingIndicator(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	chatID := c.Param("id")

	// Broadcast typing indicator
	h.hub.Broadcast <- &websocket.Message{
		Type:    "typing",
		Payload: gin.H{"user_id": userID, "chat_id": chatID},
		Room:    chatID,
	}

	c.JSON(http.StatusOK, gin.H{"message": "Typing indicator sent"})
}

// UploadMedia uploads media file
func (h *Handler) UploadMedia(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File required"})
		return
	}
	defer file.Close()

	mediaInfo, err := h.mediaService.Upload(c.Request.Context(), file, header, userID)
	if err != nil {
		h.logger.Error("Failed to upload media", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload media"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"media": mediaInfo})
}

// GetMedia returns media file
func (h *Handler) GetMedia(c *gin.Context) {
	mediaID := c.Param("id")
	
	url, err := h.mediaService.GetURL(c.Request.Context(), mediaID)
	if err != nil {
		h.logger.Error("Failed to get media", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// DeleteMedia deletes media file
func (h *Handler) DeleteMedia(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	mediaID := c.Param("id")
	if err := h.mediaService.Delete(c.Request.Context(), mediaID, userID); err != nil {
		h.logger.Error("Failed to delete media", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Media deleted"})
}

// Search searches across chats and messages
func (h *Handler) Search(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query required"})
		return
	}

	results, err := h.chatService.Search(c.Request.Context(), userID, query)
	if err != nil {
		h.logger.Error("Failed to search", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"query":   query,
	})
}
