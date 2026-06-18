package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"petstore/internal/db"
	"petstore/internal/models"
)

// Target document counts (>= 200 per collection as requested).
const (
	nCategories = 200
	nProducts   = 300
	nCustomers  = 250
	nOrders     = 400
	nReviews    = 500
)

var (
	petTypes  = []string{"dog", "cat", "small-pet"}
	garments  = []string{"Sweater", "Raincoat", "T-Shirt", "Hoodie", "Costume", "Bandana", "Booties", "Hat", "Pajamas", "Jacket", "Vest", "Bow Tie", "Harness Dress", "Scarf", "Tutu", "Onesie", "Polo Shirt", "Winter Coat", "Cooling Shirt", "Life Vest"}
	colors    = []string{"Red", "Blue", "Green", "Pink", "Black", "Yellow", "Purple", "Orange", "Teal", "Gray"}
	sizes     = []string{"XS", "S", "M", "L", "XL"}
	brands    = []string{"PawCouture", "FurryThreads", "WoofWear", "MeowModa", "TailTrends", "SnugPup", "WhiskerWardrobe", "BarkStyle", "CozyCritters", "PetVogue"}
	materials = []string{"cotton", "fleece", "waterproof", "knit", "denim", "polyester", "wool blend"}
	descWords = []string{"warm", "lightweight", "machine-washable", "breathable", "adjustable", "reflective", "stretchy", "soft", "durable", "cute"}
	statuses  = []string{"pending", "paid", "shipped", "delivered", "cancelled"}
	rtitles   = []string{"Perfect fit!", "My pet loves it", "Great quality", "Could be better", "Adorable", "Runs small", "Worth every penny", "So cozy", "Not as pictured", "Will buy again"}
)

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	store, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}

	log.Println("dropping existing collections...")
	for _, c := range []string{"products", "categories", "customers", "orders", "reviews"} {
		_ = store.DB.Collection(c).Drop(ctx)
	}

	cats := seedCategories(ctx, store)
	prods := seedProducts(ctx, store, cats)
	custs := seedCustomers(ctx, store)
	seedOrders(ctx, store, custs, prods)
	seedReviews(ctx, store, custs, prods)

	log.Println("recomputing computed-pattern fields...")
	recomputeCategoryCounts(ctx, store)
	recomputeProductRatings(ctx, store)
	recomputeCustomerStats(ctx, store)

	log.Println("ensuring indexes...")
	if err := store.EnsureIndexes(ctx); err != nil {
		log.Printf("warning: indexes: %v", err)
	}

	log.Printf("done. categories=%d products=%d customers=%d orders=%d reviews=%d",
		nCategories, nProducts, nCustomers, nOrders, nReviews)
	log.Println("admin login -> email: admin@petstore.dev  password: admin123")
}

// ---------------------------------------------------------------------------

