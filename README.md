# Mini-e-commerce Project
Designed a complete e-commerce platform starting from DBML-based relational schema design through Dockerized PostgreSQL, and delivered a production-grade cloud deployment using Terraform, EKS, and RDS.

# Database Design – E-Commerce Schema (DBML → SQL)

This phase demonstrates a complete workflow for designing a relational database schema using **DBML**, exporting it to **PostgreSQL SQL**, and validating it inside a running **Postgres container via Docker**.

It includes:

- A full **e-commerce relational schema**
- The original **DBML source (`schema.dbml`)**
- **Generated PostgreSQL DDL (`schema_postgres.sql`) using dbdiagram.io**
- A runnable **Postgres environment using Docker**
- Example queries to confirm schema correctness

---

## 🧩 Schema Overview

### 1. users
Stores customer account data.

| Field         | Type          | Notes                     |
|---------------|---------------|---------------------------|
| id            | SERIAL PK     |                           |
| email         | varchar(255)  | unique, required          |
| password_hash | varchar(255)  | bcrypt hash, required     |
| full_name     | varchar(255)  | optional                  |
| created_at    | timestamp     | defaults to now()         |

---

### 2. products

| Field        | Type            | Notes               |
|--------------|-----------------|---------------------|
| id           | SERIAL PK       |                     |
| sku          | varchar(64)     | unique, required    |
| name         | varchar(255)    | required            |
| description  | text            | optional            |
| price        | decimal(10,2)   | required            |
| available    | boolean         | defaults to true    |
| created_at   | timestamp       | defaults to now()   |

---

### 3. categories

| Field | Type          | Notes            |
|-------|---------------|------------------|
| id    | SERIAL PK     |                  |
| name  | varchar(100)  | unique, required |

---

### 4. product_categories (junction table)

| Field        | Type      | Notes                                    |
|--------------|-----------|-------------------------------------------|
| product_id   | int FK    | references products(id)                   |
| category_id  | int FK    | references categories(id)                 |
| PRIMARY KEY (product_id, category_id) | Composite PK                 |

---

### 5. orders

| Field        | Type            | Notes                    |
|--------------|-----------------|---------------------------|
| id           | SERIAL PK       |                           |
| user_id      | int FK          | references users(id)      |
| status       | varchar(50)     | defaults to 'pending'     |
| total        | decimal(12,2)   |                           |
| created_at   | timestamp       | defaults to now()         |

---

### 6. order_items

| Field        | Type            | Notes                      |
|--------------|-----------------|----------------------------|
| id           | SERIAL PK       |                            |
| order_id     | int FK          | references orders(id)      |
| product_id   | int FK          | references products(id)    |
| quantity     | int             | defaults to 1              |
| unit_price   | decimal(10,2)   | required                   |

---

## 🛠 Running PostgreSQL Using Docker

This project uses Postgres running in Docker, mapped as:

```
8040 -> 5432  #more secure to use a different port than the default
```

### Start the database:

```bash
docker compose up -d
```

Verify:

```bash
docker ps
```

---

## 🗄 Apply the Schema

Install the PostgreSQL client (WSL):

```bash
sudo apt install postgresql-client -y
```

Load the schema:

```bash
psql "postgresql://demo:demo@localhost:8040/demo" -f schema_postgres.sql
```

---

## ✔️ Test the Schema

```bash
psql "postgresql://demo:demo@localhost:8040/demo"   -c "INSERT INTO users (email, full_name) VALUES ('tia@gimmy.com','Alice'); SELECT * FROM users;"
```

---

## 🔄 Resetting the Schema

```bash
psql "postgresql://demo:demo@localhost:8040/demo"   -c "DROP TABLE IF EXISTS order_items, orders, product_categories, products, categories, users CASCADE;"
```

Then reapply:

```bash
docker compose up -d
psql "postgresql://demo:demo@localhost:8040/demo" -f schema_postgres.sql
psql "postgresql://demo:demo@localhost:8040/demo" -c "INSERT INTO users (email, full_name) VALUES ('kia.gimmy@gmail.com','Alice'); SELECT * FROM users;"
[+] Running 1/0
 ✔ Container dbdesignproject-db-1  Running                                                                                           0.0s 
psql:schema_postgres.sql:6: ERROR:  relation "users" already exists
psql:schema_postgres.sql:16: ERROR:  relation "products" already exists
psql:schema_postgres.sql:21: ERROR:  relation "categories" already exists
psql:schema_postgres.sql:27: ERROR:  relation "product_categories" already exists
psql:schema_postgres.sql:35: ERROR:  relation "orders" already exists
psql:schema_postgres.sql:43: ERROR:  relation "order_items" already exists
ALTER TABLE
ALTER TABLE
ALTER TABLE
ALTER TABLE
ALTER TABLE
 id |        email        | full_name |         created_at         
----+---------------------+-----------+----------------------------
  1 | tia.gimmy@gmail.com | Alice     | 2025-12-12 20:53:09.73588
  2 | kia.gimmy@gmail.com | Alice     | 2025-12-12 20:58:55.843224
(2 rows)
```



