# RDS Failover and Kubernetes Interaction

## 1. Multi-AZ Architecture

```
AZ us-east-1a                    AZ us-east-1b
┌──────────────────┐             ┌──────────────────┐
│  PRIMARY         │  sync repl  │  STANDBY         │
│  ecommerce-db    │ ──────────► │  ecommerce-db    │
│  (reads+writes)  │             │  (no traffic)    │
└──────────────────┘             └──────────────────┘
         ▲
         │ DNS CNAME
         │ ecommerce-db.xxxx.us-east-1.rds.amazonaws.com
         │
  ┌──────┴───────┐
  │  EKS Pods    │  Go app → DATABASE_URL → CNAME
  │  (your API)  │
  └──────────────┘
```

With `multi_az = true`, AWS maintains a **synchronous standby replica** in a second AZ.
Every write committed to the primary is acknowledged only after it has also been written
to the standby. This means **zero data loss (RPO = 0)** on failover.

The standby receives **no application traffic** under normal operation. It exists
solely to take over instantly if the primary fails.

---

## 2. What Triggers a Failover

AWS initiates failover automatically when it detects any of:

| Trigger | Typical RTO |
|---|---|
| Primary instance crashes / kernel panic | 30–60 s |
| AZ network outage | 60–120 s |
| Storage failure on primary | 30–60 s |
| OS-level patching (scheduled maintenance) | 30–60 s |
| Manual reboot with failover flag | 30–60 s |
| `aws rds failover-db-instance` API call (testing) | 30–60 s |

**RPO = 0** because replication is synchronous — standby always has every committed transaction.
**RTO = 30–120 s** depending on detection latency and DNS propagation.

---

## 3. The Failover Sequence (Step by Step)

```
t=0s   AWS health check detects primary is unreachable (3 consecutive failures)
t=10s  RDS service decides to failover
t=10s  Standby is promoted to primary (it already has all data — no catchup needed)
t=12s  New primary becomes writable
t=12s  AWS updates the DNS CNAME for the endpoint to point to the new primary
t=17s  DNS TTL expires (RDS uses ~5 second TTL for its CNAMEs)
t=17s  Your application's DNS resolver returns the new IP
t=17s  New connections succeed; existing connections must reconnect
t=90s  AWS starts provisioning a new standby in the original AZ (background)
```

**What your application actually sees**: from t=0 to ~t=17, any connection attempt fails.
Existing long-lived connections get a TCP RST at t=10 when the old primary dies.

---

## 4. The DNS-Based Failover Mechanism

RDS exposes a single **stable CNAME** endpoint:

```
ecommerce-db.xxxxxxxxxxxxxxxx.us-east-1.rds.amazonaws.com
```

This CNAME always points to the **current primary**. During failover, AWS atomically
updates this CNAME. Your application code never changes — the same connection string
works before, during, and after failover.

```
Before failover:
  CNAME → 10.0.1.45 (AZ-a primary)

After failover:
  CNAME → 10.0.2.83 (AZ-b, now primary)
```

**Critical**: your application must NOT cache IP addresses. The `lib/pq` driver in Go
does not cache IPs by default — it resolves the CNAME on each new connection, so
this is handled correctly.

---

## 5. Kubernetes Interaction During Failover

### 5.1 What happens to your pods

Your pods maintain a connection pool to RDS. During the ~17 second failover window:

```
t=0   AWS kills primary
t=0   All existing DB connections in your Go pool get TCP RST / EOF
t=1   lib/pq returns error on any query: "connection reset by peer"
t=1   Your API handlers start returning 500s (if no retry logic)
t=5   Readiness probe hits /health endpoint, which fails DB ping
t=15  After 3 failed readiness probes (3 × periodSeconds=10), pod marked NotReady
t=15  Pod removed from Service endpoints — load balancer stops sending traffic to it
t=17  DNS resolves to new primary IP
t=18  Connection pool reconnects successfully
t=18  Next readiness probe succeeds — pod re-added to endpoints
```

### 5.2 Why readiness ≠ liveness during failover

**Do NOT use the same DB-ping probe for both readiness and liveness.**

- **Readiness probe failing**: pod is temporarily removed from endpoints. Traffic stops. 
  Correct — the pod can't serve requests right now but will recover.
- **Liveness probe failing**: pod is **killed and restarted**. This adds 30–60 s of
  container restart time ON TOP of the already-recovering failover. Avoid.

Your current probes hit `/` (HTTP 200), not the DB. This is intentionally safe — 
the API returns 200 on the root route regardless of DB state. For production, add
a dedicated `/health` endpoint:

```go
// Readiness: checks DB connectivity
router.GET("/health/ready", func(c *gin.Context) {
    if err := db.PingContext(c.Request.Context()); err != nil {
        c.JSON(503, gin.H{"status": "not ready", "error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"status": "ready"})
})

// Liveness: only checks that the process is alive, never the DB
router.GET("/health/live", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "alive"})
})
```

Updated K8s probes:
```yaml
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  failureThreshold: 3      # 30s before removed from endpoints — covers failover window

livenessProbe:
  httpGet:
    path: /health/live     # Never DB-dependent
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 15
  failureThreshold: 5      # Only kill after 75s of total unresponsiveness
```

### 5.3 Connection retry in Go

Without retry logic, every in-flight request during failover returns a 500. With
retry + exponential backoff, most requests are transparent to clients.

