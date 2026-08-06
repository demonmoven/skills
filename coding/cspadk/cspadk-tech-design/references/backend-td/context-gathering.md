# Phase 1 Context Gathering — Sub-phase Commands

Load this file when running Phase 1. Sub-phases 1a–1g below are run in parallel (Agent subagents + direct tool calls) **after** Phase 0 auth is confirmed.

Every subagent prompt MUST include: *"Run `bytedcli auth status` first; if not authenticated, return immediately with a clear error — do NOT attempt login yourself."*

---

## 1a. Fetch PRD / Meego

```bash
lark-cli docs +fetch --doc "<FEISHU_URL>" > /tmp/prd-raw.json
```

Parse `parsed.data.markdown` and `parsed.data.title`. Read it fully. Any `<whiteboard token="..."/>` tags in the PRD markdown can be read via the `lark-whiteboard` skill (`whiteboard +query`) if the content matters.

If requirement is in Meego:

```bash
bytedcli meego ticket get --id <ticket_id>
```

## 1b. Explore the Service Repo

Use the Agent tool with `subagent_type: Explore`. Request:

- Service type & framework version (Spring Boot / Go / Rust / Node)
- Entry points: Thrift RPC handlers, HTTP controllers, MQ consumers, cronjobs, `@Scheduled` tasks
- Thrift IDL files in repo + generated stubs
- Data access: which RDS / ByteDoc / Abase / Redis / TOS / ES clients are configured, connection config
- MQ producers/consumers: topics, consumer groups, ack mode
- Outbound clients: how downstream PSMs are invoked (client factory, timeout/retry config, circuit breaker)
- Domain models that intersect with the new feature
- Observability hooks: metrics, log patterns, tracing integration
- Feature-flag / config mechanism (Settings, TCC, Starling for i18n)
- Test strategy (unit, integration, mock)

Focus on areas the PRD touches. Do not try to map the whole service.

## 1c. Resolve IDL (local-first, Overpass only if needed)

**Local-first.** In Spring Boot / Java repos at ByteDance, downstream IDLs are almost always vendored in a dedicated SPI Maven module (e.g. `<repo>-spi/src/main/thrift/` or compiled `<repo>-spi/target/classes/*.thrift`). Check the repo FIRST before calling Overpass.

1. **Locate the SPI module**: glob for `*-spi/` directories or any `*.thrift` files under the repo. This typically covers: (a) the service's own IDL and (b) every downstream PSM the service calls.
2. **Enumerate methods the service actually invokes**: grep generated stub packages (`com.bytedance.<psm>.thrift.*Service`) against the source tree. Record request/response types per method — capture the **complete** field-level definition (field number, type, required/optional, description) for every added/modified request and response struct. The frontend team depends on these full definitions to formulate their technical plan, so do not summarize or abbreviate — include every field including nested structs expanded to leaf level.
3. **For the service's OWN IDL**: identify methods that will add / modify / deprecate in this TD. Also capture full IDL definitions for any existing methods called by the frontend, even if unchanged — the frontend team needs complete parameter specifications to design their integration.

**Fall back to Overpass** only when one of these is true:
- The downstream PSM's IDL is NOT vendored in the repo (e.g. a new dependency added by this TD).
- You need to verify whether the vendored copy is stale vs. the team's published master — a drift check when the TD proposes depending on a recently-added field.
- The user explicitly asks to compare against the latest.

```bash
bytedcli overpass idl get --psm <PSM> > /tmp/idl-<psm>.thrift
# TODO v0.3: pin exact overpass subcommand from bytedcli overpass subskill reference
```

For non-Java/non-Spring repos (Go using `kitex_gen/`, Rust, etc.) the same principle holds: prefer the repo's vendored IDL, use Overpass only as the fallback.

## 1d. Inventory Middleware Dependencies

For each store discovered in 1b, pull authoritative info via bytedcli subskills:

- **RDS / ByteDoc**: cluster, DB, affected tables, approx row counts, existing indexes
- **TOS**: bucket, region, access pattern
- **BMQ / RocketMQ**: topic, partitions, consumer group, current TPS, retention
- **Abase / Redis**: PSM, key conventions (grep repo), TTL
- **ES**: cluster, index name, shard config

Record each as a manifest row with status NEW / EXISTING / MODIFIED. Exact bytedcli commands per store type are in the `bytedcli` skill's subskill references — invoke that skill when you need command syntax rather than hardcoding here.

**RDS table schema (frequent, easy to get wrong):**

```bash
bytedcli --site <cn|boe|i18n|i18n-bd|i18n-tt|us-ttp|...> rds db table schema <dbname> <table> -r <region>
# region examples: cn / alisg (SG) / maliva (VA) / useast5
# if site returns "地域不存在 xxx 库" or HTTP 404 "未查到指定的Domain和URL", try another --site
# search first when unsure: bytedcli --site <site> rds db search <keyword>
```

## 1e. Observability Baseline (only if feature classifies as "traffic delta")

```bash
# TODO v0.3: pin exact APM/Slardar commands
bytedcli apm metrics query --psm <PSM> --method <method> --window 7d
```

