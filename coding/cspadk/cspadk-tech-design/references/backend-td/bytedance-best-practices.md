# ByteDance Internal Best Practices for Backend TDs

Load this file when writing **Data Model Changes**, **IDL / API Changes**, **Feature Design** modules, or **Technology Selection** sections. These are anti-patterns that repeatedly burn review cycles — bake them into the design from the start.

---

## MySQL / RDS

### Do NOT use UNIQUE KEY constraints on synced tables

**Rule:** On any MySQL table that is (or may become) part of a cross-region sync / DTS / binlog replication flow, do NOT declare `UNIQUE KEY` / `UNIQUE INDEX` at the schema level.

**Why:**
- Cross-region DB sync (DES-RDS, binlog-based replication) can re-apply rows in an order that hits the unique constraint and halts replication on a duplicate-key error. Recovery is manual, region by region.
- Some internal RDS platforms reject DDL that introduces unique constraints on already-synced tables.
- Legacy soft-delete patterns (`is_deleted`) plus a unique constraint on the "logical" key causes re-insert conflicts after soft delete.

**What to do instead:**
- Enforce uniqueness in application code (check-then-insert with a transaction, or INSERT ... ON DUPLICATE KEY UPDATE against the *primary* key).
- Use a non-unique composite INDEX for lookup performance.
- If a hard uniqueness invariant is truly required, add a shadow column (`unique_hash`) that is only populated on new writes and has a regular index; enforce collision in app code.
- Document the uniqueness invariant explicitly in the TD so reviewers and future maintainers don't "add the missing UK".

### Do NOT use FOREIGN KEY constraints

**Rule:** No `FOREIGN KEY` on internal MySQL tables.

**Why:** Sharded / sync'd deployments can't honor FK constraints across shards or regions; they also complicate DDL rollouts and lock behavior. Referential integrity is an application concern.

### Do NOT use AUTO_INCREMENT for primary keys

**Rule:** Use the internal ID generator (`bigint(20) unsigned` typically) for primary keys — not `AUTO_INCREMENT`.

**Why:** AUTO_INCREMENT breaks on cross-region sync (two regions generate the same integer), on resharding, and on backup/restore. Use `id-generator` / snowflake-style IDs so every row has a globally unique, time-sortable key.

### Always specify `utf8mb4` and `COLLATE utf8mb4_0900_ai_ci` (or repo's standard)

**Rule:** Match the charset/collation the repo already uses. New tables must not silently downgrade to `utf8` (which is 3-byte and breaks emoji / 4-byte CJK).

### Timestamps

**Rule:** `create_time` / `update_time` as `TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP [ON UPDATE CURRENT_TIMESTAMP]` is the repo convention. Keep it — don't invent `created_at` / `updated_at` naming that breaks existing DAL tooling.

### Soft-delete conventions

**Rule:** Use `is_deleted TINYINT(1) NOT NULL DEFAULT 0` (or the column name the repo already uses). Match the existing DAL filter layer — don't introduce a second convention in the same service.

---

## IDL / Thrift / Kitex

### Never reuse field numbers

**Rule:** Once a Thrift field number is assigned, it is permanent. Removing a field means keeping its slot reserved (commented out); never reassign the number to a new field.

**Why:** Older callers still send the old field; reusing the number causes silent type confusion / data corruption.

### New fields must be `optional` and backwards-compatible

**Rule:** Every new field is `optional`. Don't make a field `required` unless every existing caller will be deployed with the new IDL before the server rejects old requests.

### Method additions are safer than method modifications

**Rule:** When a method's semantics change, prefer adding `methodV2` alongside the old one and deprecating the old one over time. In-place signature changes force lockstep deploys.

### DECC-touched IDLs need a label-plan update before merge

See `decc.md` (sibling reference) — cross-region IDLs need the label version bumped and re-approved *before* the field ships in traffic.

---

## Cache / Abase / Redis

### TTL is mandatory

**Rule:** Every cache key must have an explicit TTL. No `SET key value` without `EX <seconds>`. No "just let it grow" patterns.

