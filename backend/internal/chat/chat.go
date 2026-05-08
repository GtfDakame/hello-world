// Package chat implements core chat functionality: 1:1, groups, channels
package chat

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	// ChatType1to1 for direct messages
	ChatType1to1 = "1to1"
	// ChatTypeGroup for group chats (up to 200 members)
	ChatTypeGroup = "group"
	// ChatTypeChannel for broadcast channels (up to 10k subscribers)
	ChatTypeChannel = "channel"

	// MessageTypeText for text messages
	MessageTypeText = "text"
	// MessageTypeImage for image messages
	MessageTypeImage = "image"
	// MessageTypeVideo for video messages
	MessageTypeVideo = "video"
	// MessageTypeAudio for audio/voice messages
	MessageTypeAudio = "audio"
	// MessageTypeFile for document messages
	MessageTypeFile = "file"
	// MessageTypeReaction for reaction updates
	MessageTypeReaction = "reaction"

	// MaxGroupSize maximum members in a group
	MaxGroupSize = 200
	// MaxChannelSize maximum subscribers in a channel
	MaxChannelSize = 10000
	// MaxMessageLength maximum characters in a text message
	MaxMessageLength = 4096
	// MaxForwardDepth maximum times a message can be forwarded
	MaxForwardDepth = 5
)

// Chat represents a chat conversation
type Chat struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"` // 1to1, group, channel
	Name           string            `json:"name,omitempty"`
	Description    string            `json:"description,omitempty"`
	AvatarURL      string            `json:"avatar_url,omitempty"`
	CreatedBy      string            `json:"created_by"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	LastMessageID  string            `json:"last_message_id,omitempty"`
	LastReadMsgID  string            `json:"last_read_msg_id,omitempty"`
	UnreadCount    int               `json:"unread_count"`
	IsMuted        bool              `json:"is_muted"`
	IsPinned       bool              `json:"is_pinned"`
	MuteUntil      *time.Time        `json:"mute_until,omitempty"`
	Members        []string          `json:"members,omitempty"` // User IDs
	Admins         []string          `json:"admins,omitempty"`  // User IDs
	Settings       ChatSettings      `json:"settings"`
	Version        int64             `json:"version"` // For conflict resolution
}

// ChatSettings holds chat-specific settings
type ChatSettings struct {
	AutoDeleteTimer    time.Duration `json:"auto_delete_timer"`
	DisappearingMessages bool        `json:"disappearing_messages"`
	OnlyAdminsCanPost  bool          `json:"only_admins_can_post"` // For channels
	JoinApprovalRequired bool        `json:"join_approval_required"`
}

// Message represents a chat message
type Message struct {
	ID            string                 `json:"id"`
	ChatID        string                 `json:"chat_id"`
	SenderID      string                 `json:"sender_id"`
	Content       string                 `json:"content,omitempty"`
	MessageType   string                 `json:"message_type"`
	MediaURL      string                 `json:"media_url,omitempty"`
	MediaThumbnail string                `json:"media_thumbnail,omitempty"`
	MediaSize     int64                  `json:"media_size,omitempty"`
	MediaDuration int                    `json:"media_duration,omitempty"` // For audio/video in seconds
	ReplyTo       string                 `json:"reply_to,omitempty"`       // Message ID being replied to
	ForwardedFrom string                 `json:"forwarded_from,omitempty"` // Original message ID
	ForwardedCount int                   `json:"forwarded_count"`
	Reactions     map[string][]string    `json:"reactions"` // emoji -> [user_ids]
	EditedAt      *time.Time             `json:"edited_at,omitempty"`
	DeletedAt     *time.Time             `json:"deleted_at,omitempty"`
	SentAt        time.Time              `json:"sent_at"`
	DeliveredAt   *time.Time             `json:"delivered_at,omitempty"`
	ReadAt        *time.Time             `json:"read_at,omitempty"`
	Version       int64                  `json:"version"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
}

// TypingStatus represents typing indicator state
type TypingStatus struct {
	ChatID   string    `json:"chat_id"`
	UserID   string    `json:"user_id"`
	IsTyping bool      `json:"is_typing"`
	Timestamp time.Time `json:"timestamp"`
}

