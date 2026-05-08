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

---

## Production CI/CD Pipeline

This phase hardens the project with a production-grade CI/CD system, secrets management, GitOps-style deployments, and automated testing. Everything described below is implemented and committed.

### Pipeline Overview

```
Pull Request opened
  │
  ├── Unit tests (go test -race)
  ├── Integration tests (real Postgres service container)
  ├── Docker build → Trivy vulnerability scan
  │     └── FAIL on CRITICAL/HIGH CVEs with available fix
  ├── Push image to ECR (SHA tag, immutable)
  └── Deploy to ephemeral namespace pr-{N} on EKS
        └── Post preview URL as PR comment

PR merged to main
  │
  ├── Build + push (same SHA tagging)
  ├── Deploy to staging namespace
  ├── Smoke test (curl /health/ready)
  ├── Manual approval gate (GitHub Environment protection rule)
  └── Deploy to production namespace

PR closed (merged or dismissed)
  └── kubectl delete namespace pr-{N}  ← ephemeral env destroyed
```

### GitHub Actions Workflows

| Workflow | Trigger | Purpose |
|---|---|---|
| `ci.yml` | Pull request → main | Test, scan, build, deploy ephemeral env |
| `cd.yml` | Push to main | Staging → approval → production |
| `cleanup.yml` | PR closed | Destroy ephemeral namespace |
| `tf-plan.yml` | PR with Infra/ changes | Run terraform plan, post output to PR |

### Authentication — GitHub OIDC (Zero Stored Keys)

CI authenticates to AWS using **OIDC federation** — no `AWS_ACCESS_KEY_ID` or `AWS_SECRET_ACCESS_KEY` stored anywhere in GitHub.

```
GitHub Actions workflow starts
  → GitHub generates a short-lived OIDC JWT for the run
  → aws-actions/configure-aws-credentials exchanges it with STS
  → STS validates JWT against the OIDC provider in your AWS account
  → Returns 15-minute temporary credentials scoped to github-actions-ecommerce IAM role
```

Defined in `Infra/github-oidc.tf`. The IAM role trust policy restricts assumption to your exact repo.

### Ephemeral Environments

Every PR gets a fully isolated Kubernetes namespace (`pr-{number}`) on the existing EKS cluster:

- Created when the PR opens, destroyed when it closes
- Never touches `staging` or `production` namespaces
- Uses Kustomize overlay to inject the SHA-tagged image and staging secrets
- Load balancer URL posted as a PR comment for manual testing

Namespaces are used instead of separate clusters because: one EKS cluster costs ~$72/month; a namespace is free, provides RBAC and NetworkPolicy isolation, and `kubectl delete namespace` tears down everything atomically.

---

## Secrets Management

Secrets flow through three layers with no plaintext stored in Git, environment variables, or CI secrets.

### Layer 1 — Terraform secrets (AWS Secrets Manager)

```hcl
# Infra/data.tf
data "aws_secretsmanager_secret_version" "db_creds" {
  secret_id = "ecommerce/${local.env}/rds"
}

resource "aws_db_instance" "postgres" {
  password = jsondecode(data.aws_secretsmanager_secret_version.db_creds.secret_string)["password"]
}
```

Bootstrap once per environment:
```bash
aws secretsmanager create-secret \
  --name "ecommerce/production/rds" \
  --secret-string '{"username":"demo","password":"<strong>","connection_string":"postgres://..."}'
```

### Layer 2 — CI pipeline (GitHub OIDC → IAM)

No secrets stored in GitHub. The OIDC token is ephemeral and scoped to a single workflow run.

### Layer 3 — Runtime (External Secrets Operator → K8s Secrets)

The `api-secrets` Kubernetes Secret referenced by pods is **never created manually**. External Secrets Operator (ESO) reads from Secrets Manager using IRSA and creates/rotates the Secret automatically.

```
ESO pod (IRSA) → Secrets Manager → creates K8s Secret → pod mounts it
Rotation: update secret in AWS → ESO picks up in 1h → zero pod restarts needed
```

Manifests: `k8s/base/external-secret.yaml`, `k8s/overlays/*/patch-secret-store.yaml`

---

## Helm — Multi-Environment K8s Manifests

```
helm/ecommerce-api/
  Chart.yaml
  values.yaml              ← defaults (ephemeral PR envs)
  values-staging.yaml      ← staging overrides
  values-production.yaml   ← production overrides (3 replicas)
  templates/
    _helpers.tpl           ← name/label helpers
    deployment.yaml
    service.yaml
    serviceaccount.yaml    ← IRSA annotation for Secrets Manager access
    external-secret.yaml   ← ESO SecretStore + ExternalSecret
```

CI deploys with a single command per environment:

```bash
# Ephemeral PR environment
helm upgrade --install ecommerce-api ./helm/ecommerce-api \
  --namespace pr-42 --create-namespace \
  --set image.tag=<git-sha> \
  --set secrets.env=staging \
  --wait --atomic --timeout 120s

# Production
helm upgrade --install ecommerce-api ./helm/ecommerce-api \
  --namespace production --create-namespace \
  --values helm/ecommerce-api/values-production.yaml \
  --set image.tag=<git-sha> \
  --wait --atomic --timeout 300s
```