---

## 📌 Purpose

This project demonstrates:

- DBML schema design  
- SQL DDL generation  
- Dockerized Postgres setup  
- Real schema validation

## 🔌 API Design, Build, and Testing

To validate that the database schema is usable in a real application context, a **minimal REST API** was implemented on top of the database.

The API is intentionally lightweight and exists to **prove the correctness of the schema, relationships, and access patterns**, not to provide a full application.

---

### 🧱 API Stack

- **Go (Gin)** – HTTP routing and middleware  
- **PostgreSQL** – database (Dockerized)  
- **sqlc** – type-safe Go code generated from SQL  
- **bcrypt** – password hashing  
- **JWT** – stateless authentication   
---

### 📁 API Structure

The API follows a schema-first design: the database schema defines the data model, sqlc generates access code, and the API orchestrates requests on top of it.

```text
cmd/api/
  main.go              # API bootstrap, CORS, frontend serving
internal/api/
  handlers.go          # HTTP endpoints (login, products, orders)
  middleware.go        # JWT authentication middleware
internal/db/
  queries.sql          # SQL queries (sqlc input)
  *.go                 # sqlc-generated code
frontend/
  index.html           # E-commerce UI (products, login, orders)
  app.js               # Client-side logic (fetch, auth, cart)
  styles.css           # Responsive layout and styling
```
---

## 🔐 Authentication

- Users authenticate via `POST /login`
- Passwords are stored as **bcrypt hashes** in the `users` table
- Successful login returns a **JWT**
- Protected routes require a valid  
  `Authorization: Bearer <token>` header

Authentication is enforced centrally via middleware to keep handlers simple and consistent.


## 📌 API Endpoints

