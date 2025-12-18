package models

import (
	"encoding/json"
	"time"
)

type ConversationStatus string

type Split string

type Role string

const (
	SplitTrain Split = "train"
	SplitValid Split = "valid"
	SplitTest  Split = "test"
)

const (
	ConversationStatusDraft    ConversationStatus = "draft"
	ConversationStatusPending  ConversationStatus = "pending"
	ConversationStatusApproved ConversationStatus = "approved"
	ConversationStatusRejected ConversationStatus = "rejected"
	ConversationStatusArchived ConversationStatus = "archived"
)

const (
	ProposalStatusPending  = "pending"
	ProposalStatusApproved = "approved"
	ProposalStatusRejected = "rejected"
)

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Dataset struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`

	ItemCount         int64 `json:"item_count"`
	ConversationCount int64 `json:"conversation_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Conversation struct {
	ID        int64              `json:"id"`
	DatasetID int64              `json:"dataset_id"`
	Split     Split              `json:"split"`
	Status    ConversationStatus `json:"status"`
	Tags      []string           `json:"tags"`
	Source    string             `json:"source"`
	Notes     string             `json:"notes"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`

	MessageCount     int    `json:"message_count,omitempty"`
	PreviewUser      string `json:"preview_user,omitempty"`
	PreviewAssistant string `json:"preview_assistant,omitempty"`

	Messages []Message `json:"messages,omitempty"`
}

type Message struct {
	Role    Role            `json:"role"`
	Content string          `json:"content"`
	Name    string          `json:"name,omitempty"`
	Meta    json.RawMessage `json:"meta,omitempty"`
}
