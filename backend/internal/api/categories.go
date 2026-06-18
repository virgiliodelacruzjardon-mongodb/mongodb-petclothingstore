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

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, limit := paginate(r)
	filter := bson.M{}
	if q := r.URL.Query().Get("q"); q != "" {
		filter["name"] = bson.M{"$regex": q, "$options": "i"}
	}
	total, _ := h.S.Categories.CountDocuments(ctx, filter)
	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).
		SetSkip((page - 1) * limit).SetLimit(limit)
	cur, err := h.S.Categories.Find(ctx, filter, opts)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var out []models.Category
	if err := cur.All(ctx, &out); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Data: out, Page: page, Limit: limit, Total: total})
}

func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var c models.Category
	if err := h.S.Categories.FindOne(r.Context(), bson.M{"_id": id}).Decode(&c); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var c models.Category
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	now := time.Now().UTC()
	c.ID = primitive.NilObjectID
	c.ProductCount = 0
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.Slug == "" {
		c.Slug = slugify(c.Name)
	}
	res, err := h.S.Categories.InsertOne(r.Context(), c)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	c.ID = res.InsertedID.(primitive.ObjectID)
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body models.Category
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	set := bson.M{
		"name":        body.Name,
		"slug":        body.Slug,
		"description": body.Description,
		"icon":        body.Icon,
		"updatedAt":   time.Now().UTC(),
	}
	if body.Slug == "" {
		set["slug"] = slugify(body.Name)
	}
	_, err = h.S.Categories.UpdateByID(r.Context(), id, bson.M{"$set": set})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.GetCategory(w, r)
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	_, err = h.S.Categories.DeleteOne(r.Context(), bson.M{"_id": id})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