```go
// internal/db/db.go
package db

import (
    "database/sql"
    "fmt"
    "time"

    _ "github.com/lib/pq"
)

func Open(dsn string) (*sql.DB, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }

    // Connection pool tuning for EKS — important for RDS Proxy compatibility
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)  // Force reconnect before RDS reaps idle conns
    db.SetConnMaxIdleTime(1 * time.Minute)

    // Retry the initial Ping so the app doesn't crash at startup if RDS is still warming
    if err := pingWithRetry(db, 5, 2*time.Second); err != nil {
        return nil, fmt.Errorf("db: failed to connect after retries: %w", err)
    }

    return db, nil
}

func pingWithRetry(db *sql.DB, attempts int, delay time.Duration) error {
    var err error
    for i := 0; i < attempts; i++ {
        if err = db.Ping(); err == nil {
            return nil
        }
        time.Sleep(delay * time.Duration(1<<i)) // exponential: 2s, 4s, 8s, 16s, 32s
    }
    return err
}
```

For request-level retries (handling mid-failover errors):
```go
// Wrap DB calls that are idempotent (reads, or writes with unique constraints)
func withRetry(ctx context.Context, fn func() error) error {
    backoff := 100 * time.Millisecond
    for attempt := 0; attempt < 3; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }
        // Only retry on connection errors, not constraint violations
        if isRetryableDBError(err) && attempt < 2 {
            select {
            case <-time.After(backoff):
                backoff *= 2
            case <-ctx.Done():
                return ctx.Err()
            }
            continue
        }
        return err
    }
    return nil
}

func isRetryableDBError(err error) bool {
    if err == nil {
        return false
    }
    msg := err.Error()
    return strings.Contains(msg, "connection reset") ||
        strings.Contains(msg, "broken pipe") ||
        strings.Contains(msg, "connection refused") ||
        strings.Contains(msg, "EOF")
}
```

---

## 6. RDS Proxy: Making Failover Transparent

RDS Proxy sits between your EKS pods and RDS:

```
EKS Pods ──► RDS Proxy (stays stable) ──► RDS Primary
                    │
                    └──► On failover: holds connections,
                         retries against new primary,
                         releases connections when ready
```

Without Proxy, your app sees 17–60 s of errors.
With Proxy, your app sees **1–2 s of query delay** — the proxy holds in-flight queries
and replays them against the new primary once it's available.

Additional benefits for EKS:
- **Connection pooling**: 25 pods × 25 connections = 625 raw connections to RDS. 
  RDS `db.t3.micro` max connections = 87. Proxy multiplexes all pods onto ~10 real connections.
- **IAM auth**: Proxy enforces IAM authentication — no password needed in your connection string.

```
# With RDS Proxy, DATABASE_URL becomes:
DATABASE_URL=postgres://demo@ecommerce-proxy.proxy-xxx.us-east-1.rds.amazonaws.com:5432/demo?sslmode=require
# No password — IAM token injected by the Proxy using IRSA on your service account
```

---

## 7. Testing Failover

### 7.1 Manual failover test
```bash
# Trigger a failover in ~30 seconds
aws rds failover-db-instance \
  --db-instance-identifier ecommerce-db \
  --region us-east-1

# Watch the endpoint flip
watch -n2 "dig +short $(aws rds describe-db-instances \
  --db-instance-identifier ecommerce-db \
  --query 'DBInstances[0].Endpoint.Address' \
  --output text)"
```

### 7.2 What to observe during the test
```bash
# Watch pod readiness in real time
kubectl get pods -n production -w

# Stream pod logs to see DB error messages
kubectl logs -n production -l app=ecommerce-api -f --max-log-requests=5

# Watch endpoints — pods should drop out then rejoin
kubectl get endpoints ecommerce-api -n production -w
```

### 7.3 Acceptance criteria
- [ ] Failover completes within 120 s (AWS SLA is 2 minutes)
- [ ] Pods are removed from endpoints during outage (readiness probe fires)
- [ ] Pods re-join endpoints within 30 s of RDS recovery
- [ ] No data was lost (check transaction logs)
- [ ] Zero pod restarts (liveness probe never fired)
- [ ] Incoming requests received 503 (not 500) during the outage — 
      K8s removed unhealthy pods from load balancer, so clients got clean errors

---

## 8. Terraform Configuration for Production RDS

The key fields that enable failover in your `aws_db_instance`:

```hcl
resource "aws_db_instance" "postgres" {
  # --- failover-critical settings ---
  multi_az                = local.is_production   # synchronous standby in second AZ
  deletion_protection     = local.is_production   # prevents accidental terraform destroy
  skip_final_snapshot     = !local.is_production  # keep final snapshot in production
  backup_retention_period = local.is_production ? 7 : 1  # 7 days PITR in production

  # --- performance and reliability ---
  storage_encrypted           = true
  storage_type                = "gp3"
  max_allocated_storage       = 100       # autoscale up to 100GB, never runs out
  performance_insights_enabled = true
  enabled_cloudwatch_logs_exports = ["postgresql"]

  # --- connection pool tuning ---
  # db.t3.small supports ~97 max connections
  # Use RDS Proxy in production to multiplex EKS pods onto fewer real connections
}
```

> **The `multi_az` line is the only required change** to go from zero-failover to
> production-grade HA. Everything else is defense-in-depth.
