package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"petstore/internal/models"
)

func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "dev-insecure-secret-change-me"
	}
	return []byte(s)
}

type registerReq struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResp struct {
	Token    string           `json:"token"`
	Customer *models.Customer `json:"customer"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || len(req.Password) < 6 {
		writeErr(w, http.StatusBadRequest, "email required and password must be >= 6 chars")
		return
	}

	count, _ := h.S.Customers.CountDocuments(r.Context(), bson.M{"email": req.Email})
	if count > 0 {
		writeErr(w, http.StatusConflict, "email already registered")
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	now := time.Now().UTC()
	c := models.Customer{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         "customer",
		Addresses:    []models.Address{},
		Stats:        models.CustomerStats{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	res, err := h.S.Customers.InsertOne(r.Context(), c)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not create user")
		return
	}
	c.ID = res.InsertedID.(primitive.ObjectID)

	token, _ := issueToken(c.ID.Hex(), c.Email, c.Role)
	writeJSON(w, http.StatusCreated, authResp{Token: token, Customer: &c})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var c models.Customer
	err := h.S.Customers.FindOne(r.Context(), bson.M{"email": req.Email}).Decode(&c)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(req.Password)) != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, _ := issueToken(c.ID.Hex(), c.Email, c.Role)
	writeJSON(w, http.StatusOK, authResp{Token: token, Customer: &c})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	uid, _ := r.Context().Value(ctxUserID).(string)
	id, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var c models.Customer
	if err := h.S.Customers.FindOne(r.Context(), bson.M{"_id": id}).Decode(&c); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func issueToken(uid, email, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   uid,
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(72 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(jwtSecret())
}

type ctxKey string

const (
	ctxUserID ctxKey = "uid"
	ctxRole   ctxKey = "role"
)

// AuthMiddleware validates the Bearer token and injects user info into context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			writeErr(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		tok, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			return jwtSecret(), nil
		})
		if err != nil || !tok.Valid {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		claims, _ := tok.Claims.(jwt.MapClaims)
		ctx := context.WithValue(r.Context(), ctxUserID, claims["sub"])
		ctx = context.WithValue(ctx, ctxRole, claims["role"])
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
