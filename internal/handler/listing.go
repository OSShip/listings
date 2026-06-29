package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/OSShip/listings/internal/events"
	"github.com/OSShip/listings/internal/model"
	"github.com/OSShip/listings/internal/store"
	"github.com/OSShip/utils/observability"
)

type Handler struct {
	Store  *store.Store
	Events *events.Publisher
}

func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}
	ossProject := r.URL.Query().Get("oss_project")

	list, err := h.Store.List(r.Context(), status, ossProject)
	if err != nil {
		observability.RespondError(w, r, http.StatusInternalServerError, "internal", "list listings", err, "status", status)
		return
	}
	if list == nil {
		list = []model.Listing{}
	}
	slog.DebugContext(r.Context(), "listings fetched", "count", len(list), "status", status)
	WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l, err := h.Store.Get(r.Context(), id)
	if err != nil {
		slog.InfoContext(r.Context(), "listing not found", "id", id)
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	slog.DebugContext(r.Context(), "listing fetched", "id", id)
	WriteJSON(w, http.StatusOK, l)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	role := r.Header.Get("X-User-Role")
	if userID == "" || role != "mentor" {
		slog.WarnContext(r.Context(), "create listing forbidden", "user_id", userID, "role", role)
		http.Error(w, `{"error":"mentor role required"}`, http.StatusForbidden)
		return
	}

	approved, _ := h.Store.IsMentorApproved(r.Context(), userID)
	if !approved {
		slog.WarnContext(r.Context(), "create listing mentor not approved", "user_id", userID)
		http.Error(w, `{"error":"mentor not approved"}`, http.StatusForbidden)
		return
	}

	var req model.Listing
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	created, err := h.Store.Create(r.Context(), userID, req)
	if err != nil {
		observability.RespondError(w, r, http.StatusInternalServerError, "internal", "create listing", err, "user_id", userID)
		return
	}
	if err := h.Events.PublishListingCreated(r.Context(), created.ID); err != nil {
		slog.WarnContext(r.Context(), "listing created event publish failed", "listing_id", created.ID, "err", err)
	}
	slog.InfoContext(r.Context(), "listing created", "listing_id", created.ID, "mentor_id", userID)
	WriteJSON(w, http.StatusCreated, created)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := r.Header.Get("X-User-Id")
	mentorID, err := h.Store.GetMentorID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if mentorID != userID {
		slog.WarnContext(r.Context(), "update listing forbidden", "listing_id", id, "user_id", userID)
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}
	var req struct {
		Description   string `json:"description"`
		Status        string `json:"status"`
		PriceCents    int    `json:"price_cents"`
		DurationWeeks int    `json:"duration_weeks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if err := h.Store.Update(r.Context(), id, req.Description, req.Status, req.PriceCents, req.DurationWeeks); err != nil {
		observability.RespondError(w, r, http.StatusInternalServerError, "internal", "update listing", err, "listing_id", id)
		return
	}
	if err := h.Events.PublishListingUpdated(r.Context(), id); err != nil {
		slog.WarnContext(r.Context(), "listing updated event publish failed", "listing_id", id, "err", err)
	}
	slog.InfoContext(r.Context(), "listing updated", "listing_id", id, "status", req.Status)
	h.Get(w, r)
}
