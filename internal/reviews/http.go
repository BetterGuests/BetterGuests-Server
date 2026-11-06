package reviews

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/BetterGuests/BetterGuests-Server/internal/accounts"
	"github.com/google/uuid"
)

type Creator interface {
	CreateReview(context.Context, uuid.UUID, CreateReviewInput) (Review, error)
}
type HTTPHandler struct{ creator Creator }

func NewHTTPHandler(creator Creator) *HTTPHandler { return &HTTPHandler{creator: creator} }

func (handler *HTTPHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	guestID, ok := accounts.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var input CreateReviewInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return
	}
	review, err := handler.creator.CreateReview(r.Context(), guestID, input)
	switch {
	case errors.Is(err, ErrInvalidReview):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, ErrBookingNotReviewable):
		writeError(w, http.StatusForbidden, "booking is not reviewable")
		return
	case errors.Is(err, ErrReviewAlreadyExists):
		writeError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "could not create review")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(review)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
