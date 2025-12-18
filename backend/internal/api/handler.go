package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"caiatech-datalab/backend/internal/models"
)

type HandlerDeps struct {
	DB         *sql.DB
	AdminToken string
}

type Handler struct {
	db         *sql.DB
	adminToken string
}

func NewHandler(deps HandlerDeps) *Handler {
	return &Handler{db: deps.DB, adminToken: deps.AdminToken}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.handleHealthz)

	// datasets
	mux.HandleFunc("GET /api/v1/datasets", h.withCORS(h.handleListDatasets))
	mux.HandleFunc("POST /api/v1/datasets", h.withCORS(h.handleCreateDataset))
	mux.HandleFunc("GET /api/v1/datasets/{id}", h.withCORS(h.handleGetDataset))
	mux.HandleFunc("PATCH /api/v1/datasets/{id}", h.withCORS(h.handleUpdateDataset))
	mux.HandleFunc("DELETE /api/v1/datasets/{id}", h.withCORS(h.handleDeleteDataset))
	mux.HandleFunc("GET /api/v1/datasets/{id}/conversations", h.withCORS(h.handleListDatasetConversations))
	mux.HandleFunc("GET /api/v1/datasets/{id}/items", h.withCORS(h.handleListDatasetItems))
	mux.HandleFunc("POST /api/v1/datasets/{id}/items", h.withCORS(h.handleCreateDatasetItem))

	mux.HandleFunc("GET /api/v1/items/{id}", h.withCORS(h.handleGetDatasetItem))
	mux.HandleFunc("PATCH /api/v1/items/{id}", h.withCORS(h.handleUpdateDatasetItem))
	mux.HandleFunc("DELETE /api/v1/items/{id}", h.withCORS(h.handleDeleteDatasetItem))

	// conversations
	mux.HandleFunc("GET /api/v1/conversations/{id}", h.withCORS(h.handleGetConversation))
	mux.HandleFunc("POST /api/v1/conversations", h.withCORS(h.handleCreateConversation))
	mux.HandleFunc("PATCH /api/v1/conversations/{id}", h.withCORS(h.handleUpdateConversation))
	mux.HandleFunc("DELETE /api/v1/conversations/{id}", h.withCORS(h.handleDeleteConversation))

	// proposals (review workflow)
	mux.HandleFunc("POST /api/v1/proposals", h.withCORS(h.handleCreateProposal))
	mux.HandleFunc("GET /api/v1/proposals", h.withCORS(h.handleListProposalsAdmin))
	mux.HandleFunc("POST /api/v1/proposals/{id}/approve", h.withCORS(h.handleApproveProposal))
	mux.HandleFunc("POST /api/v1/proposals/{id}/reject", h.withCORS(h.handleRejectProposal))

	// export
	mux.HandleFunc("GET /api/v1/export.jsonl", h.withCORS(h.handleExportJSONL))

	return mux
}

func (h *Handler) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,X-Admin-Token")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "ts": time.Now().UTC().Format(time.RFC3339)})
}

// ----------------------------
// Datasets
// ----------------------------

type createDatasetRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

type updateDatasetRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

