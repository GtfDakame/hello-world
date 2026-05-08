package chat

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestCreateChat(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	members := []string{"user1", "user2", "user3"}

	chat, err := svc.CreateChat(ctx, ChatTypeGroup, "user1", "Test Group", members)
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	if chat.Type != ChatTypeGroup {
		t.Errorf("Expected type '%s', got '%s'", ChatTypeGroup, chat.Type)
	}

	if chat.Name != "Test Group" {
		t.Errorf("Expected name 'Test Group', got '%s'", chat.Name)
	}

	if len(chat.Members) != 3 {
		t.Errorf("Expected 3 members, got %d", len(chat.Members))
	}

	if chat.Admins[0] != "user1" {
		t.Error("Creator should be an admin")
	}
}

func TestCreateChatGroupSizeLimit(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	// Create members array exceeding limit
	members := make([]string, MaxGroupSize+1)
	for i := 0; i < MaxGroupSize+1; i++ {
		members[i] = string(rune('a' + i))
	}

	_, err := svc.CreateChat(ctx, ChatTypeGroup, "user1", "Large Group", members)
	if err == nil {
		t.Error("Expected error for exceeding group size limit")
	}
}

func TestSendMessage(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	msg, err := svc.SendMessage(ctx, chat.ID, "user1", "Hello, World!", MessageTypeText)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	if msg.Content != "Hello, World!" {
		t.Errorf("Expected content 'Hello, World!', got '%s'", msg.Content)
	}

	if msg.MessageType != MessageTypeText {
		t.Errorf("Expected type '%s', got '%s'", MessageTypeText, msg.MessageType)
	}

	if msg.SenderID != "user1" {
		t.Errorf("Expected sender 'user1', got '%s'", msg.SenderID)
	}

	// Verify chat's last message was updated
	updatedChat, err := svc.GetChat(chat.ID)
	if err != nil {
		t.Fatalf("Failed to get chat: %v", err)
	}

	if updatedChat.LastMessageID != msg.ID {
		t.Error("Chat's last message ID should be updated")
	}
}

func TestSendMessageNonMember(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	_, err = svc.SendMessage(ctx, chat.ID, "user3", "Hello!", MessageTypeText)
	if err == nil {
		t.Error("Expected error when non-member sends message")
	}
}

func TestSendMessageTooLong(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	// Create message exceeding max length
	longContent := ""
	for i := 0; i < MaxMessageLength+1; i++ {
		longContent += "a"
	}

	_, err = svc.SendMessage(ctx, chat.ID, "user1", longContent, MessageTypeText)
	if err == nil {
		t.Error("Expected error for message exceeding max length")
	}
}

func TestEditMessage(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	msg, _ := svc.SendMessage(ctx, chat.ID, "user1", "Original", MessageTypeText)

	editedMsg, err := svc.EditMessage(msg.ID, "user1", "Edited content")
	if err != nil {
		t.Fatalf("Failed to edit message: %v", err)
	}

	if editedMsg.Content != "Edited content" {
		t.Errorf("Expected edited content, got '%s'", editedMsg.Content)
	}

	if editedMsg.EditedAt == nil {
		t.Error("EditedAt should be set")
	}

	if editedMsg.Version != 2 {
		t.Errorf("Expected version 2, got %d", editedMsg.Version)
	}
}

func TestEditMessageNotOwner(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	msg, _ := svc.SendMessage(ctx, chat.ID, "user1", "Original", MessageTypeText)

	_, err = svc.EditMessage(msg.ID, "user2", "Edited by other")
	if err == nil {
		t.Error("Expected error when editing another user's message")
	}
}

func TestDeleteMessage(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	msg, _ := svc.SendMessage(ctx, chat.ID, "user1", "To delete", MessageTypeText)

	err = svc.DeleteMessage(msg.ID, "user1", false)
	if err != nil {
		t.Fatalf("Failed to delete message: %v", err)
	}

	// Find and verify message is deleted
	deletedMsg, _ := svc.findMessage(msg.ID)
	if deletedMsg.DeletedAt == nil {
		t.Error("DeletedAt should be set after deletion")
	}
}

