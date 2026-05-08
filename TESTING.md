# Testing Steps

## 1. Local Setup

```bash
# Start PostgreSQL
docker-compose up -d

# Wait for DB to be ready
sleep 3

# Create tables
PGPASSWORD=demo psql -h localhost -p 8040 -U demo -d demo -f schema_postgres.sql

# Seed test data
PGPASSWORD=demo psql -h localhost -p 8040 -U demo -d demo <<'SQL'
-- Test user (password: password123!)
INSERT INTO users (email, full_name, password_hash) VALUES
  ('test@example.com', 'Test User', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy');

-- Test products
INSERT INTO products (sku, name, description, price) VALUES
  ('SKU-001', 'Wireless Mouse',   'Ergonomic wireless mouse', 29.99),
  ('SKU-002', 'Mechanical Keyboard', '60% layout, blue switches', 79.99),
  ('SKU-003', 'USB-C Hub',        '7-in-1 adapter', 49.99);

-- Categories
INSERT INTO categories (name) VALUES ('Electronics'), ('Accessories');
INSERT INTO product_categories VALUES (1, 2), (2, 1), (3, 2);
SQL

# Run the API
DATABASE_URL="postgres://demo:demo@localhost:8040/demo?sslmode=disable" \
JWT_SECRET="test-secret-key" \
go run ./cmd/api
```

Open http://localhost:8080 in your browser.

## 2. Test the UX Flow

### Step 1 — Browse Products (No Login Required)
- Open http://localhost:8080
- You should see 3 product cards loaded from the database
- No "Order" buttons visible (not logged in)

### Step 2 — Login
- Enter email: `test@example.com`
- Enter password: `password123!`
- Click Login
- Login form disappears, "My Orders" section appears
- "Order" buttons now visible on product cards

### Step 3 — Place an Order
- Click "Order" on any product
- Order appears in "My Orders" section with status "pending"

### Step 4 — Place Multiple Orders
- Order different products
- Each order shows with its total and status

### Step 5 — Session Persistence
- Refresh the page
- You should still be logged in (JWT stored in localStorage)
- Products and orders reload automatically

### Step 6 — Logout
- Click "Logout" in the header
- "Order" buttons disappear
- "My Orders" section hides
- Login form reappears

### Step 7 — Bad Login
- Enter wrong email/password
- Error message "Invalid email or password" appears

## 3. API Tests (curl)

```bash
API=http://localhost:8080

# List products (public)
curl -s $API/products | jq

# Login
TOKEN=$(curl -s $API/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123!"}' | jq -r .access_token)

echo "Token: $TOKEN"

# Get current user
curl -s $API/users/me -H "Authorization: Bearer $TOKEN" | jq

# Create an order
curl -s -X POST $API/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"total": 29.99}' | jq

# List orders
curl -s $API/orders -H "Authorization: Bearer $TOKEN" | jq

# Unauthorized request (no token)
curl -s -w "\nHTTP %{http_code}\n" $API/orders
```

## 4. Docker Build Test

```bash
docker build -t eshop-api .
docker run --rm -p 8080:8080 \
  -e DATABASE_URL="postgres://demo:demo@host.docker.internal:8040/demo?sslmode=disable" \
  -e JWT_SECRET="test-secret-key" \
  eshop-api
```

## 5. Kubernetes Deployment Test

```bash
# Create secret
kubectl create secret generic api-secrets \
  --from-literal=database_url="postgres://USER:PASS@RDS_ENDPOINT:5432/eshop?sslmode=require" \
  --from-literal=jwt_secret="your-production-secret"

# Deploy
kubectl apply -f k8s/

# Verify
kubectl get pods
kubectl get svc ecommerce-api

# Access via LoadBalancer external IP
curl http://<EXTERNAL_IP>/products
```

## 6. Cleanup

```bash
docker-compose down -v
```
