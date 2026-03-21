# Building & Deploying a Cloud-Native E-Commerce Platform on AWS EKS — The Real Story

*A hands-on walkthrough of building a Go/Gin REST API with PostgreSQL, containerizing it, deploying to AWS EKS with Terraform, and wrestling with the real-world bugs you won't find in tutorials.*

---

## The Goal

Build a production-grade mini e-commerce platform from scratch:

- **Go/Gin** REST API with JWT authentication
- **PostgreSQL** on AWS RDS (private subnet, no public access)
- **EKS** cluster with managed node groups
- **CloudFront + S3** for frontend delivery
- **Terraform** for the entire infrastructure
- A clean, interactive frontend that actually works

Sounds straightforward. It wasn't.

---

## Architecture

```
User Browser
    │
    ▼
LoadBalancer (:80)
    │
    ▼
EKS Cluster (2 replicas)
├── Go/Gin API (:8080)
│   ├── POST /login       → JWT auth
│   ├── GET  /products    → Public
│   ├── POST /orders      → Authenticated
│   ├── GET  /orders      → Authenticated
│   └── GET  /            → Serves frontend
│
└── TCP/5432
    ▼
RDS PostgreSQL (private subnet)
├── users
├── products
├── categories
├── orders
└── order_items
```

The API serves both the frontend files and the JSON endpoints from the same container — no separate frontend deployment needed for the API path. S3 + CloudFront handles the static CDN version.

---

## Part 1: The API — Three Bugs That Broke Everything

The Go API used `sqlc` for type-safe SQL queries and `gin` for routing. The code looked clean. It compiled. The tests in the README said it worked. Then I actually ran it.

### Bug #1: `BindJSON` vs `ShouldBindJSON`

The login handler:

```go
if err := c.BindJSON(&req); err != nil {
    c.Status(http.StatusBadRequest)
    return
}
```

This returned `400 Bad Request` on valid JSON. Why?

In newer versions of Gin, `BindJSON()` **automatically writes a 400 response** if binding fails. Then my code wrote *another* 400 on top of it. The double-write corrupted the response. Worse — on some requests it worked fine, and on others it didn't, making it maddening to debug.

**The fix:** Switch to `ShouldBindJSON()`, which returns the error without auto-responding:

```go
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
    return
}
```

This is documented in Gin's README, but it's the kind of thing you only discover when your login endpoint randomly fails in production.

### Bug #2: The Password Hash That Didn't Match

The test seed data used this bcrypt hash:

```
$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

This is a widely-shared example hash from Stack Overflow. It hashes the string `"password"` — **not** `"password123!"` which is what the test credentials specified. Every login attempt returned 401.

I had to generate the correct hash inside a container (since the project requires Go 1.24 and the host only had Go 1.21):

```bash
docker run --rm golang:1.24-alpine sh -c '
  mkdir /tmp/h && cd /tmp/h &&
  go mod init genhash &&
  go get golang.org/x/crypto@v0.36.0 &&
  cat > main.go << "EOF"
package main
import (
  "fmt"
  "golang.org/x/crypto/bcrypt"
)
func main() {
  hash, _ := bcrypt.GenerateFromPassword(
    []byte("password123!"), bcrypt.DefaultCost)
  fmt.Println(string(hash))
}
EOF
  go run main.go'
