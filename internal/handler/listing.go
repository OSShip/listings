package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/OSShip/listings/internal/events"
	"github.com/OSShip/listings/internal/model"
	"github.com/OSShip/listings/internal/store"
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
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = []model.Listing{}
	}
	WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l, err := h.Store.Get(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	WriteJSON(w, http.StatusOK, l)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	role := r.Header.Get("X-User-Role")
	if userID == "" || role != "mentor" {
		http.Error(w, `{"error":"mentor role required"}`, http.StatusForbidden)
		return
	}

	approved, _ := h.Store.IsMentorApproved(r.Context(), userID)
	if !approved {
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
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	_ = h.Events.PublishListingCreated(r.Context(), created.ID)
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
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	_ = h.Events.PublishListingUpdated(r.Context(), id)
	h.Get(w, r)
}
