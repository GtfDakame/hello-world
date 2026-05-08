package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// Config holds database configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Database wraps the sql.DB connection pool
type Database struct {
	db     *sql.DB
	logger *zap.Logger
	config Config
}

// NewDatabase creates a new database connection pool
func NewDatabase(ctx context.Context, cfg Config, logger *zap.Logger) (*Database, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:     db,
		logger: logger,
		config: cfg,
	}

	// Run migrations
	if err := database.migrate(ctx); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

// Close closes the database connection pool
func (d *Database) Close() error {
	return d.db.Close()
}

// DB returns the underlying sql.DB for direct access
func (d *Database) DB() *sql.DB {
	return d.db
}

// migrate runs database migrations
func (d *Database) migrate(ctx context.Context) error {
	migrations := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			phone VARCHAR(20) UNIQUE,
			email VARCHAR(255) UNIQUE,
			username VARCHAR(64),
			display_name VARCHAR(128),
			avatar_url TEXT,
			status_message TEXT,
			public_key BYTEA,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			last_seen TIMESTAMPTZ,
			is_online BOOLEAN DEFAULT FALSE
		)`,

		// Sessions table for multi-device support
		`CREATE TABLE IF NOT EXISTS sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			device_id VARCHAR(64) NOT NULL,
			device_name VARCHAR(128),
			access_token_hash BYTEA NOT NULL,
			refresh_token_hash BYTEA,
			public_key BYTEA,
			ip_address INET,
			user_agent TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			expires_at TIMESTAMPTZ,
			last_active TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id, device_id)
		)`,

		// Chats table
		`CREATE TABLE IF NOT EXISTS chats (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			type VARCHAR(20) NOT NULL CHECK (type IN ('direct', 'group', 'channel')),
			name VARCHAR(128),
			avatar_url TEXT,
			creator_id UUID REFERENCES users(id),
			max_members INTEGER DEFAULT 200,
			is_public BOOLEAN DEFAULT FALSE,
			public_link VARCHAR(64) UNIQUE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			deleted_at TIMESTAMPTZ
		)`,

		// Chat participants
		`CREATE TABLE IF NOT EXISTS chat_participants (
			chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role VARCHAR(20) DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
			joined_at TIMESTAMPTZ DEFAULT NOW(),
			left_at TIMESTAMPTZ,
			last_read_message_id UUID,
			notification_settings JSONB DEFAULT '{"muted": false, "sound": "default"}',
			PRIMARY KEY (chat_id, user_id)
		)`,

		// Messages table with CRDT support
		`CREATE TABLE IF NOT EXISTS messages (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
			sender_id UUID NOT NULL REFERENCES users(id),
			content TEXT,
			content_encrypted BYTEA,
			message_type VARCHAR(20) DEFAULT 'text' CHECK (message_type IN ('text', 'image', 'video', 'audio', 'file', 'voice', 'sticker')),
			media_url TEXT,
			media_size BIGINT,
			media_mime_type VARCHAR(64),
			reply_to_id UUID REFERENCES messages(id),
			edit_count INTEGER DEFAULT 0,
			is_deleted BOOLEAN DEFAULT FALSE,
			is_edited BOOLEAN DEFAULT FALSE,
			version_vector JSONB NOT NULL DEFAULT '{}',
			lamport_clock BIGINT DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			deleted_at TIMESTAMPTZ
		)`,

		// Message reactions
		`CREATE TABLE IF NOT EXISTS message_reactions (
			message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			reaction VARCHAR(32) NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			PRIMARY KEY (message_id, user_id)
		)`,

		// Message delivery status
		`CREATE TABLE IF NOT EXISTS message_status (
			message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL CHECK (status IN ('sent', 'delivered', 'read')),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			PRIMARY KEY (message_id, user_id)
		)`,

		// E2EE sessions
		`CREATE TABLE IF NOT EXISTS e2ee_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			peer_id UUID NOT NULL REFERENCES users(id),
			session_data BYTEA NOT NULL,
			version INTEGER DEFAULT 1,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id, peer_id)
		)`,

		// Pre-keys for X3DH
		`CREATE TABLE IF NOT EXISTS pre_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			key_id INTEGER NOT NULL,
			key_type VARCHAR(20) NOT NULL CHECK (key_type IN ('identity', 'signed_prekey', 'one_time_prekey')),
			public_key BYTEA NOT NULL,
			signature BYTEA,
			is_used BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(user_id, key_id, key_type)
		)`,

		// Indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_messages_chat_created ON messages(chat_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_participants_user ON chat_participants(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(access_token_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_pre_keys_user_unused ON pre_keys(user_id, is_used) WHERE is_used = FALSE`,
		`CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
	}

	for i, query := range migrations {
		if _, err := d.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("migration %d failed: %w", i, err)
		}
	}

	d.logger.Info("Database migrations completed successfully")
	return nil
}