```

**Lesson:** Never trust example hashes from the internet. Always generate your own.

### Bug #3: `int32` vs `int` — The Silent Zero

The auth middleware stored the user ID from the JWT:

```go
c.Set("user_id", int32(claims["sub"].(float64)))
```

But the handlers retrieved it with:

```go
UserID: int32(c.GetInt("user_id"))
```

`GetInt()` looks for a value of type `int`. The stored value was `int32`. Go's type system silently returned `0`. Every authenticated request used `user_id = 0`. Orders were created for a nonexistent user. The `/users/me` endpoint returned `{"user_id": 0}`.

**The fix:**

```go
uid, _ := c.Get("user_id")
UserID: uid.(int32)
```

Three bugs. All silent. All returning valid HTTP responses with wrong data. This is the kind of thing that passes code review and breaks in production.

---

## Part 2: Infrastructure with Terraform

The Terraform configuration provisions:

- **VPC** with public/private subnets across 2 AZs, NAT gateway
- **EKS** cluster (v1.29) with managed node group (2x `t3.medium`)
- **RDS** PostgreSQL (`db.t3.micro`) in private subnets
- **S3 bucket** with CloudFront distribution for the frontend
- **IRSA** role for pod-level AWS permissions
- **Security groups** restricting RDS access to EKS nodes only

```hcl
resource "aws_security_group" "rds" {
  name   = "rds-sg"
  vpc_id = module.vpc.vpc_id

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [module.eks.node_security_group_id]
  }
}
```

This is the critical piece — RDS only accepts connections from the EKS node security group. No public access. No bastion needed for normal operations.

### The Deploy

```bash
terraform apply -auto-approve -var="db_password=DemoPass123!"
```

66 resources. ~15 minutes. The EKS cluster alone takes 8-10 minutes (control plane provisioning), RDS another 5 minutes, CloudFront about 3 minutes.

The outputs give you everything you need:

```
eks_cluster_name     = "ecommerce-eks"
rds_endpoint         = "ecommerce-db.xxxxx.us-east-1.rds.amazonaws.com"
frontend_url         = "https://d1ktjbmehi6qzk.cloudfront.net"
frontend_bucket_name = "ecommerce-frontend-d4057dfc"
```

---

## Part 3: Container Build & ECR

The Dockerfile uses a multi-stage build:

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api ./cmd/api

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/api .
COPY frontend/ ./frontend/
EXPOSE 8080
CMD ["./api"]
```

Final image: ~15MB. The frontend files are baked into the container alongside the Go binary.

Push to ECR:

```bash
aws ecr create-repository --repository-name ecommerce-api
aws ecr get-login-password | docker login --username AWS --password-stdin <account>.dkr.ecr.us-east-1.amazonaws.com
docker tag eshop-api:latest <account>.dkr.ecr.us-east-1.amazonaws.com/ecommerce-api:latest
docker push <account>.dkr.ecr.us-east-1.amazonaws.com/ecommerce-api:latest
```

---

## Part 4: Database Initialization — The Private Subnet Problem

RDS is in a private subnet. You can't reach it from your laptop. This is correct for security, but it means you can't just `psql` into it to run the schema.

The solution: run the schema migration from *inside* the cluster:

```bash
# Load schema as a ConfigMap
kubectl create configmap db-schema \
  --from-file=schema=schema_postgres.sql

# Run a one-shot pod that mounts the schema and executes it
kubectl run db-init --rm -i --restart=Never \
  --image=postgres:15-alpine \
  --env="PGPASSWORD=DemoPass123!" \
  --overrides='{
    "spec": {
      "containers": [{
        "name": "db-init",
        "image": "postgres:15-alpine",
        "command": ["sh", "-c",
          "psql -h <RDS_ENDPOINT> -U demo -d demo -f /schema/schema"],
        "env": [{"name": "PGPASSWORD", "value": "DemoPass123!"}],
        "volumeMounts": [{"name": "schema", "mountPath": "/schema"}]
      }],
      "volumes": [{
        "name": "schema",
        "configMap": {"name": "db-schema"}
      }]
    }
  }'
```

The pod runs inside the VPC, connects to RDS over the private network, executes the SQL, and self-destructs. No SSH tunnels, no bastion hosts, no security group modifications.

---

## Part 5: Kubernetes Deployment

The deployment is straightforward — two replicas behind a LoadBalancer:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ecommerce-api
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ecommerce-api
  template:
    spec:
      containers:
        - name: api
          image: <account>.dkr.ecr.us-east-1.amazonaws.com/ecommerce-api:latest
          ports:
            - containerPort: 8080
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: api-secrets
                  key: database_url
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: api-secrets
                  key: jwt_secret
```

Secrets are created manually (in production, you'd use AWS Secrets Manager with the CSI driver):

```bash
kubectl create secret generic api-secrets \
  --from-literal=database_url="postgres://demo:DemoPass123!@<RDS_ENDPOINT>:5432/demo?sslmode=require" \
  --from-literal=jwt_secret="prod-jwt-secret-key-2026"
```

Deploy and verify:

```bash
kubectl apply -f k8s/
kubectl rollout status deployment/ecommerce-api
kubectl get svc ecommerce-api
```

The LoadBalancer provisions an AWS ELB with an external DNS name. Full end-to-end test:

```bash
LB="<load-balancer-dns>"