func seedCategories(ctx context.Context, s *db.Store) []models.Category {
	out := make([]models.Category, 0, nCategories)
	docs := make([]interface{}, 0, nCategories)
	seen := map[string]bool{}
	now := time.Now().UTC()

	for len(out) < nCategories {
		pet := petTypes[rand.Intn(len(petTypes))]
		g := garments[rand.Intn(len(garments))]
		name := fmt.Sprintf("%s %ss", strings.Title(pet), g)
		slug := slugify(name)
		if seen[slug] {
			// disambiguate to keep unique slug index happy
			name = fmt.Sprintf("%s %ss %s", strings.Title(pet), g, colors[rand.Intn(len(colors))])
			slug = slugify(name)
			if seen[slug] {
				continue
			}
		}
		seen[slug] = true
		c := models.Category{
			ID:          primitive.NewObjectID(),
			Name:        name,
			Slug:        slug,
			Description: fmt.Sprintf("%s for your beloved %s.", g, pet),
			Icon:        []string{"paw", "shirt", "bone", "heart", "star"}[rand.Intn(5)],
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		out = append(out, c)
		docs = append(docs, c)
	}
	if _, err := s.Categories.InsertMany(ctx, docs); err != nil {
		log.Fatalf("insert categories: %v", err)
	}
	return out
}

func seedProducts(ctx context.Context, s *db.Store, cats []models.Category) []models.Product {
	out := make([]models.Product, 0, nProducts)
	docs := make([]interface{}, 0, nProducts)
	seen := map[string]bool{}
	now := time.Now().UTC()

	for i := 0; len(out) < nProducts; i++ {
		cat := cats[rand.Intn(len(cats))]
		pet := petTypes[rand.Intn(len(petTypes))]
		g := garments[rand.Intn(len(garments))]
		brand := brands[rand.Intn(len(brands))]
		name := fmt.Sprintf("%s %s %s", brand, materials[rand.Intn(len(materials))], g)
		slug := slugify(name) + fmt.Sprintf("-%d", i)
		if seen[slug] {
			continue
		}
		seen[slug] = true

		base := round2(9.99 + rand.Float64()*60)
		// embedded variants (subset pattern)
		variantCount := 2 + rand.Intn(4)
		variants := make([]models.Variant, 0, variantCount)
		totalStock := 0
		usedSize := map[string]bool{}
		for v := 0; v < variantCount; v++ {
			sz := sizes[rand.Intn(len(sizes))]
			col := colors[rand.Intn(len(colors))]
			key := sz + col
			if usedSize[key] {
				continue
			}
			usedSize[key] = true
			stock := rand.Intn(120)
			totalStock += stock
			variants = append(variants, models.Variant{
				SKU:   fmt.Sprintf("%s-%s-%s-%d", strings.ToUpper(brand[:3]), sz, strings.ToUpper(col[:3]), i),
				Size:  sz,
				Color: col,
				Price: round2(base + rand.Float64()*5),
				Stock: stock,
			})
		}

		tags := []string{pet, strings.ToLower(g), materials[rand.Intn(len(materials))], descWords[rand.Intn(len(descWords))]}
		p := models.Product{
			ID:          primitive.NewObjectID(),
			Name:        name,
			Slug:        slug,
			Description: fmt.Sprintf("A %s, %s %s designed for %ss. %s.", descWords[rand.Intn(len(descWords))], descWords[rand.Intn(len(descWords))], g, pet, strings.Title(gofakeit.Sentence(6))),
			Brand:       brand,
			PetType:     pet,
			Category:    models.CategoryRef{ID: cat.ID, Name: cat.Name, Slug: cat.Slug},
			BasePrice:   base,
			Currency:    "USD",
			Variants:    variants,
			Images:      []string{fmt.Sprintf("https://picsum.photos/seed/pet%d/600/600", i)},
			Tags:        tags,
			TotalStock:  totalStock,
			RatingSummary: models.RatingSummary{
				Distribution: map[string]int64{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0},
			},
			Active:    rand.Float64() > 0.1,
			CreatedAt: now.Add(-time.Duration(rand.Intn(365*24)) * time.Hour),
			UpdatedAt: now,
		}
		out = append(out, p)
		docs = append(docs, p)
	}
	if _, err := s.Products.InsertMany(ctx, docs); err != nil {
		log.Fatalf("insert products: %v", err)
	}
	return out
}

func seedCustomers(ctx context.Context, s *db.Store) []models.Customer {
	out := make([]models.Customer, 0, nCustomers)
	docs := make([]interface{}, 0, nCustomers)
	now := time.Now().UTC()

	// deterministic admin account
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	admin := models.Customer{
		ID:           primitive.NewObjectID(),
		FirstName:    "Store",
		LastName:     "Admin",
		Email:        "admin@petstore.dev",
		Phone:        gofakeit.Phone(),
		PasswordHash: string(adminHash),
		Role:         "admin",
		Addresses:    []models.Address{randAddress(true)},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	out = append(out, admin)
	docs = append(docs, admin)

	defaultHash, _ := bcrypt.GenerateFromPassword([]byte("changeme123"), bcrypt.DefaultCost)
	for i := 1; i < nCustomers; i++ {
		first := gofakeit.FirstName()
		last := gofakeit.LastName()
		email := fmt.Sprintf("%s.%s%d@example.com", strings.ToLower(first), strings.ToLower(last), i)
		addrCount := 1 + rand.Intn(2)
		addrs := make([]models.Address, 0, addrCount)
		for a := 0; a < addrCount; a++ {
			addrs = append(addrs, randAddress(a == 0))
		}
		c := models.Customer{
			ID:           primitive.NewObjectID(),
			FirstName:    first,
			LastName:     last,
			Email:        email,
			Phone:        gofakeit.Phone(),
			PasswordHash: string(defaultHash),
			Role:         "customer",
			Addresses:    addrs,
			CreatedAt:    now.Add(-time.Duration(rand.Intn(500*24)) * time.Hour),
			UpdatedAt:    now,
		}
		out = append(out, c)
		docs = append(docs, c)
	}
	if _, err := s.Customers.InsertMany(ctx, docs); err != nil {
		log.Fatalf("insert customers: %v", err)
	}
	return out
}

func randAddress(def bool) models.Address {
	return models.Address{
		Label:   []string{"Home", "Work", "Other"}[rand.Intn(3)],
		Line1:   gofakeit.Street(),
		City:    gofakeit.City(),
		State:   gofakeit.StateAbr(),
		Zip:     gofakeit.Zip(),
		Country: "US",
		Default: def,
	}
}

func seedOrders(ctx context.Context, s *db.Store, custs []models.Customer, prods []models.Product) {
	docs := make([]interface{}, 0, nOrders)
	now := time.Now().UTC()

	for i := 0; i < nOrders; i++ {
		c := custs[rand.Intn(len(custs))]
		itemCount := 1 + rand.Intn(4)
		items := make([]models.OrderItem, 0, itemCount)
		var subtotal float64
		for it := 0; it < itemCount; it++ {
			p := prods[rand.Intn(len(prods))]
			v := p.Variants[rand.Intn(len(p.Variants))]
			qty := 1 + rand.Intn(3)
			line := round2(v.Price * float64(qty))
			subtotal += line
			items = append(items, models.OrderItem{
				ProductID: p.ID,
				Name:      p.Name,
				SKU:       v.SKU,
				Size:      v.Size,
				Color:     v.Color,
				Price:     v.Price,
				Qty:       qty,
				LineTotal: line,
			})
		}
		tax := round2(subtotal * 0.08)
		shipping := 6.99
		if subtotal > 75 {
			shipping = 0
		}
		discount := 0.0
		if rand.Float64() < 0.2 {
			discount = round2(subtotal * 0.1)
		}
		total := round2(subtotal + tax + shipping - discount)

		var addr models.Address
		if len(c.Addresses) > 0 {
			addr = c.Addresses[0]
		} else {
			addr = randAddress(true)
		}
		placed := now.Add(-time.Duration(rand.Intn(400*24)) * time.Hour)
		o := models.Order{
			ID:          primitive.NewObjectID(),
			OrderNumber: fmt.Sprintf("PET-%06d", i+1),
			Customer:    models.CustomerRef{ID: c.ID, Name: c.FirstName + " " + c.LastName, Email: c.Email},
			Items:       items,
			Pricing:     models.Pricing{Subtotal: round2(subtotal), Tax: tax, Shipping: shipping, Discount: discount, Total: total},
			Status:      statuses[rand.Intn(len(statuses))],
			ShippingAddress: addr,
			PlacedAt:    placed,
			CreatedAt:   placed,
			UpdatedAt:   now,
		}
		docs = append(docs, o)
	}
	if _, err := s.Orders.InsertMany(ctx, docs); err != nil {
		log.Fatalf("insert orders: %v", err)
	}
}

func seedReviews(ctx context.Context, s *db.Store, custs []models.Customer, prods []models.Product) {
	docs := make([]interface{}, 0, nReviews)
	now := time.Now().UTC()
	for i := 0; i < nReviews; i++ {
		p := prods[rand.Intn(len(prods))]
		c := custs[rand.Intn(len(custs))]
		// skew ratings toward the positive end
		rating := []int{5, 5, 5, 4, 4, 4, 3, 3, 2, 1}[rand.Intn(10)]
		img := ""
		if len(p.Images) > 0 {
			img = p.Images[0]
		}
		rev := models.Review{
			ID:        primitive.NewObjectID(),
			Product:   models.ProductRef{ID: p.ID, Name: p.Name, Slug: p.Slug, Image: img},
			Customer:  models.CustomerRef{ID: c.ID, Name: c.FirstName + " " + c.LastName, Email: c.Email},
			Rating:    rating,
			Title:     rtitles[rand.Intn(len(rtitles))],
			Body:      gofakeit.Sentence(12),
			Verified:  rand.Float64() > 0.3,
			CreatedAt: now.Add(-time.Duration(rand.Intn(300*24)) * time.Hour),
			UpdatedAt: now,
		}
		docs = append(docs, rev)
	}
	if _, err := s.Reviews.InsertMany(ctx, docs); err != nil {
		log.Fatalf("insert reviews: %v", err)
	}
}

// ---- computed-pattern recalculation -------------------------------------

func recomputeCategoryCounts(ctx context.Context, s *db.Store) {
	cur, err := s.Products.Aggregate(ctx, bson.A{
		bson.M{"$group": bson.M{"_id": "$category.id", "count": bson.M{"$sum": 1}}},
	})
	if err != nil {
		log.Fatalf("agg category counts: %v", err)
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var row struct {
			ID    primitive.ObjectID `bson:"_id"`
			Count int64              `bson:"count"`
		}
		_ = cur.Decode(&row)
		_, _ = s.Categories.UpdateByID(ctx, row.ID, bson.M{"$set": bson.M{"productCount": row.Count}})
	}
}

func recomputeProductRatings(ctx context.Context, s *db.Store) {
	cur, err := s.Reviews.Aggregate(ctx, bson.A{
		bson.M{"$group": bson.M{
			"_id":   bson.M{"p": "$product.id", "r": "$rating"},
			"count": bson.M{"$sum": 1},
		}},
	})
	if err != nil {
		log.Fatalf("agg ratings: %v", err)
	}
	defer cur.Close(ctx)

	type acc struct {
		dist  map[string]int64
		total int64
		sum   int64
	}
	byProduct := map[primitive.ObjectID]*acc{}
	for cur.Next(ctx) {
		var row struct {
			ID struct {
				P primitive.ObjectID `bson:"p"`
				R int                `bson:"r"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}
		_ = cur.Decode(&row)
		a := byProduct[row.ID.P]
		if a == nil {
			a = &acc{dist: map[string]int64{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0}}
			byProduct[row.ID.P] = a
		}
		a.dist[fmt.Sprintf("%d", row.ID.R)] = row.Count
		a.total += row.Count
		a.sum += int64(row.ID.R) * row.Count
	}
	for pid, a := range byProduct {
		avg := 0.0
		if a.total > 0 {
			avg = round1(float64(a.sum) / float64(a.total))
		}
		_, _ = s.Products.UpdateByID(ctx, pid, bson.M{"$set": bson.M{
			"ratingSummary.avg":          avg,
			"ratingSummary.count":        a.total,
			"ratingSummary.distribution": a.dist,
		}})
	}
}

func recomputeCustomerStats(ctx context.Context, s *db.Store) {
	cur, err := s.Orders.Aggregate(ctx, bson.A{
		bson.M{"$match": bson.M{"status": bson.M{"$ne": "cancelled"}}},
		bson.M{"$group": bson.M{
			"_id":        "$customer.id",
			"orderCount": bson.M{"$sum": 1},
			"totalSpent": bson.M{"$sum": "$pricing.total"},
		}},
	})
	if err != nil {
		log.Fatalf("agg customer stats: %v", err)
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var row struct {
			ID         primitive.ObjectID `bson:"_id"`
			OrderCount int64              `bson:"orderCount"`
			TotalSpent float64            `bson:"totalSpent"`
		}
		_ = cur.Decode(&row)
		_, _ = s.Customers.UpdateByID(ctx, row.ID, bson.M{"$set": bson.M{
			"stats.orderCount": row.OrderCount,
			"stats.totalSpent": round2(row.TotalSpent),
		}})
	}
}

// ---- small helpers -------------------------------------------------------

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func round1(f float64) float64 { return float64(int(f*10+0.5)) / 10 }
func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }
