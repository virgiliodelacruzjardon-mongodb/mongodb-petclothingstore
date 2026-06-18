# 🐾 PawCouture — Pet Clothing Store (CRUD)

A full-stack CRUD demo for an online pet-clothing store, fully containerized with **Podman**.

| Layer    | Tech                                          |
|----------|-----------------------------------------------|
| Frontend | Next.js 14 (App Router) + TypeScript + Tailwind, JWT auth |
| Backend  | Go 1.23 + chi router + official MongoDB driver |
| Database | MongoDB **8.0**                                |
| Runtime  | Podman (mongo + backend + frontend + one-shot seeder) |

The database is seeded with **200+ documents in every collection**.

---

## 1. Quick start (everything in Podman)

You need Podman 4.4+ (`podman compose`) or the `podman-compose` Python wrapper.

```bash
cd pet-store

# Option A — native compose (Podman 4.4+)
podman compose up --build

# Option B — podman-compose wrapper
podman-compose up --build
```

This builds and starts four things:

1. `petstore-mongo`   → MongoDB 8 on `localhost:27017`
2. `petstore-backend` → Go API on `http://localhost:8080`
3. `petstore-seeder`  → runs once, drops & seeds the DB, then exits
4. `petstore-frontend`→ Next.js on `http://localhost:3000`

Two surfaces:
- **Storefront** (customers): http://localhost:3000/shop → add to cart → http://localhost:3000/cart → checkout.
- **Admin CRUD**: http://localhost:3000 (Dashboard, Products, Categories, Customers, Orders, Reviews).

Checkout requires being logged in; the logged-in user is used as the order's
customer and the order is created through the transactional `POST /orders`
endpoint (atomic stock decrement). Log in with the seeded admin:

```
email:    admin@petstore.dev
password: admin123
```

> The seeder and backend both retry the Mongo connection on startup, so start
> ordering is handled automatically — no healthcheck plugin required.

### Re-seed at any time

```bash
podman compose run --rm seeder
# or
podman-compose run --rm seeder
```

---

## 2. Collections & schema design

Per the requested approach, the model **maximizes embedding** (denormalization)
so the hot read paths need no `$lookup`, and uses the **computed pattern** to
keep pre-aggregated totals on the parent documents.

### `categories`
```jsonc
{
  "_id": ObjectId,
  "name": "Dog Sweaters",
  "slug": "dog-sweaters",
  "description": "...",
  "icon": "paw",
  "productCount": 14,      // COMPUTED PATTERN (recalculated from products)
  "createdAt": ISODate, "updatedAt": ISODate
}
```

### `products`  (the most denormalized document)
```jsonc
{
  "_id": ObjectId,
  "name": "PawCouture fleece Hoodie",
  "slug": "...",
  "brand": "PawCouture",
  "petType": "dog",
  "category": {            // EXTENDED REFERENCE (embedded snapshot of category)
    "id": ObjectId, "name": "Dog Hoodies", "slug": "dog-hoodies"
  },
  "basePrice": 29.99, "currency": "USD",
  "variants": [            // SUBSET / EMBEDDED ARRAY (size+color+stock+price)
    { "sku": "PAW-M-BLK-3", "size": "M", "color": "Black", "price": 31.5, "stock": 42 }
  ],
  "images": ["https://..."],
  "tags": ["dog", "hoodie", "fleece", "warm"],
  "totalStock": 118,       // COMPUTED PATTERN (sum of variant stock)
  "ratingSummary": {       // COMPUTED PATTERN (from reviews collection)
    "avg": 4.3, "count": 27,
    "distribution": { "1": 1, "2": 0, "3": 3, "4": 8, "5": 15 }
  },
  "active": true,
  "createdAt": ISODate, "updatedAt": ISODate
}
```

### `customers`
```jsonc
{
  "_id": ObjectId,
  "firstName": "...", "lastName": "...", "email": "...", "phone": "...",
  "passwordHash": "(bcrypt, never returned by the API)",
  "role": "customer | admin",
  "addresses": [           // EMBEDDED ARRAY
    { "label": "Home", "line1": "...", "city": "...", "state": "...",
      "zip": "...", "country": "US", "default": true }
  ],
  "stats": {               // COMPUTED PATTERN (from orders, excludes cancelled)
    "orderCount": 5, "totalSpent": 412.30
  }
}
```

### `orders`
```jsonc
{
  "_id": ObjectId,
  "orderNumber": "PET-000123",
  "customer": {            // EXTENDED REFERENCE (snapshot)
    "id": ObjectId, "name": "Jane Doe", "email": "..."
  },
  "items": [               // EMBEDDED line items = immutable product snapshots
    { "productId": ObjectId, "name": "...", "sku": "...", "size": "M",
      "color": "Black", "price": 31.5, "qty": 2,
      "lineTotal": 63.0 }  // COMPUTED PATTERN
  ],
  "pricing": {             // COMPUTED PATTERN (computed server-side on write)
    "subtotal": 63.0, "tax": 5.04, "shipping": 6.99,
    "discount": 0, "total": 75.03
  },
  "status": "pending|paid|shipped|delivered|cancelled",
  "shippingAddress": { ... },   // EMBEDDED snapshot
  "placedAt": ISODate
}
```

### `reviews`
```jsonc
{
  "_id": ObjectId,
  "product":  { "id": ObjectId, "name": "...", "slug": "...", "image": "..." }, // snapshot
  "customer": { "id": ObjectId, "name": "...", "email": "..." },                // snapshot
  "rating": 5, "title": "...", "body": "...",
  "verified": true, "createdAt": ISODate
}
```