func (h *Handler) handleListDatasets(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)
	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	items, err := models.ListDatasets(r.Context(), h.db, models.ListDatasetsParams{Query: q, Limit: limit, Offset: offset})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list datasets")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (h *Handler) handleGetDataset(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := models.GetDataset(r.Context(), h.db, id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to get dataset")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleCreateDataset(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	var req createDatasetRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	item, err := models.CreateDataset(r.Context(), h.db, req.Name, req.Description, req.Kind)
	if err != nil {
		if errors.Is(err, models.ErrInvalidInput) {
			writeJSONError(w, http.StatusBadRequest, "invalid dataset")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to create dataset")
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) handleUpdateDataset(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	id, err := parsePathInt64(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req updateDatasetRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	item, err := models.UpdateDataset(r.Context(), h.db, id, req.Name, req.Description, req.Kind)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to update dataset")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleDeleteDataset(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	id, err := parsePathInt64(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := models.DeleteDataset(r.Context(), h.db, id); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to delete dataset")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleListDatasetConversations(w http.ResponseWriter, r *http.Request) {
	datasetID, err := parsePathInt64(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid dataset id")
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	splitText := strings.TrimSpace(r.URL.Query().Get("split"))
	statusText := strings.TrimSpace(r.URL.Query().Get("status"))
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	if splitText == "" {
		splitText = string(models.SplitTrain)
	}
	if statusText == "" {
		statusText = string(models.ConversationStatusApproved)
	}
	split, ok := models.NormalizeSplit(splitText)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid split")
		return
	}
	status, ok := models.NormalizeConversationStatus(statusText)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid status")
		return
	}

	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	items, err := models.ListConversations(r.Context(), h.db, models.ListConversationsParams{
		DatasetID: datasetID,
		Split:     split,
		Status:    status,
		Query:     q,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

	// ----------------------------
	// Conversations
	// ----------------------------

	// ----------------------------
	// Dataset Items (generic JSONB)
	// ----------------------------

	type createDatasetItemRequest struct {
		Data      json.RawMessage `json:"data"`
		SourceRef string          `json:"source_ref"`
	}

	type updateDatasetItemRequest struct {
		Data      *json.RawMessage `json:"data"`
		SourceRef *string          `json:"source_ref"`
	}

	func (h *Handler) handleListDatasetItems(w http.ResponseWriter, r *http.Request) {
		datasetID, err := parsePathInt64(r, "id")
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid dataset id")
			return
		}

		// Ensure dataset exists (so we can return 404 instead of empty list).
		if _, err := models.GetDataset(r.Context(), h.db, datasetID); err != nil {
			if errors.Is(err, models.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to get dataset")
			return
		}

		q := strings.TrimSpace(r.URL.Query().Get("q"))
		limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
		offset := parseIntDefault(r.URL.Query().Get("offset"), 0)
		if limit < 1 {
			limit = 1
		}
		if limit > 200 {
			limit = 200
		}
		if offset < 0 {
			offset = 0
		}

		items, err := models.ListDatasetItems(r.Context(), h.db, models.ListDatasetItemsParams{
			DatasetID: datasetID,
			Query:     q,
			Limit:     limit,
			Offset:    offset,
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list items")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
	}

	func (h *Handler) handleCreateDatasetItem(w http.ResponseWriter, r *http.Request) {
		if !h.isAdmin(r) {
			writeJSONError(w, http.StatusUnauthorized, "admin token required")
			return
		}

		datasetID, err := parsePathInt64(r, "id")
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid dataset id")
			return
		}

		// Ensure dataset exists.
		if _, err := models.GetDataset(r.Context(), h.db, datasetID); err != nil {
			if errors.Is(err, models.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to get dataset")
			return
		}

		var req createDatasetItemRequest
		if err := decodeJSON(r.Body, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		it, err := models.CreateDatasetItem(r.Context(), h.db, datasetID, req.Data, req.SourceRef)
		if err != nil {
			if errors.Is(err, models.ErrInvalidInput) {
				writeJSONError(w, http.StatusBadRequest, "invalid item")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to create item")
			return
		}
		writeJSON(w, http.StatusCreated, it)
	}

	func (h *Handler) handleGetDatasetItem(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		it, err := models.GetDatasetItem(r.Context(), h.db, id)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to get item")
			return
		}
		writeJSON(w, http.StatusOK, it)
	}

	func (h *Handler) handleUpdateDatasetItem(w http.ResponseWriter, r *http.Request) {
		if !h.isAdmin(r) {
			writeJSONError(w, http.StatusUnauthorized, "admin token required")
			return
		}

		id, err := parsePathInt64(r, "id")
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var req updateDatasetItemRequest
		if err := decodeJSON(r.Body, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		existing, err := models.GetDatasetItem(r.Context(), h.db, id)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to get item")
			return
		}

		newData := existing.Data
		if req.Data != nil {
			newData = *req.Data
		}
		newSourceRef := existing.SourceRef
		if req.SourceRef != nil {
			newSourceRef = *req.SourceRef
		}

		updated, err := models.UpdateDatasetItem(r.Context(), h.db, id, newData, newSourceRef)
		if err != nil {
			if errors.Is(err, models.ErrInvalidInput) {
				writeJSONError(w, http.StatusBadRequest, "invalid item")
				return
			}
			if errors.Is(err, models.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to update item")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}

	func (h *Handler) handleDeleteDatasetItem(w http.ResponseWriter, r *http.Request) {
		if !h.isAdmin(r) {
			writeJSONError(w, http.StatusUnauthorized, "admin token required")
			return
		}

		id, err := parsePathInt64(r, "id")
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		if err := models.DeleteDatasetItem(r.Context(), h.db, id); err != nil {
			if errors.Is(err, models.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to delete item")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}

	type upsertConversationRequest struct {
		DatasetID int64            `json:"dataset_id"`
		Split     string           `json:"split"`
	Status    string           `json:"status"`
	Tags      []string         `json:"tags"`
	Source    string           `json:"source"`
	Notes     string           `json:"notes"`
	Messages  []models.Message `json:"messages"`
}

func (h *Handler) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	c, err := models.GetConversation(r.Context(), h.db, id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to get conversation")
		return
	}

	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	var req upsertConversationRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	conv, err := normalizeConversationUpsert(req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	inserted, err := models.InsertConversationWithMessages(r.Context(), tx, conv)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create conversation")
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to commit")
		return
	}

	writeJSON(w, http.StatusCreated, inserted)
}

func (h *Handler) handleUpdateConversation(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	id, err := parsePathInt64(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req upsertConversationRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	conv, err := normalizeConversationUpsert(req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	conv.ID = id

	updated, err := models.UpdateConversation(r.Context(), h.db, conv)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to update conversation")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	id, err := parsePathInt64(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := models.DeleteConversation(r.Context(), h.db, id); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to delete conversation")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func normalizeConversationUpsert(req upsertConversationRequest) (models.Conversation, error) {
	splitText := strings.TrimSpace(req.Split)
	if splitText == "" {
		splitText = string(models.SplitTrain)
	}
	split, ok := models.NormalizeSplit(splitText)
	if !ok {
		return models.Conversation{}, errors.New("invalid split")
	}

	statusText := strings.TrimSpace(req.Status)
	if statusText == "" {
		statusText = string(models.ConversationStatusApproved)
	}
	status, ok := models.NormalizeConversationStatus(statusText)
	if !ok {
		return models.Conversation{}, errors.New("invalid status")
	}

	if req.DatasetID <= 0 {
		return models.Conversation{}, errors.New("dataset_id required")
	}

	msgs := req.Messages
	if len(msgs) == 0 {
		return models.Conversation{}, errors.New("messages required")
	}
	for i := range msgs {
		msgs[i].Content = strings.TrimSpace(msgs[i].Content)
		msgs[i].Name = strings.TrimSpace(msgs[i].Name)
		if msgs[i].Content == "" && status != models.ConversationStatusDraft {
			return models.Conversation{}, errors.New("message content cannot be empty")
		}
		switch msgs[i].Role {
		case models.RoleSystem, models.RoleUser, models.RoleAssistant:
		default:
			return models.Conversation{}, errors.New("invalid role")
		}
		if len(msgs[i].Meta) == 0 {
			msgs[i].Meta = json.RawMessage("{}")
		}
	}

	return models.Conversation{
		DatasetID: req.DatasetID,
		Split:     split,
		Status:    status,
		Tags:      req.Tags,
		Source:    strings.TrimSpace(req.Source),
		Notes:     strings.TrimSpace(req.Notes),
		Messages:  msgs,
	}, nil
}

// ----------------------------
// Proposals
// ----------------------------

type createProposalRequest struct {
	DatasetID int64            `json:"dataset_id"`
	Split     string           `json:"split"`
	Tags      []string         `json:"tags"`
	Source    string           `json:"source"`
	Notes     string           `json:"notes"`
	Messages  []models.Message `json:"messages"`

	// Convenience: allow single-turn submissions.
	User      string `json:"user"`
	Assistant string `json:"assistant"`
	System    string `json:"system"`
}

func (h *Handler) handleCreateProposal(w http.ResponseWriter, r *http.Request) {
	var req createProposalRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	conv, err := normalizeConversationFromProposal(req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	payload, _ := json.Marshal(conv)
	p, err := models.CreateProposal(r.Context(), h.db, payload)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create proposal")
		return
	}

	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) handleListProposalsAdmin(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = models.ProposalStatusPending
	}

	items, err := models.ListProposals(r.Context(), h.db, status)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list proposals")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) handleApproveProposal(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	id, err := parsePathInt64(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	ctx := r.Context()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	proposal, err := models.GetProposalForDecision(ctx, tx, id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "proposal not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load proposal")
		return
	}

	conv, err := decodeConversationPayload(proposal.Payload)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "proposal payload invalid")
		return
	}
	conv.Status = models.ConversationStatusApproved

	inserted, err := models.InsertConversationWithMessages(ctx, tx, conv)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to insert conversation")
		return
	}

	now := time.Now().UTC()
	if err := models.MarkProposalApproved(ctx, tx, id, now); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to mark proposal approved")
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to commit")
		return
	}

	writeJSON(w, http.StatusOK, inserted)
}

func (h *Handler) handleRejectProposal(w http.ResponseWriter, r *http.Request) {
	if !h.isAdmin(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin token required")
		return
	}

	id, err := parsePathInt64(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := models.MarkProposalRejected(r.Context(), h.db, id); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "proposal not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to reject proposal")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func normalizeConversationFromProposal(req createProposalRequest) (models.Conversation, error) {
	splitText := strings.TrimSpace(req.Split)
	if splitText == "" {
		splitText = string(models.SplitTrain)
	}
	split, ok := models.NormalizeSplit(splitText)
	if !ok {
		return models.Conversation{}, errors.New("invalid split")
	}

	datasetID := req.DatasetID
	if datasetID <= 0 {
		return models.Conversation{}, errors.New("dataset_id required")
	}

	msgs := req.Messages
	if len(msgs) == 0 {
		user := strings.TrimSpace(req.User)
		assistant := strings.TrimSpace(req.Assistant)
		system := strings.TrimSpace(req.System)
		if user == "" || assistant == "" {
			return models.Conversation{}, errors.New("messages or (user+assistant) required")
		}
		if system != "" {
			msgs = append(msgs, models.Message{Role: models.RoleSystem, Content: system, Meta: json.RawMessage("{}")})
		}
		msgs = append(msgs,
			models.Message{Role: models.RoleUser, Content: user, Meta: json.RawMessage("{}")},
			models.Message{Role: models.RoleAssistant, Content: assistant, Meta: json.RawMessage("{}")},
		)
	}

	for i := range msgs {
		msgs[i].Content = strings.TrimSpace(msgs[i].Content)
		msgs[i].Name = strings.TrimSpace(msgs[i].Name)
		if len(msgs[i].Meta) == 0 {
			msgs[i].Meta = json.RawMessage("{}")
		}
		switch msgs[i].Role {
		case models.RoleSystem, models.RoleUser, models.RoleAssistant:
		default:
			return models.Conversation{}, errors.New("invalid role")
		}
		if msgs[i].Content == "" {
			return models.Conversation{}, errors.New("message content cannot be empty")
		}
	}

	return models.Conversation{
		DatasetID: datasetID,
		Split:     split,
		Status:    models.ConversationStatusPending,
		Tags:      req.Tags,
		Source:    strings.TrimSpace(req.Source),
		Notes:     strings.TrimSpace(req.Notes),
		Messages:  msgs,
	}, nil
}

func decodeConversationPayload(payload []byte) (models.Conversation, error) {
	var c models.Conversation
	if err := json.Unmarshal(payload, &c); err != nil {
		return models.Conversation{}, err
	}
	if len(c.Messages) == 0 {
		return models.Conversation{}, errors.New("no messages")
	}
	if c.DatasetID <= 0 {
		return models.Conversation{}, errors.New("missing dataset_id")
	}
	return c, nil
}

// ----------------------------
// Export
// ----------------------------

func (h *Handler) handleExportJSONL(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	outType := strings.TrimSpace(q.Get("type"))
	if outType == "" {
		outType = "pairs"
	}

	datasetID := int64(parseIntDefault(q.Get("dataset_id"), 0))
	split := strings.TrimSpace(q.Get("split"))
	status := strings.TrimSpace(q.Get("status"))
	if split == "" {
		split = string(models.SplitTrain)
	}
	if status == "" {
		status = string(models.ConversationStatusApproved)
	}

	includeSystem := parseBoolDefault(q.Get("include_system"), false)
	contextMode := strings.TrimSpace(q.Get("context"))
	if contextMode == "" {
		contextMode = "none" // none|window|full
	}
	contextTurns := parseIntDefault(q.Get("context_turns"), 6)
	if contextTurns < 0 {
		contextTurns = 0
	}
	roleStyle := strings.TrimSpace(q.Get("role_style"))
	if roleStyle == "" {
		roleStyle = "labels" // labels|plain
	}
	maxExamples := parseIntDefault(q.Get("max_examples"), 0)
	if maxExamples < 0 {
		maxExamples = 0
	}

	opts := models.ExportOptions{
		Type:          outType,
		DatasetID:     datasetID,
		Split:         split,
		Status:        status,
		IncludeSystem: includeSystem,
		Context:       contextMode,
		ContextTurns:  contextTurns,
		RoleStyle:     roleStyle,
		MaxExamples:   maxExamples,
	}

	// Validate export mode up-front so we can return a helpful error.
	if opts.Type == "items" || opts.Type == "items_with_meta" {
		if opts.DatasetID <= 0 {
			writeJSONError(w, http.StatusBadRequest, "dataset_id is required for items exports")
			return
		}
	}
	if opts.DatasetID > 0 {
		ds, err := models.GetDataset(r.Context(), h.db, opts.DatasetID)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				writeJSONError(w, http.StatusNotFound, "dataset not found")
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "failed to load dataset")
			return
		}
		isItems := strings.EqualFold(ds.Kind, "items")
		if isItems {
			if opts.Type == "conversations" {
				writeJSONError(w, http.StatusBadRequest, "type=conversations is not valid for items datasets")
				return
			}
		} else {
			if opts.Type == "items" || opts.Type == "items_with_meta" {
				writeJSONError(w, http.StatusBadRequest, "items export types are only valid for items datasets")
				return
			}
		}
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Disposition", "attachment; filename=caiatech-datalab.jsonl")
	if err := models.StreamExport(r.Context(), h.db, w, opts); err != nil {
		// Headers are already set; return a JSON error body anyway for easier debugging in-browser.
		writeJSONError(w, http.StatusInternalServerError, "export failed")
		return
	}
}

// ----------------------------
// Helpers
// ----------------------------

func (h *Handler) isAdmin(r *http.Request) bool {
	if h.adminToken == "" {
		return false
	}
	return r.Header.Get("X-Admin-Token") == h.adminToken
}

func parseIntDefault(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return i
}

func parseBoolDefault(s string, fallback bool) bool {
	if s == "" {
		return fallback
	}
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "1" || s == "true" || s == "yes" || s == "y" {
		return true
	}
	if s == "0" || s == "false" || s == "no" || s == "n" {
		return false
	}
	return fallback
}

func parsePathInt64(r *http.Request, param string) (int64, error) {
	v := r.PathValue(param)
	if v == "" {
		return 0, errors.New("missing")
	}
	return strconv.ParseInt(v, 10, 64)
}

func decodeJSON(body io.Reader, dst any) error {
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"error": msg})
}