// Reaction represents a message reaction
type Reaction struct {
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Emoji     string    `json:"emoji"`
	Timestamp time.Time `json:"timestamp"`
}

// Member represents a chat member
type Member struct {
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"` // admin, member, subscriber
	JoinedAt    time.Time `json:"joined_at"`
	LastReadMsg string    `json:"last_read_msg,omitempty"`
	IsMuted     bool      `json:"is_muted"`
	CustomTitle string    `json:"custom_title,omitempty"`
}

// Service handles chat operations
type Service struct {
	logger        *zap.Logger
	mu            sync.RWMutex
	chats         map[string]*Chat
	messages      map[string][]*Message // chat_id -> messages
	typingStatus  map[string]*TypingStatus
	subscribers   map[string][]chan *Message // chat_id -> subscriber channels
}

// NewService creates a new chat service
func NewService(logger *zap.Logger) *Service {
	return &Service{
		logger:       logger,
		chats:        make(map[string]*Chat),
		messages:     make(map[string][]*Message),
		typingStatus: make(map[string]*TypingStatus),
		subscribers:  make(map[string][]chan *Message),
	}
}

// CreateChat creates a new chat
func (s *Service) CreateChat(ctx context.Context, chatType, createdBy, name string, members []string) (*Chat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate member count based on chat type
	if chatType == ChatTypeGroup && len(members) > MaxGroupSize {
		return nil, errors.New("group size exceeds maximum limit")
	}
	if chatType == ChatTypeChannel && len(members) > MaxChannelSize {
		return nil, errors.New("channel size exceeds maximum limit")
	}

	chat := &Chat{
		ID:        generateID(), // Implement proper ID generation
		Type:      chatType,
		Name:      name,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Members:   members,
		Admins:    []string{createdBy},
		Settings: ChatSettings{
			OnlyAdminsCanPost: chatType == ChatTypeChannel,
		},
		Version: 1,
	}

	s.chats[chat.ID] = chat
	s.messages[chat.ID] = make([]*Message, 0)
	s.subscribers[chat.ID] = make([]chan *Message, 0)

	s.logger.Info("Chat created", zap.String("chat_id", chat.ID), zap.String("type", chatType))
	return chat, nil
}

// GetChat retrieves a chat by ID
func (s *Service) GetChat(chatID string) (*Chat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chat, exists := s.chats[chatID]
	if !exists {
		return nil, errors.New("chat not found")
	}
	return chat, nil
}

// SendMessage sends a message to a chat
func (s *Service) SendMessage(ctx context.Context, chatID, senderID, content, messageType string) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chat, exists := s.chats[chatID]
	if !exists {
		return nil, errors.New("chat not found")
	}

	// Verify sender is a member
	isMember := false
	for _, m := range chat.Members {
		if m == senderID {
			isMember = true
			break
		}
	}
	if !isMember {
		return nil, errors.New("user is not a member of this chat")
	}

	// Check if only admins can post (for channels)
	if chat.Settings.OnlyAdminsCanPost {
		isAdmin := false
		for _, a := range chat.Admins {
			if a == senderID {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			return nil, errors.New("only admins can post in this channel")
		}
	}

	// Validate message length for text messages
	if messageType == MessageTypeText && len(content) > MaxMessageLength {
		return nil, errors.New("message too long")
	}

	msg := &Message{
		ID:           generateID(),
		ChatID:       chatID,
		SenderID:     senderID,
		Content:      content,
		MessageType:  messageType,
		SentAt:       time.Now(),
		Reactions:    make(map[string][]string),
		ForwardedCount: 0,
		Version:      1,
	}

	// Update chat's last message
	chat.LastMessageID = msg.ID
	chat.UpdatedAt = msg.SentAt
	chat.Version++

	// Store message
	s.messages[chatID] = append(s.messages[chatID], msg)

	// Notify subscribers
	s.notifySubscribers(chatID, msg)

	s.logger.Debug("Message sent", zap.String("message_id", msg.ID), zap.String("chat_id", chatID))
	return msg, nil
}