Capture p50 / p99 / QPS / error-rate for affected endpoints.

## 1f. Upstream Caller Discovery

**Default: user-supplied.** Ask the user which PSMs call the methods being changed. Auto-discovery (APM trace aggregation, codebase-search across caller repos) is a v0.3 enhancement — current version requires user input so we don't ship a confidently-wrong caller list.

## 1g. Prior Art & Internal Tooling Research  [run when PRD introduces a new tech/concept/integration pattern]

Before designing a solution, check whether an internal tool, platform, or library already solves this. Building from scratch when an internal option exists wastes effort and fragments the stack.

**Trigger:** any of these signal this step matters —
- PRD names a capability the service doesn't already have (e.g. "stateful workflow", "approval flow", "rate limiting", "cross-region cache")
- User's implementation idea mentions a new third-party library
- Feature requires integrating with a system the repo doesn't currently touch

**Run all three source types in parallel:**

> **Search in BOTH English and Chinese.** A large fraction of internal TDs, ByteTech articles, and BitsAI Q&A are written in Chinese. Searching only English misses the majority of prior art. For every concept, run the query twice — once with the English term, once with the Chinese translation (e.g. `"rate limiting"` + `"限流"`, `"idempotency"` + `"幂等"`, `"cross-region sync"` + `"跨地域同步"`). Merge the hits. If you're not sure of the Chinese term, ask the user or pick the term used in the PRD.

```bash
# Broad keyword search across 5 sources — good for TDs, design docs, articles, solved Q&A
bytedcli insearch query "<concept>" --source feishu.cn              # Lark wiki & docs (team TDs, design docs, ADRs)
bytedcli insearch query "<concept>" --source cloud.bytedance.net    # ByteCloud platform pages
bytedcli insearch query "<concept>" --source bitsai.bytedance.net   # Engineering Q&A (solved problems)
bytedcli insearch query "<concept>" --source bytedance.net          # Intranet portals + ByteTech tech articles
# Bilingual: re-run each of the above with the Chinese term
bytedcli insearch query "<中文概念>" --source bytedance.net          # ByteTech / 字节内网 articles (most are CN)
bytedcli insearch query "<中文概念>" --source feishu.cn              # CN TDs and design docs

# Fetch a specific ByteTech article by ID (from a URL like https://bytetech.bytedance.net/article/<id>)
bytedcli insearch get <article_id>

# Structured Cloud Docs search — official SDK/middleware reference docs for TCC, TCE, Kitex, Hertz, BMQ, RDS, TOS, Abase, Archer, Settings, Starling, etc.
bytedcli cloud-docs search "<concept>"

# For a specific platform, browse its full API doc set:
bytedcli cloud-docs list-business
bytedcli cloud-docs list-docs "<business_id>" --api-doc "tce:v1"
bytedcli cloud-docs get "<doc_id>"
```

**Why both.** `insearch` is fuzzy keyword search across docs written by humans — good for finding prior designs, solved problems, and team knowledge. `cloud-docs` is structured, official, SDK-level reference maintained by platform teams — good for "what's the canonical API for X", "does this SDK support Y", "what are the default timeouts". Run both; they catch different things.

**Query wording matters.** Search for the *capability* (e.g. "approval workflow framework", "distributed rate limiter java", "idempotency key middleware"), not the PRD's feature name. Run 2–3 queries per concept with different phrasings — Chinese terms often match better for internal tooling (e.g. `审批流`, `限流`, `幂等`).

**Fetch top hits:**
```bash
bytedcli insearch get "<url>"
```

**Synthesize findings** into the TD's **Technology Selection & Key Decisions** section. For each candidate internal tool: one row in the comparison table with "Pros / Cons / Fit". Recommend "adopt", "adapt", or "build-our-own" with explicit reasoning. If you recommend "build-our-own", cite why the internal options don't fit — don't silently ignore them.

If no internal option exists, say so in the TD ("Searched: X / 中文X, Y / 中文Y, Z / 中文Z across feishu.cn + cloud.bytedance.net + bitsai + ByteTech — no existing tooling found"). This makes the decision defensible to reviewers — and explicitly noting the Chinese queries shows you didn't miss half the corpus.

## Resolving SDK / implementation doubts mid-design (Phase 3 usage)

When writing a feature-design section you hit a specific question ("does the Abase SDK support conditional writes?", "what's the default retry policy for the Kitex client factory?", "can BMQ guarantee ordering per partition key?"), don't guess or mark TODO — look it up immediately:

```bash
bytedcli cloud-docs search "<specific SDK question>"          # structured platform docs
bytedcli insearch query "<specific SDK question>" --source bitsai.bytedance.net   # engineering Q&A — often has the exact answer from someone who hit the same question
bytedcli insearch query "<specific SDK question>" --source cloud.bytedance.net    # broader ByteCloud pages
```

Capture the answer with a citation (doc URL) into the TD's relevant section. If the answer isn't found, add an explicit Open Question with the queries you tried — reviewers can then point you at the right doc.
