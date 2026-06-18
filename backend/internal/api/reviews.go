package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"petstore/internal/models"
)

func (h *Handler) ListReviews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, limit := paginate(r)
	filter := bson.M{}
	if pid := r.URL.Query().Get("productId"); pid != "" {
		if id, err := primitive.ObjectIDFromHex(pid); err == nil {
			filter["product.id"] = id
		}
	}
	total, _ := h.S.Reviews.CountDocuments(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip((page - 1) * limit).SetLimit(limit)
	cur, err := h.S.Reviews.Find(ctx, filter, opts)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var out []models.Review
	if err := cur.All(ctx, &out); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Data: out, Page: page, Limit: limit, Total: total})
}

func (h *Handler) GetReview(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var rev models.Review
	if err := h.S.Reviews.FindOne(r.Context(), bson.M{"_id": id}).Decode(&rev); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, rev)
}

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	var rev models.Review
	if err := json.NewDecoder(r.Body).Decode(&rev); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if rev.Rating < 1 || rev.Rating > 5 {
		writeErr(w, http.StatusBadRequest, "rating must be 1..5")
		return
	}
	now := time.Now().UTC()
	rev.ID = primitive.NilObjectID
	rev.CreatedAt = now
	rev.UpdatedAt = now

	res, err := h.S.Reviews.InsertOne(r.Context(), rev)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	rev.ID = res.InsertedID.(primitive.ObjectID)
	// computed pattern: refresh the product's denormalized ratingSummary
	if !rev.Product.ID.IsZero() {
		_ = h.recomputeProductRating(r.Context(), rev.Product.ID)
	}
	writeJSON(w, http.StatusCreated, rev)
}

func (h *Handler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body models.Review
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	set := bson.M{
		"rating":    body.Rating,
		"title":     body.Title,
		"body":      body.Body,
		"verified":  body.Verified,
		"updatedAt": time.Now().UTC(),
	}
	_, err = h.S.Reviews.UpdateByID(r.Context(), id, bson.M{"$set": set})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var rev models.Review
	_ = h.S.Reviews.FindOne(r.Context(), bson.M{"_id": id}).Decode(&rev)
	if !rev.Product.ID.IsZero() {
		_ = h.recomputeProductRating(r.Context(), rev.Product.ID)
	}
	writeJSON(w, http.StatusOK, rev)
}

func (h *Handler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var rev models.Review
	_ = h.S.Reviews.FindOne(r.Context(), bson.M{"_id": id}).Decode(&rev)
	if _, err := h.S.Reviews.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !rev.Product.ID.IsZero() {
		_ = h.recomputeProductRating(r.Context(), rev.Product.ID)
	}
	w.WriteHeader(http.StatusNoContent)
}