### Why these choices
- **Embed what you read together.** A product page shows variants, category name,
  image and rating in one read — all embedded, zero joins.
- **Extended reference, not full reference.** Orders/reviews store only the few
  fields they display (name, email, sku) instead of the whole related doc.
- **Snapshots for history.** Order items copy the price/sku at purchase time, so
  later product edits never rewrite past orders.
- **Computed pattern.** `productCount`, `totalStock`, `ratingSummary`,
  `customer.stats` and order `pricing` are pre-calculated on write so reads are
  O(1). The API recomputes them whenever the underlying data changes
  (`backend/internal/api/recompute.go`), and the seeder recomputes them in bulk.

---

## 2b. Multi-document transactions (atomic stock decrement)

Creating an order via `POST /api/orders` runs inside a **MongoDB multi-document
transaction** (`backend/internal/api/orders.go`). In a single atomic unit it:

1. For each line item, decrements the matching variant's `stock` **only if**
   `stock >= qty`, and decrements the product's `totalStock`.
2. Inserts the order document.

If **any** item lacks stock, the transaction aborts and every decrement already
applied is rolled back automatically — the order is never created and inventory
is left untouched. The API returns `409 Conflict` with
`insufficient stock for sku ...`.

The conditional decrement is race-safe even under concurrency:

```js
// match product + the specific variant ONLY when it has enough stock
db.products.updateOne(
  { _id: productId, variants: { $elemMatch: { sku: sku, stock: { $gte: qty } } } },
  { $inc: { "variants.$[v].stock": -qty, totalStock: -qty } },
  { arrayFilters: [ { "v.sku": sku, "v.stock": { $gte: qty } } ] }
)
// ModifiedCount === 0  =>  not enough stock  =>  abort the transaction
```

Transaction options use `readConcern: snapshot` and `writeConcern: majority`.
`WithTransaction` automatically retries on transient/commit errors but **not**
on the application-level "insufficient stock" error.

> **Why the replica set?** MongoDB transactions require a replica set (or
> sharded cluster); they do not work on a standalone `mongod`. That is why the
> `mongo` service runs with `--replSet rs0` and a one-shot `mongo-init` service
> initiates it, and why `MONGODB_URI` includes `?replicaSet=rs0`.

Try it:
```bash
TOKEN=$(curl -s localhost:8080/api/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@petstore.dev","password":"admin123"}' | jq -r .token)

# grab a product id + a variant sku, then order an absurd quantity:
curl -i localhost:8080/api/orders -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"customer":{"id":"<id>","name":"X","email":"x@x.com"},
       "items":[{"productId":"<pid>","sku":"<sku>","price":10,"qty":99999}]}'
# -> HTTP/1.1 409 Conflict  {"error":"insufficient stock for sku <sku>"}
```

### Indexes (`backend/internal/db/indexes.go`)
- `categories`: unique `slug`
- `products`: unique `slug`, `category.id`, `petType`, `tags`, `ratingSummary.avg`, **text** index on `name`+`description`
- `customers`: unique `email`
- `orders`: unique `orderNumber`, `customer.id`, `status`, `placedAt`
- `reviews`: `product.id`, `customer.id`, `rating`

---

## 3. REST API

Base URL: `http://localhost:8080/api`

Public (read): `GET /products`, `/products/{id}`, `/categories`, `/categories/{id}`, `/reviews`, `/reviews/{id}`
Auth: `POST /auth/register`, `POST /auth/login`, `GET /auth/me`
Protected (Bearer JWT) — full CRUD: `products`, `categories`, `customers`, `orders`, `reviews`

All list endpoints accept `?page=`, `?limit=`, and resource-specific filters
(`q`, `petType`, `categoryId`, `status`, `customerId`, `productId`).

Example:
```bash
# login
TOKEN=$(curl -s localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@petstore.dev","password":"admin123"}' | jq -r .token)

# create a category
curl -s localhost:8080/api/categories \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Cat Costumes","description":"spooky season"}'
```

---

## 4. Project layout

```
pet-store/
├── podman-compose.yml
├── .env.example
├── backend/                 # Go API
│   ├── Dockerfile
│   ├── cmd/api/main.go       # HTTP server
│   ├── cmd/seed/main.go      # seeds 200+ docs/collection + recomputes + indexes
│   └── internal/
│       ├── models/           # BSON-tagged structs (schema)
│       ├── db/               # connection + indexes
│       └── api/              # router, auth(JWT), CRUD handlers, computed pattern
└── frontend/                # Next.js (App Router)
    ├── Dockerfile
    ├── lib/ (api client, types, auth context)
    ├── components/ (Navbar, Modal, Guard)
    └── app/
        ├── shop/      # customer storefront (browse + add to cart)
        ├── cart/      # cart review + checkout (creates an order via POST /orders)
        └── (admin) dashboard, login, register, products, categories, customers, orders, reviews
```

---

## 5. Local dev (without containers, optional)

```bash
# backend
cd backend && go mod tidy && go run ./cmd/seed && go run ./cmd/api
# frontend (new terminal)
cd frontend && npm install && NEXT_PUBLIC_API_URL=http://localhost:8080/api npm run dev
```

Requires a local MongoDB 8 **running as a replica set** (transactions need it).
Quick one-node replica set with Podman:

```bash
podman run -d --name mongo -p 27017:27017 docker.io/library/mongo:8.0 \
  mongod --replSet rs0 --bind_ip_all
podman exec mongo mongosh --quiet --eval \
  'rs.initiate({_id:"rs0",members:[{_id:0,host:"localhost:27017"}]})'

export MONGODB_URI="mongodb://localhost:27017/?replicaSet=rs0"
```