func TestAddReaction(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	msg, _ := svc.SendMessage(ctx, chat.ID, "user1", "React to this", MessageTypeText)

	err = svc.AddReaction(msg.ID, "user2", "👍")
	if err != nil {
		t.Fatalf("Failed to add reaction: %v", err)
	}

	// Verify reaction was added
	updatedMsg, _ := svc.findMessage(msg.ID)
	if len(updatedMsg.Reactions["👍"]) != 1 {
		t.Error("Expected 1 reaction")
	}

	if updatedMsg.Reactions["👍"][0] != "user2" {
		t.Error("Expected reaction from user2")
	}
}

func TestAddReactionReplaceExisting(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	msg, _ := svc.SendMessage(ctx, chat.ID, "user1", "React to this", MessageTypeText)

	// Add first reaction
	svc.AddReaction(msg.ID, "user2", "👍")
	// Replace with different reaction
	svc.AddReaction(msg.ID, "user2", "❤️")

	updatedMsg, _ := svc.findMessage(msg.ID)

	if len(updatedMsg.Reactions) != 1 {
		t.Error("Expected only 1 reaction type after replacement")
	}

	if _, exists := updatedMsg.Reactions["👍"]; exists {
		t.Error("Old reaction should be removed")
	}

	if len(updatedMsg.Reactions["❤️"]) != 1 {
		t.Error("Expected 1 heart reaction")
	}
}

func TestSubscribeToChat(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	ch := svc.SubscribeToChat(chat.ID)

	// Send message
	_, err = svc.SendMessage(ctx, chat.ID, "user1", "Test", MessageTypeText)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Should receive message on channel
	select {
	case msg := <-ch:
		if msg.Content != "Test" {
			t.Error("Received message content mismatch")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected to receive message on subscription channel")
	}

	svc.UnsubscribeFromChat(chat.ID, ch)
}

func TestGetMessages(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	// Send multiple messages
	for i := 0; i < 5; i++ {
		_, _ = svc.SendMessage(ctx, chat.ID, "user1", string(rune('A'+i)), MessageTypeText)
	}

	// Get messages with limit
	messages, err := svc.GetMessages(chat.ID, 3, "")
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Messages should be in reverse chronological order
	if messages[0].Content != "E" {
		t.Errorf("Expected latest message 'E', got '%s'", messages[0].Content)
	}
}

func TestGetMessagesPagination(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	// Send messages
	var msgIDs []string
	for i := 0; i < 10; i++ {
		msg, _ := svc.SendMessage(ctx, chat.ID, "user1", string(rune('A'+i%26)), MessageTypeText)
		msgIDs = append(msgIDs, msg.ID)
	}

	// Get messages before a specific message
	messages, err := svc.GetMessages(chat.ID, 3, msgIDs[7])
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}
}

func TestUpdateTypingStatus(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	chat, err := svc.CreateChat(ctx, ChatType1to1, "user1", "Direct Chat", []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("Failed to create chat: %v", err)
	}

	// Set typing status
	svc.UpdateTypingStatus(chat.ID, "user1", true)

	key := chat.ID + ":user1"
	if _, exists := svc.typingStatus[key]; !exists {
		t.Error("Typing status should be set")
	}

	// Clear typing status
	svc.UpdateTypingStatus(chat.ID, "user1", false)

	if _, exists := svc.typingStatus[key]; exists {
		t.Error("Typing status should be cleared")
	}
}

func TestChannelOnlyAdminsCanPost(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewService(logger)

	ctx := context.Background()
	channel, err := svc.CreateChat(ctx, ChatTypeChannel, "admin1", "News Channel", []string{"admin1", "subscriber1", "subscriber2"})
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	// Admin can post
	_, err = svc.SendMessage(ctx, channel.ID, "admin1", "News!", MessageTypeText)
	if err != nil {
		t.Errorf("Admin should be able to post: %v", err)
	}

	// Subscriber cannot post
	_, err = svc.SendMessage(ctx, channel.ID, "subscriber1", "Comment", MessageTypeText)
	if err == nil {
		t.Error("Subscriber should not be able to post in channel")
	}
}
