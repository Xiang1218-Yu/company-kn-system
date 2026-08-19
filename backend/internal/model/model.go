package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// UUID is the shared primary-key type for every domain entity. Centralizing it
// here keeps the column definition consistent across models and lets GORM
// populate a fresh UUID on creation.
type UUID = uuid.UUID

// Role enumerates the user roles described in the spec. Storing them as typed
// constants prevents magic strings from drifting through the codebase.
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleMember  Role = "member"
)

// User maps to the users table. PasswordHash is stored bcrypt-hashed; the raw
// password never lives in memory after registration/login.
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	Name         string         `gorm:"type:varchar(100);not null" json:"name"`
	Role         Role           `gorm:"type:varchar(50);not null;default:member" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

// BeforeCreate ensures every new user gets a UUID even when the caller forgets
// to set one, which keeps inserts deterministic across the codebase.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// KnowledgeBase maps to knowledge_bases. TeamID scopes ownership; isolation of
// knowledge spaces is enforced at the repository/service layer.
type KnowledgeBase struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	TeamID      *uuid.UUID     `gorm:"type:uuid;index" json:"team_id,omitempty"`
	CreatedBy   uuid.UUID      `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (KnowledgeBase) TableName() string { return "knowledge_bases" }

func (k *KnowledgeBase) BeforeCreate(tx *gorm.DB) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}

// DocStatus enumerates the document lifecycle states from the spec.
type DocStatus string

const (
	DocStatusPending  DocStatus = "pending"
	DocStatusIndexing DocStatus = "indexing"
	DocStatusIndexed  DocStatus = "indexed"
	DocStatusFailed   DocStatus = "failed"
)

// Document maps to the documents table. FilePath points at the object key in
// the storage backend (local file or MinIO object), not a local disk path.
type Document struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	KbID       uuid.UUID      `gorm:"type:uuid;index;not null" json:"kb_id"`
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`
	FilePath   string         `gorm:"type:varchar(500);not null" json:"file_path"`
	FileSize   int64          `gorm:"not null" json:"file_size"`
	FileType   string         `gorm:"type:varchar(20);not null" json:"file_type"`
	Status     DocStatus      `gorm:"type:varchar(20);not null;default:pending" json:"status"`
	ChunkCount int            `gorm:"default:0" json:"chunk_count"`
	UploadedBy uuid.UUID      `gorm:"type:uuid;not null" json:"uploaded_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Document) TableName() string { return "documents" }

func (d *Document) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// Chunk maps to the chunks table. Vector stores the embedding; pgvector treats
// this column as a vector(dim) updated via migrations. ChunkIndex preserves the
// original document order so answers can cite position.
type Chunk struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	DocID      uuid.UUID      `gorm:"type:uuid;index;not null" json:"doc_id"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	Vector     pgvector.Vector `gorm:"type:vector(1536)" json:"-"`
	ChunkIndex int            `gorm:"not null" json:"chunk_index"`
	// Metadata uses GORM's json serializer so a map[string]any marshals into
	// the jsonb column without a custom type.
	Metadata   map[string]any `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

func (Chunk) TableName() string { return "chunks" }

func (c *Chunk) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// Feedback captures whether a user found an answer helpful, tied back to the
// originating QA log for the operations dashboard.
type Feedback string

const (
	FeedbackUp   Feedback = "up"
	FeedbackDown Feedback = "down"
	FeedbackNone Feedback = "none"
)

// QALog maps to qa_logs. Sources stores the cited chunk/document references as
// JSON so the schema stays flexible as citation formats evolve.
type QALog struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       uuid.UUID      `gorm:"type:uuid;index;not null" json:"user_id"`
	KbID         uuid.UUID      `gorm:"type:uuid;index;not null" json:"kb_id"`
	Question     string         `gorm:"type:text;not null" json:"question"`
	Answer       string         `gorm:"type:text" json:"answer"`
	// Sources is a slice of citation refs serialized into the jsonb column via
	// GORM's json serializer.
	Sources      []SourceRef    `gorm:"type:jsonb;serializer:json" json:"sources"`
	Feedback     Feedback       `gorm:"type:varchar(20);default:none" json:"feedback"`
	ResponseTime int            `json:"response_time_ms"`
	CreatedAt    time.Time      `json:"created_at"`
}

// SourceRef is one citation in a QA answer: the document name plus the text
// fragment the model grounded its answer in.
type SourceRef struct {
	DocumentID   uuid.UUID `json:"document_id"`
	DocumentName string    `json:"document_name"`
	Snippet      string    `json:"snippet"`
	ChunkIndex   int       `json:"chunk_index"`
}

func (QALog) TableName() string { return "qa_logs" }

func (q *QALog) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}