`--atomic` rolls back the release automatically if any pod fails to become Ready within the timeout — no manual intervention needed on a bad deploy. `--wait` blocks the CI job until rollout is confirmed, replacing a separate `kubectl rollout status` step.

The `secrets.env` value controls which Secrets Manager path ESO reads from (`ecommerce/<env>/rds`, `ecommerce/<env>/jwt`), so the same chart serves all environments without duplication.

---

## Terraform Hardening

### Remote State — S3 + Native Locking

State is stored in S3 with workspace-scoped keys. No DynamoDB table required — Terraform 1.10 added `use_lockfile = true` which uses S3 conditional writes (`If-None-Match`) to create an atomic `.tflock` file.

```hcl
# Infra/backend.tf
terraform {
  backend "s3" {
    bucket       = "ecommerce-tfstate-776235864987"
    key          = "ecommerce/terraform.tfstate"
    region       = "us-east-1"
    encrypt      = true
    kms_key_id   = "alias/terraform-state"
    use_lockfile = true   # Terraform >= 1.10, replaces DynamoDB
  }
}
```

Workspace → state file mapping:
- `default` → `ecommerce/terraform.tfstate`
- `staging` → `env:/staging/ecommerce/terraform.tfstate`
- `pr-42` → `env:/pr-42/ecommerce/terraform.tfstate`

### Terraform Workspaces

```bash
terraform workspace new staging
terraform workspace select staging
terraform apply          # only touches staging state, never production
```

### New Terraform Files

| File | Purpose |
|---|---|
| `Infra/bootstrap/main.tf` | One-time S3 bucket + KMS key creation. Run before everything else. |
| `Infra/backend.tf` | Remote state config with `use_lockfile` |
| `Infra/locals.tf` | `is_production`, `is_ephemeral`, `common_tags` computed from workspace name |
| `Infra/data.tf` | Data sources: account ID, Secrets Manager credentials, ECR repo, EKS OIDC cert |
| `Infra/ecr.tf` | ECR repo with `IMMUTABLE` tags + lifecycle policy (keeps last 20 SHA images) |
| `Infra/github-oidc.tf` | OIDC provider + IAM role for GitHub Actions |

### RDS Production Configuration

```hcl
resource "aws_db_instance" "postgres" {
  multi_az                  = local.is_production   # synchronous standby in AZ-b
  deletion_protection       = local.is_production   # blocks terraform destroy
  skip_final_snapshot       = !local.is_production  # keeps snapshot before delete
  backup_retention_period   = local.is_production ? 7 : 1
  storage_encrypted         = true
  max_allocated_storage     = 100                   # autoscales, never runs out
  performance_insights_enabled = true
}
```

See `docs/rds-failover-k8s.md` for the full failover sequence, K8s probe behaviour during failover, Go retry logic, and RDS Proxy explanation.

---

## Automated Testing

### Unit Tests

Tests live in `internal/api/` and run against a **manual mock** of `db.Querier` — no database required.

```bash
go test ./internal/api/ -race -count=1 -v
```

| Suite | Tests | What is covered |
|---|---|---|
| `handlers_test.go` | 15 | Login (4 cases), products (4), orders (5), users/me (2) |
| `middleware_test.go` | 6 | No header, wrong scheme, malformed token, expired, wrong secret, valid |

**21 tests, all passing under `-race`.**

The mock (`mock_querier_test.go`) uses optional function fields — each test overrides only the DB method it needs, no boilerplate for unrelated methods.

`handlers.go` was updated to accept `db.Querier` (interface) instead of `*db.Queries` (concrete type) to enable mocking without touching sqlc-generated code.

### Integration Tests

Tagged with `//go:build integration` — excluded from normal runs.

```bash
# Requires a running Postgres
TEST_DATABASE_URL="postgres://testuser:testpass@localhost:5432/ecommerce_test?sslmode=disable" \
JWT_SECRET="test" \
go test -tags=integration ./internal/api/ -v
```

Each test:
1. Applies the schema (`CREATE TABLE IF NOT EXISTS`)
2. Truncates all tables (`RESTART IDENTITY CASCADE`)
3. Seeds test data via direct SQL
4. Makes real HTTP requests through the full router
5. Asserts real DB-round-tripped responses

In CI, GitHub Actions provides Postgres via `services: postgres:` — no external DB needed.

### Image Vulnerability Scanning (Trivy)

Runs automatically in `ci.yml` after the Docker build, before the push to ECR.

```yaml
- name: Scan image for vulnerabilities
  uses: aquasecurity/trivy-action@master
  with:
    exit-code: 1               # pipeline fails on CRITICAL or HIGH
    severity: CRITICAL,HIGH
    ignore-unfixed: true       # only fail on CVEs that have an available fix
```

The image **never reaches ECR** if it contains unfixed critical vulnerabilities.

### Full CI Gate Order

```
Unit tests ──┐
              ├──► Docker build → Trivy scan → push ECR → deploy pr-{N}
Integration ──┘
```

All four gates must pass before the image is pushed or deployed.