# Public endpoint
curl -s http://$LB/products | jq

# Authenticate
TOKEN=$(curl -s http://$LB/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123!"}' \
  | jq -r .access_token)

# Place an order
curl -s -X POST http://$LB/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"total": 29.99}' | jq
```

All endpoints return correct data. Orders persist in RDS across pod restarts.

---

## Part 6: The Frontend CSS Disaster

The frontend worked locally. It broke on EKS. The page loaded but looked completely unstyled — no layout, no colors, no grid. Just raw HTML.

The investigation:

1. **HTML loaded fine** — `curl` confirmed the correct HTML was served
2. **CSS returned 200** — the file existed and the content type was `text/css`
3. **But the browser showed no styling**

The root cause was **aggressive browser caching**. The browser cached an old (or empty) version of `styles.css` and refused to fetch the new one. The Go server was using Gin's `router.Static()` which sets long cache headers by default.

**Fix #1 — Cache-busting query strings:**

```html
<link rel="stylesheet" href="/static/styles.css?v=5">
<script src="/static/app.js?v=5"></script>
```

**Fix #2 — No-cache middleware on the Go server:**

```go
router.Use(func(c *gin.Context) {
    c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
    c.Header("Pragma", "no-cache")
    c.Header("Expires", "0")
    c.Next()
})
```

**Fix #3 (the trap) — Don't replace `router.Static()` with a custom handler:**

I initially tried:

```go
router.GET("/static/*filepath", func(c *gin.Context) {
    c.File("./frontend" + c.Param("filepath"))
})
```

This broke because `c.Param("filepath")` doesn't strip query parameters. A request for `/static/styles.css?v=5` tried to serve `./frontend/styles.css?v=5` — a file that doesn't exist. **404.** The CSS wasn't loading at all, and I was debugging layout issues in the CSS when the file wasn't even being served.

The working solution: keep `router.Static()` (which handles query strings correctly) and add the no-cache middleware globally.

---

## What I'd Do Differently

1. **Add health checks to the K8s deployment.** No readiness probe means the LoadBalancer can route to a pod that's still connecting to RDS.

2. **Use AWS Secrets Manager** instead of `kubectl create secret`. Secrets in etcd are base64-encoded, not encrypted.

3. **Add a `.dockerignore`** — the build context was 800MB because it included `.terraform/` modules. The build worked but the context transfer took 27 seconds.

4. **Pin the image tag.** Using `:latest` in production means `kubectl rollout restart` is your deploy mechanism. Use Git SHA tags instead.

5. **Add Terraform state backend (S3 + DynamoDB).** Local state works for a demo but is a disaster for teams.

---

## Final Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24 + Gin |
| Database | PostgreSQL 15 (RDS) |
| Auth | JWT (HS256, 15-min expiry) |
| ORM | sqlc (type-safe SQL) |
| Container | Multi-stage Docker (Alpine, ~15MB) |
| Registry | AWS ECR |
| Orchestration | AWS EKS (K8s 1.29) |
| Infrastructure | Terraform (VPC, EKS, RDS, S3, CloudFront, IRSA) |
| Frontend | Vanilla HTML/CSS/JS |
| CDN | CloudFront + S3 |

---

## Key Takeaways

- **`BindJSON` vs `ShouldBindJSON`** in Gin is a real production footgun. Always use `ShouldBindJSON` unless you want Gin to own the error response.
- **Go's type system doesn't warn you** when `GetInt()` returns `0` because the stored value is `int32`. This class of bug is silent and dangerous.
- **Browser caching will break your frontend deploys** unless you actively fight it. Cache-busting query strings + no-cache headers on the server.
- **Private RDS + init-from-pod** is the correct pattern. Don't expose your database to run migrations.
- **Test with `curl` before testing in the browser.** `curl` doesn't cache, doesn't have extensions, and shows you exactly what the server returns. Half the bugs in this project were found faster with `curl` than they would have been in a browser.

The final app: a fully deployed e-commerce platform with real product images, JWT authentication, order management, and proper cloud infrastructure — running on 2 EKS pods talking to RDS over private networking. Not bad for a day's work. The bugs made it educational.