// EditMessage edits an existing message
func (s *Service) EditMessage(messageID, senderID, newContent string) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg, err := s.findMessage(messageID)
	if err != nil {
		return nil, err
	}

	if msg.SenderID != senderID {
		return nil, errors.New("can only edit your own messages")
	}

	now := time.Now()
	msg.Content = newContent
	msg.EditedAt = &now
	msg.Version++

	s.logger.Debug("Message edited", zap.String("message_id", messageID))
	return msg, nil
}

// DeleteMessage deletes a message (soft delete)
func (s *Service) DeleteMessage(messageID, userID string, forEveryone bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg, err := s.findMessage(messageID)
	if err != nil {
		return err
	}

	// Check permissions
	if msg.SenderID != userID && !forEveryone {
		return errors.New("can only delete your own messages")
	}

	now := time.Now()
	msg.DeletedAt = &now
	msg.Version++

	s.logger.Debug("Message deleted", zap.String("message_id", messageID))
	return nil
}

// AddReaction adds a reaction to a message
func (s *Service) AddReaction(messageID, userID, emoji string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg, err := s.findMessage(messageID)
	if err != nil {
		return err
	}

	// Remove existing reaction from same user
	for e, users := range msg.Reactions {
		newUsers := make([]string, 0)
		for _, u := range users {
			if u != userID {
				newUsers = append(newUsers, u)
			}
		}
		if len(newUsers) == 0 {
			delete(msg.Reactions, e)
		} else {
			msg.Reactions[e] = newUsers
		}
	}

	// Add new reaction
	if _, exists := msg.Reactions[emoji]; !exists {
		msg.Reactions[emoji] = make([]string, 0)
	}
	msg.Reactions[emoji] = append(msg.Reactions[emoji], userID)
	msg.Version++

	s.logger.Debug("Reaction added", zap.String("message_id", messageID), zap.String("emoji", emoji))
	return nil
}

// SubscribeToChat subscribes to real-time message updates
func (s *Service) SubscribeToChat(chatID string) chan *Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan *Message, 64)
	s.subscribers[chatID] = append(s.subscribers[chatID], ch)
	return ch
}

// UnsubscribeFromChat unsubscribes from message updates
func (s *Service) UnsubscribeFromChat(chatID string, ch chan *Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscribers := s.subscribers[chatID]
	for i, c := range subscribers {
		if c == ch {
			close(c)
			s.subscribers[chatID] = append(subscribers[:i], subscribers[i+1:]...)
			break
		}
	}
}

// UpdateTypingStatus updates typing indicator
func (s *Service) UpdateTypingStatus(chatID, userID string, isTyping bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := chatID + ":" + userID
	if isTyping {
		s.typingStatus[key] = &TypingStatus{
			ChatID:    chatID,
			UserID:    userID,
			IsTyping:  true,
			Timestamp: time.Now(),
		}
	} else {
		delete(s.typingStatus, key)
	}
}

// GetMessages retrieves messages for a chat with pagination
func (s *Service) GetMessages(chatID string, limit int, beforeID string) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messages, exists := s.messages[chatID]
	if !exists {
		return nil, errors.New("chat not found")
	}

	result := make([]*Message, 0, limit)
	startIndex := len(messages)

	if beforeID != "" {
		for i, msg := range messages {
			if msg.ID == beforeID {
				startIndex = i
				break
			}
		}
	}

	endIndex := startIndex - limit
	if endIndex < 0 {
		endIndex = 0
	}

	for i := startIndex - 1; i >= endIndex; i-- {
		result = append(result, messages[i])
	}

	return result, nil
}

// Helper methods

func (s *Service) findMessage(messageID string) (*Message, error) {
	for _, messages := range s.messages {
		for _, msg := range messages {
			if msg.ID == messageID {
				return msg, nil
			}
		}
	}
	return nil, errors.New("message not found")
}

func (s *Service) notifySubscribers(chatID string, msg *Message) {
	for _, ch := range s.subscribers[chatID] {
		select {
		case ch <- msg:
		default:
			// Channel full, skip
		}
	}
}

func generateID() string {
	// Simplified ID generation - use UUID in production
	return time.Now().Format("20060102150405.000000")
}
