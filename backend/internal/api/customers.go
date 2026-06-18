package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"petstore/internal/models"
)

func (h *Handler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, limit := paginate(r)
	filter := bson.M{}
	if q := r.URL.Query().Get("q"); q != "" {
		filter["$or"] = bson.A{
			bson.M{"firstName": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"lastName": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"email": bson.M{"$regex": q, "$options": "i"}},
		}
	}
	total, _ := h.S.Customers.CountDocuments(ctx, filter)
	opts := options.Find().
		SetProjection(bson.M{"passwordHash": 0}).
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip((page - 1) * limit).SetLimit(limit)
	cur, err := h.S.Customers.Find(ctx, filter, opts)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var out []models.Customer
	if err := cur.All(ctx, &out); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Data: out, Page: page, Limit: limit, Total: total})
}

func (h *Handler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var c models.Customer
	if err := h.S.Customers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&c); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

type customerInput struct {
	models.Customer
	Password string `json:"password"`
}

func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var in customerInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	c := in.Customer
	c.ID = primitive.NilObjectID
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	if c.Role == "" {
		c.Role = "customer"
	}
	if c.Addresses == nil {
		c.Addresses = []models.Address{}
	}
	pw := in.Password
	if pw == "" {
		pw = "changeme123"
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	c.PasswordHash = string(hash)
	c.Stats = models.CustomerStats{}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	res, err := h.S.Customers.InsertOne(r.Context(), c)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	c.ID = res.InsertedID.(primitive.ObjectID)
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body models.Customer
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Addresses == nil {
		body.Addresses = []models.Address{}
	}
	set := bson.M{
		"firstName": body.FirstName,
		"lastName":  body.LastName,
		"email":     strings.ToLower(strings.TrimSpace(body.Email)),
		"phone":     body.Phone,
		"role":      body.Role,
		"addresses": body.Addresses,
		"updatedAt": time.Now().UTC(),
	}
	_, err = h.S.Customers.UpdateByID(r.Context(), id, bson.M{"$set": set})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.GetCustomer(w, r)
}

func (h *Handler) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := h.S.Customers.DeleteOne(r.Context(), bson.M{"_id": id}); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
