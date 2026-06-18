package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---------------------------------------------------------------------------
// Schema design notes (embedding-heavy / denormalized + computed pattern)
//
// - CategoryRef / CustomerRef / ProductRef are "extended references": a small
//   denormalized snapshot of the related document is embedded so the most
//   common reads need NO $lookup / no second round-trip.
// - Variants and addresses are embedded arrays (subset pattern) because they
//   are always read together with their parent and are bounded in size.
// - RatingSummary, CustomerStats and Category.ProductCount use the COMPUTED
//   PATTERN: pre-aggregated values are stored on write so reads are O(1).
// ---------------------------------------------------------------------------

// ----- Category -----------------------------------------------------------

type Category struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Slug         string             `bson:"slug" json:"slug"`
	Description  string             `bson:"description" json:"description"`
	Icon         string             `bson:"icon" json:"icon"`
	ProductCount int64              `bson:"productCount" json:"productCount"` // computed pattern
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// CategoryRef is the denormalized snapshot embedded inside products.
type CategoryRef struct {
	ID   primitive.ObjectID `bson:"id" json:"id"`
	Name string             `bson:"name" json:"name"`
	Slug string             `bson:"slug" json:"slug"`
}

// ----- Product ------------------------------------------------------------

type Variant struct {
	SKU   string  `bson:"sku" json:"sku"`
	Size  string  `bson:"size" json:"size"`
	Color string  `bson:"color" json:"color"`
	Price float64 `bson:"price" json:"price"`
	Stock int     `bson:"stock" json:"stock"`
}

// RatingSummary is maintained via the computed pattern from the reviews col.
type RatingSummary struct {
	Avg          float64       `bson:"avg" json:"avg"`
	Count        int64         `bson:"count" json:"count"`
	Distribution map[string]int64 `bson:"distribution" json:"distribution"` // "1".."5" -> count
}

type Product struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name          string             `bson:"name" json:"name"`
	Slug          string             `bson:"slug" json:"slug"`
	Description   string             `bson:"description" json:"description"`
	Brand         string             `bson:"brand" json:"brand"`
	PetType       string             `bson:"petType" json:"petType"` // dog | cat | small-pet
	Category      CategoryRef        `bson:"category" json:"category"`
	BasePrice     float64            `bson:"basePrice" json:"basePrice"`
	Currency      string             `bson:"currency" json:"currency"`
	Variants      []Variant          `bson:"variants" json:"variants"`
	Images        []string           `bson:"images" json:"images"`
	Tags          []string           `bson:"tags" json:"tags"`
	TotalStock    int                `bson:"totalStock" json:"totalStock"` // computed pattern
	RatingSummary RatingSummary      `bson:"ratingSummary" json:"ratingSummary"`
	Active        bool               `bson:"active" json:"active"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// ProductRef is the denormalized snapshot embedded inside reviews.
type ProductRef struct {
	ID    primitive.ObjectID `bson:"id" json:"id"`
	Name  string             `bson:"name" json:"name"`
	Slug  string             `bson:"slug" json:"slug"`
	Image string             `bson:"image" json:"image"`
}

// ----- Customer -----------------------------------------------------------

type Address struct {
	Label   string `bson:"label" json:"label"`
	Line1   string `bson:"line1" json:"line1"`
	City    string `bson:"city" json:"city"`
	State   string `bson:"state" json:"state"`
	Zip     string `bson:"zip" json:"zip"`
	Country string `bson:"country" json:"country"`
	Default bool   `bson:"default" json:"default"`
}

// CustomerStats is maintained via the computed pattern from the orders col.
type CustomerStats struct {
	OrderCount int64   `bson:"orderCount" json:"orderCount"`
	TotalSpent float64 `bson:"totalSpent" json:"totalSpent"`
}

type Customer struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FirstName    string             `bson:"firstName" json:"firstName"`
	LastName     string             `bson:"lastName" json:"lastName"`
	Email        string             `bson:"email" json:"email"`
	Phone        string             `bson:"phone" json:"phone"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	Role         string             `bson:"role" json:"role"` // admin | customer
	Addresses    []Address          `bson:"addresses" json:"addresses"`
	Stats        CustomerStats      `bson:"stats" json:"stats"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// CustomerRef is the denormalized snapshot embedded inside orders.
type CustomerRef struct {
	ID    primitive.ObjectID `bson:"id" json:"id"`
	Name  string             `bson:"name" json:"name"`
	Email string             `bson:"email" json:"email"`
}

// ----- Order --------------------------------------------------------------

// OrderItem embeds a product snapshot (extended reference) so the order is
// immutable historically even if the product later changes.
type OrderItem struct {
	ProductID primitive.ObjectID `bson:"productId" json:"productId"`
	Name      string             `bson:"name" json:"name"`
	SKU       string             `bson:"sku" json:"sku"`
	Size      string             `bson:"size" json:"size"`
	Color     string             `bson:"color" json:"color"`
	Price     float64            `bson:"price" json:"price"`
	Qty       int                `bson:"qty" json:"qty"`
	LineTotal float64            `bson:"lineTotal" json:"lineTotal"` // computed pattern
}

// Pricing holds computed-pattern totals so the order summary needs no math on read.
type Pricing struct {
	Subtotal float64 `bson:"subtotal" json:"subtotal"`
	Tax      float64 `bson:"tax" json:"tax"`
	Shipping float64 `bson:"shipping" json:"shipping"`
	Discount float64 `bson:"discount" json:"discount"`
	Total    float64 `bson:"total" json:"total"`
}

type Order struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderNumber     string             `bson:"orderNumber" json:"orderNumber"`
	Customer        CustomerRef        `bson:"customer" json:"customer"`
	Items           []OrderItem        `bson:"items" json:"items"`
	Pricing         Pricing            `bson:"pricing" json:"pricing"`
	Status          string             `bson:"status" json:"status"` // pending|paid|shipped|delivered|cancelled
	ShippingAddress Address            `bson:"shippingAddress" json:"shippingAddress"`
	PlacedAt        time.Time          `bson:"placedAt" json:"placedAt"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// ----- Review -------------------------------------------------------------

type Review struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Product   ProductRef         `bson:"product" json:"product"`
	Customer  CustomerRef        `bson:"customer" json:"customer"`
	Rating    int                `bson:"rating" json:"rating"`
	Title     string             `bson:"title" json:"title"`
	Body      string             `bson:"body" json:"body"`
	Verified  bool               `bson:"verified" json:"verified"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}