| Endpoint          | Auth | Description                               |
|-------------------|------|-------------------------------------------|
| GET /             | No   | Serve frontend UI (index.html)             |
| GET /static/*     | No   | Serve CSS, JS, and static assets           |
| POST /login       | No   | Authenticate user and issue JWT            |
| GET /users/me     | Yes  | Return authenticated user identity         |
| GET /products     | No   | Public product listing                     |
| GET /products/:id | No   | Get product by ID                          |
| POST /orders      | Yes  | Create order for authenticated user        |
| GET /orders       | Yes  | List authenticated user’s orders           |

---

## 🧪 API Testing

The API was tested using both **curl** and the **browser-based frontend** against the running Dockerized PostgreSQL instance.

Validated scenarios include:

- Successful and failed authentication
- Access to protected routes with and without JWT
- Real database reads and writes
- Frontend rendering with live product data and order placement

Example commands used for testing:

```bash
# Login
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"tia.gimmy@gmail.com","password":"password123!"}'

# Access protected endpoint
curl http://localhost:8080/users/me \
  -H "Authorization: Bearer <JWT_TOKEN>"

# List products
curl http://localhost:8080/products

# Place an order (authenticated)
curl -X POST http://localhost:8080/orders \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"total": 29.99}'
```

**Server Access**

![server-access](docs/server-access.png)


**Login – Successful Authentication**

![successful-login](docs/successful-login.png)







## ✅ Result

This API layer validates the database design by exercising real application workflows, including authentication, authorization, and transactional data access.

The project demonstrates an end-to-end flow from:

**DBML schema → SQL DDL → Dockerized PostgreSQL → type-safe data access → authenticated API**

---

## ☁️ Infrastructure & Deployment Overview

This phase of the project includes a complete, production-oriented infrastructure to deploy the e-commerce application using AWS managed services and Kubernetes.


**Architecture Diagram**

![deployment architecture diagram](docs/deployment-architecture-diagram.jpeg)



The infrastructure is provisioned using **Terraform** and consists of:

### Core Components

- **VPC**
  - Public subnets for internet-facing components
  - Private subnets for EKS worker nodes & RDS

- **Amazon EKS**
  - Managed Kubernetes control plane
  - Managed node groups running in private subnets
  - Public and private API endpoint access enabled
    

- **Amazon RDS (PostgreSQL)**
  - Private database instance
  - Accessible only from EKS node security group
  - Used by the Go API as the primary datastore

- **S3 + CloudFront (Frontend)**
  - Static frontend hosted in S3
  - CloudFront used for global CDN, TLS, and security
  - Frontend communicates with the API via HTTPS

This separation enforces:
- Network isolation
- Least-privilege access
- Clear boundaries between frontend, API, and database

## ✅ Result

![eks-deployment-outputs](docs/eks-deployment-outputs.png)

```
kubectl get nodes


NAME                         STATUS   ROLES    AGE   VERSION
ip-10-0-1-159.ec2.internal   Ready    <none>   20m   v1.29.15-eks-ecaa3a6
ip-10-0-2-69.ec2.internal    Ready    <none>   20m   v1.29.15-eks-ecaa3a6
```

![successful-frontend-deployment](docs/successful-frontend-deployment.png)

---

## 🚀 API Deployment on EKS

The Go API is deployed as a containerized service running inside the EKS cluster.

### Deployment Flow

1. **Go API is containerized**
   - Multi-stage Docker build
   - Minimal runtime image
   - Exposes port `8080`

2. **Image pushed to Amazon ECR**
   - Used as the image source for Kubernetes pods

3. **Kubernetes Deployment**
   - Runs multiple API replicas for availability
   - Environment variables injected via Kubernetes Secrets
   - Database connection string points to the RDS endpoint

4. **ALB**
   - Application Load Balancer exposes the API publicly
   - Frontend communicates with the API via the ALB DNS name

5. **CORS enabled**
   - Allows browser-based frontend to access the API securely

This setup enables horizontal scaling, rolling updates, and safe separation of concerns.

---

## 🔧 API Code Changes for Deployment

To support deployment on EKS and integration with the frontend, several targeted changes were made to the API codebase — including three bug fixes discovered during real testing.

### 1️⃣ Centralized Server Bootstrap (`cmd/api/main.go`)

- Database connection is created once and injected into handlers
- Environment variables are used for:
  - `DATABASE_URL`
  - `JWT_SECRET`
- Global middleware is registered at startup (CORS, no-cache headers)
- Frontend is served from the same container

```go
router := gin.Default()
router.Use(corsMiddleware)
router.Use(noCacheMiddleware)
api.RegisterRoutes(router, queries)
router.Static("/static", "./frontend")
router.GET("/", func(c *gin.Context) { c.File("./frontend/index.html") })
router.Run(":8080")
```

### 2️⃣ CORS & Cache-Control Middleware

CORS support allows the browser-hosted frontend to call the API. No-cache middleware was added to prevent browser caching issues with static assets during deployment.

### 3️⃣ Bug Fix: `BindJSON` → `ShouldBindJSON`

Gin's `BindJSON()` auto-writes a 400 response on binding failure. The handler then wrote a second 400, corrupting the response. Switching to `ShouldBindJSON()` returns the error without auto-responding.

```go
// Before (broken — double 400 response)
if err := c.BindJSON(&req); err != nil { ... }

// After (correct)
if err := c.ShouldBindJSON(&req); err != nil { ... }
```

### 4️⃣ Bug Fix: `int32` vs `int` Type Assertion

The auth middleware stored `user_id` as `int32`, but handlers retrieved it with `GetInt()` which expects `int`. Go silently returned `0`, causing all authenticated requests to use `user_id = 0`.

```go
// Before (broken — silently returns 0)
UserID: int32(c.GetInt("user_id"))

// After (correct)
uid, _ := c.Get("user_id")
UserID: uid.(int32)
```

### 5️⃣ Bug Fix: Incorrect Password Hash

The seed data used a widely-shared example bcrypt hash that hashed `"password"`, not the actual test password `"password123!"`. The correct hash was generated and replaced.

### 6️⃣ Schema-First Database Access

Database schema defines the data model, sqlc generates type-safe Go code, and handlers use generated queries.

---

## 🤖 Claude Code Agents

This project was built, tested, debugged, and deployed with the help of **Claude Code** and a set of custom reusable agents designed to automate review and operational tasks.

### Custom Agents Used

| Agent | Purpose | Tools |
|-------|---------|-------|
| `review-infra` | Reviews Terraform code — VPC, EKS, RDS, security groups, tagging, cost optimization | Read, Glob, Grep, Bash |
| `review-k8s` | Reviews Kubernetes manifests — deployments, services, security contexts, resource limits, scaling | Read, Glob, Grep, Bash |
| `review-cicd` | Reviews CI/CD pipelines and security posture — GitHub Actions, ArgoCD, image scanning, secrets management | Read, Glob, Grep, Bash |
| `git-push` | Handles version control — compares local changes against remote, creates timestamped backup branches, stages, commits, and pushes safely | Read, Glob, Grep, Bash |

### How They Work

Agents are defined as Markdown files in `~/.claude/agents/` and are invoked via `@agent-name` in Claude Code. Each agent:

- Has a **specific role** (e.g., infra reviewer, K8s reviewer, version control)
- Has **scoped tool access** — only the tools it needs
- Follows a **structured review format** with severity levels (CRITICAL / HIGH / MEDIUM / LOW)
- Runs on a specified model (e.g., Sonnet for speed, Opus for depth)