### Key convention: `{service}:{entity}:{id}[:{variant}]`

**Rule:** Namespace every key. Don't use bare IDs. Example: `csp_quality:attribute:12345` — not `12345`.

### Single-flight / cache-aside for hot reads

**Rule:** Hot-path cache reads must guard against stampede (use singleflight / mutex on cache miss, or probabilistic early refresh). A 1-second outage on Abase shouldn't translate into N×QPS hitting the DB.

### Do not cache negative results without short TTL

**Rule:** If you cache "not found", use a TTL measured in seconds, not minutes — or a bloom filter. Otherwise deletes leave stale "not found" in cache.

---

## BMQ / RocketMQ

### Consumer ack mode must be explicit in the TD

**Rule:** State whether the consumer is at-most-once / at-least-once / exactly-once (the last requires idempotency on the consumer side). Don't leave this implicit.

### Partitioning key must be deterministic on the business key

**Rule:** If ordering matters per-entity (per-user, per-order), partition by that entity's ID. Don't default to random/round-robin if the consumer expects order.

### DLQ / retry policy must be defined before launch

**Rule:** Every consumer needs: max retries, backoff, dead-letter topic, and an alert on DLQ depth. Not having this is a production incident waiting to happen.

---

## TCC (config service)

### Separate boe / ppe / prod values at the config level, not in code

**Rule:** Use TCC environment/cluster separation for per-env values. Do NOT hard-code `if env == "prod"` branches.

### Never embed secrets in TCC

**Rule:** Secrets belong in KMS v2 / DKMS. TCC is for non-sensitive runtime config only.

### Config changes need a rollback note

**Rule:** Document in the TD: which key, default value, allowed range, and what a bad value causes. A typo in TCC ships instantly.

---

## Feature Gates / Rollout

### Every risky feature needs a kill switch that flips fast

**Rule:** Kill switch must be a single config flip, not a code deploy. TCC key or feature-gate service — not an env var that requires a redeploy to change.

### Canary by PSM cluster or traffic %, not by "prod vs. boe"

**Rule:** boe/ppe don't see prod traffic shapes. A canary that only tests boe has tested nothing. Use % of prod traffic or a single prod cluster.

### Rollback story must cover backfills, not just code

**Rule:** "We'll flip the feature gate" is not a rollback if you already wrote 10M rows. State explicitly what's reversible and what's not. If a backfill is irreversible, the TD must own that constraint in the Rollout section.

---

## Observability

### Metrics naming: `{psm}.{module}.{action}.{outcome}`

**Rule:** Match the repo's existing metric prefix. New top-level namespaces fragment the dashboard story.

### Every write path needs a counter + latency histogram

**Rule:** At minimum: request count by outcome (success/error), p50/p95/p99 latency, downstream call count + latency. Don't ship a write path with only logs.

### logid propagation is mandatory

**Rule:** Every entry-point must extract/assign a `logid` and propagate it through every downstream call. Non-negotiable for debuggability at ByteDance scale.

---

## Security / Permissions

### PII fields must be tagged in the IDL and the DB schema

**Rule:** If a field holds PII, mark it in Thrift annotations and in the DB column comment so downstream tooling (data catalog, masking middleware) can discover it.

### Auth must be at the entry point, not scattered

**Rule:** One auth middleware / interceptor. Not per-handler ad-hoc `if userId == 0` checks.

---

## Code Conventions (fill in per-repo)

> *Placeholder — per-repo conventions should be added as the skill learns each codebase. The generator should read the repo's `CLAUDE.md` / `AGENTS.md` / style docs and append repo-specific rules here.*

- **Go repos:** error wrapping with `fmt.Errorf("…: %w", err)`, not `errors.New(err.Error())`.
- **Spring Boot repos:** favor constructor injection; no field `@Autowired` except in tests.
- **Kitex handlers:** always accept `ctx context.Context` as first param; propagate to every downstream call.

---

## Rules to keep growing

When reviewers flag the same pattern twice across two different TDs, add the rule here. The point of this file is to prevent the same review comment from being written by hand for the 40th time.
