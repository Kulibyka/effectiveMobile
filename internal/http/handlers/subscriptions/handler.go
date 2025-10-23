package subscriptions

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	domain "github.com/Kulibyka/effective-mobile/internal/domain/subscription"
	"github.com/Kulibyka/effective-mobile/internal/lib/uuid"
	"github.com/Kulibyka/effective-mobile/internal/services/subscriptions"
)

const (
	basePath    = "/api/v1/subscriptions"
	summaryPath = basePath + "/summary"
)

type Handler struct {
	service *subscriptions.Service
	logger  *slog.Logger
}

func New(service *subscriptions.Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger.WithGroup("subscriptions_http")}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc(summaryPath, h.handleSummary)
	mux.HandleFunc(basePath, h.handleBase)
	mux.HandleFunc(basePath+"/", h.handleWithID)
}

func (h *Handler) handleBase(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handling base route", slog.String("method", r.Method), slog.String("path", r.URL.Path))
	switch r.Method {
	case http.MethodPost:
		h.handleCreate(w, r)
	case http.MethodGet:
		h.handleList(w, r)
	default:
		h.logger.Warn("method not allowed", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleWithID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, basePath+"/")
	if idStr == "" {
		h.logger.Warn("subscription id is required", slog.String("path", r.URL.Path))
		http.NotFound(w, r)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn("failed to parse subscription id", slog.String("subscription_id", idStr), slog.Any("error", err))
		http.Error(w, "invalid subscription id", http.StatusBadRequest)
		return
	}

	h.logger.Debug("handling request with subscription id", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.String("subscription_id", id.String()))
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, id)
	case http.MethodPut:
		h.handleUpdate(w, r, id)
	case http.MethodDelete:
		h.handleDelete(w, r, id)
	default:
		h.logger.Warn("method not allowed", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req subscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("failed to decode create request", slog.Any("error", err))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input, err := req.toCreateInput()
	if err != nil {
		h.logger.Warn("invalid create request", slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Info("creating subscription", slog.String("user_id", input.UserID.String()), slog.String("service_name", input.ServiceName))
	sub, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.logger.Error("failed to create subscription", slog.Any("error", err), slog.String("user_id", input.UserID.String()), slog.String("service_name", input.ServiceName))
		http.Error(w, "failed to create subscription", http.StatusInternalServerError)
		return
	}

	h.logger.Info("subscription created", slog.String("subscription_id", sub.ID.String()))
	writeJSON(w, http.StatusCreated, subscriptionResponseFromDomain(sub))
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	h.logger.Debug("getting subscription", slog.String("subscription_id", id.String()))
	sub, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			h.logger.Warn("subscription not found", slog.String("subscription_id", id.String()))
			http.Error(w, "subscription not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get subscription", slog.Any("error", err), slog.String("subscription_id", id.String()))
		http.Error(w, "failed to get subscription", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("subscription fetched", slog.String("subscription_id", sub.ID.String()))
	writeJSON(w, http.StatusOK, subscriptionResponseFromDomain(sub))
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req subscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("failed to decode update request", slog.String("subscription_id", id.String()), slog.Any("error", err))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input, err := req.toUpdateInput()
	if err != nil {
		h.logger.Warn("invalid update request", slog.String("subscription_id", id.String()), slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Info("updating subscription", slog.String("subscription_id", id.String()))
	sub, err := h.service.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			h.logger.Warn("subscription not found", slog.String("subscription_id", id.String()))
			http.Error(w, "subscription not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to update subscription", slog.Any("error", err), slog.String("subscription_id", id.String()))
		http.Error(w, "failed to update subscription", http.StatusInternalServerError)
		return
	}

	h.logger.Info("subscription updated", slog.String("subscription_id", sub.ID.String()))
	writeJSON(w, http.StatusOK, subscriptionResponseFromDomain(sub))
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	h.logger.Info("deleting subscription", slog.String("subscription_id", id.String()))
	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			h.logger.Warn("subscription not found", slog.String("subscription_id", id.String()))
			http.Error(w, "subscription not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to delete subscription", slog.Any("error", err), slog.String("subscription_id", id.String()))
		http.Error(w, "failed to delete subscription", http.StatusInternalServerError)
		return
	}

	h.logger.Info("subscription deleted", slog.String("subscription_id", id.String()))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	filter, err := parseListFilter(r)
	if err != nil {
		h.logger.Warn("invalid list filter", slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Debug("listing subscriptions", slog.Any("filter", filter))
	subs, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("failed to list subscriptions", slog.Any("error", err), slog.Any("filter", filter))
		http.Error(w, "failed to list subscriptions", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("subscriptions listed", slog.Int("count", len(subs)))
	resp := make([]subscriptionResponse, 0, len(subs))
	for _, sub := range subs {
		resp = append(resp, subscriptionResponseFromDomain(sub))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("method not allowed", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	summaryFilter, err := parseSummaryFilter(r)
	if err != nil {
		h.logger.Warn("invalid summary filter", slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Debug("calculating summary", slog.Any("filter", summaryFilter))
	total, err := h.service.Sum(r.Context(), summaryFilter)
	if err != nil {
		h.logger.Error("failed to calculate summary", slog.Any("error", err), slog.Any("filter", summaryFilter))
		http.Error(w, "failed to calculate summary", http.StatusInternalServerError)
		return
	}

	h.logger.Info("summary calculated", slog.Int("total", total))
	writeJSON(w, http.StatusOK, map[string]int{"total": total})
}

type subscriptionRequest struct {
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

func (r subscriptionRequest) toCreateInput() (domain.CreateInput, error) {
	userID, err := uuid.Parse(r.UserID)
	if err != nil {
		return domain.CreateInput{}, errors.New("invalid user_id")
	}

	start, err := time.Parse(domain.MonthLayout, r.StartDate)
	if err != nil {
		return domain.CreateInput{}, errors.New("invalid start_date format, expected MM-YYYY")
	}

	var end *time.Time
	if r.EndDate != nil {
		if *r.EndDate == "" {
			end = nil
		} else {
			parsed, err := time.Parse(domain.MonthLayout, *r.EndDate)
			if err != nil {
				return domain.CreateInput{}, errors.New("invalid end_date format, expected MM-YYYY")
			}
			end = &parsed
		}
	}

	return domain.CreateInput{
		ServiceName: r.ServiceName,
		Price:       r.Price,
		UserID:      userID,
		StartMonth:  start,
		EndMonth:    end,
	}, nil
}

func (r subscriptionRequest) toUpdateInput() (domain.UpdateInput, error) {
	input, err := r.toCreateInput()
	if err != nil {
		return domain.UpdateInput{}, err
	}

	return domain.UpdateInput{
		ServiceName: input.ServiceName,
		Price:       input.Price,
		StartMonth:  input.StartMonth,
		EndMonth:    input.EndMonth,
	}, nil
}

type subscriptionResponse struct {
	ID          uuid.UUID `json:"id"`
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      uuid.UUID `json:"user_id"`
	StartDate   string    `json:"start_date"`
	EndDate     *string   `json:"end_date,omitempty"`
}

func subscriptionResponseFromDomain(sub domain.Subscription) subscriptionResponse {
	resp := subscriptionResponse{
		ID:          sub.ID,
		ServiceName: sub.ServiceName,
		Price:       sub.Price,
		UserID:      sub.UserID,
		StartDate:   sub.StartMonth.Format(domain.MonthLayout),
	}

	if sub.EndMonth != nil {
		formatted := sub.EndMonth.Format(domain.MonthLayout)
		resp.EndDate = &formatted
	}

	return resp
}

func parseListFilter(r *http.Request) (domain.ListFilter, error) {
	var filter domain.ListFilter

	if userID := r.URL.Query().Get("user_id"); userID != "" {
		parsed, err := uuid.Parse(userID)
		if err != nil {
			return domain.ListFilter{}, errors.New("invalid user_id")
		}
		filter.UserID = &parsed
	}

	if serviceName := r.URL.Query().Get("service_name"); serviceName != "" {
		filter.ServiceName = &serviceName
	}

	if start := r.URL.Query().Get("start_date"); start != "" {
		parsed, err := time.Parse(domain.MonthLayout, start)
		if err != nil {
			return domain.ListFilter{}, errors.New("invalid start_date format, expected MM-YYYY")
		}
		filter.StartMonthFrom = &parsed
	}

	if end := r.URL.Query().Get("end_date"); end != "" {
		parsed, err := time.Parse(domain.MonthLayout, end)
		if err != nil {
			return domain.ListFilter{}, errors.New("invalid end_date format, expected MM-YYYY")
		}
		filter.StartMonthTo = &parsed
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		parsed, err := strconv.Atoi(limit)
		if err != nil || parsed < 0 {
			return domain.ListFilter{}, errors.New("invalid limit")
		}
		filter.Limit = parsed
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		parsed, err := strconv.Atoi(offset)
		if err != nil || parsed < 0 {
			return domain.ListFilter{}, errors.New("invalid offset")
		}
		filter.Offset = parsed
	}

	return filter, nil
}

func parseSummaryFilter(r *http.Request) (domain.SummaryFilter, error) {
	var filter domain.SummaryFilter

	start := r.URL.Query().Get("start_date")
	end := r.URL.Query().Get("end_date")

	if start == "" || end == "" {
		return domain.SummaryFilter{}, errors.New("start_date and end_date are required")
	}

	startMonth, err := time.Parse(domain.MonthLayout, start)
	if err != nil {
		return domain.SummaryFilter{}, errors.New("invalid start_date format, expected MM-YYYY")
	}

	endMonth, err := time.Parse(domain.MonthLayout, end)
	if err != nil {
		return domain.SummaryFilter{}, errors.New("invalid end_date format, expected MM-YYYY")
	}

	if endMonth.Before(startMonth) {
		return domain.SummaryFilter{}, errors.New("end_date must be after start_date")
	}

	filter.PeriodStart = startMonth
	filter.PeriodEnd = endMonth

	if userID := r.URL.Query().Get("user_id"); userID != "" {
		parsed, err := uuid.Parse(userID)
		if err != nil {
			return domain.SummaryFilter{}, errors.New("invalid user_id")
		}
		filter.UserID = &parsed
	}

	if serviceName := r.URL.Query().Get("service_name"); serviceName != "" {
		filter.ServiceName = &serviceName
	}

	return filter, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Default().Error("failed to encode response", slog.Any("error", err))
	}
}
