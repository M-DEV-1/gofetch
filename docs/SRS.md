# SRS - Concurrent Web Crawler & Search Indexer

This is a GPT-generated SRS document which I used to refer, while coding gofetch.

## 1. Document purpose

**Purpose:** Provide a comprehensive specification for a single-machine, production-capable web crawler and search indexer implemented in Go. Covers functional requirements, non-functional requirements, architecture, APIs, data models, operational needs, test plans, and deployment.

---

## 2. Project summary

A highly concurrent, resilient, configurable web crawler and lightweight search indexer that can:

* crawl seed URLs with politeness and per-domain throttling,
* extract and tokenize page content,
* persist and serve an inverted index with ranking (TF-IDF/BM25),
* expose REST APIs (and optional web UI) for control, search, and metrics,
* resume after crashes and run containerized.

Designed to teach and demonstrate advanced Go concurrency patterns, safe persistence, HTTP services, and operational hygiene.

---

## 3. Goals & non-goals

### Goals

* Robust concurrency: worker pools, pipeline stages, and graceful cancellation.
* Safety: dedupe, politeness, backoff, circuit breakers, and avoidance of livelock.
* Persistence: resumeable frontier and disk-backed index.
* Search: reasonable ranking (TF-IDF or BM25) and fast query times for medium datasets (100k pages).
* Observability: metrics, logs, health checks, and profiling endpoints.
* Containerized deployment with sane defaults.

### Non-Goals

* Extremely large-scale crawling across the whole web. Not distributed by default.
* Replace production search engines. Ranking will be simple, not neural.
* Full robots.txt enforcement or advanced crawler traps detection (optional extensions).

---

## 4. Actors & stakeholders

* **Operator**: configures, starts, stops, and monitors the crawler.
* **API client**: issues start/stop/search/status requests (curl, UI).
* **Indexer**: internal process that updates index storage.
* **Crawler workers**: fetch and parse pages.
* **Storage**: embedded DB (bbolt/Badger) + local filesystem.

---

## 5. Functional requirements (FR)

### FR1 - Crawl Control

* FR1.1: `POST /crawl` - start a crawl with JSON options: `seeds[]`, `maxDepth`, `maxPages`, `concurrency`, `perHostRate`, `resume` flag.
* FR1.2: `POST /control` - actions: `pause`, `resume`, `stop`, `snapshot`, `reseed`.
* FR1.3: `GET /status` - current state: `running|paused|stopped`, queued count, active workers, pages crawled, errors.

### FR2 - Fetching & Parsing

* FR2.1: Fetch workers obey per-host politeness (requests per second, concurrency per host).
* FR2.2: Respect global concurrency limits.
* FR2.3: Each fetch respects request timeout and retry policy (configurable attempts, exponential backoff).
* FR2.4: Parse HTML to extract title, meta description, main text, and links; extract content language where possible.

### FR3 - Frontier & Deduplication

* FR3.1: URL canonicalization and normalization (scheme, trailing slashes, query normalization heuristics).
* FR3.2: Thread-safe visited set and duplicate suppression (persisted for resume).
* FR3.3: Frontier is a prioritized queue (BFS by default, depth-limit aware), persisted for crash resume.

### FR4 - Indexing & Search

* FR4.1: Tokenize and normalize (lowercase, basic stemming/stoplist optional) text into terms.
* FR4.2: Maintain posting lists: term -> [(urlID, termFreq, positions...)].
* FR4.3: Compute document statistics (doc length, term frequencies) for ranking.
* FR4.4: Implement TF-IDF and/or BM25 ranking. Support boolean and simple phrase queries.
* FR4.5: `GET /search?q=...&limit=20&offset=0&rank=bm25|tfidf` returns results: URL, title, score, snippet.
* FR4.6: Ability to re-index pages, purge documents, and compact index.

### FR5 - Persistence & Resume

* FR5.1: Persist index and frontier to embedded DB so the crawler can resume where it left off.
* FR5.2: Snapshot capability (manual or scheduled) with integrity checks.

### FR6 - Metrics & Observability

* FR6.1: `GET /metrics` JSON and Prometheus-compatible `/metrics`.
* FR6.2: Logs with structured JSON (levels: debug/info/warn/error).
* FR6.3: Health endpoints: `/healthz`, `/readyz`.
* FR6.4: Profiling endpoints (pprof) behind admin auth or dev flag.

### FR7 - Security & Access Control

* FR7.1: API access control using API keys or basic auth (configurable).
* FR7.2: Input validation for seeds and control actions to avoid injection.

### FR8 - Admin Operations

* FR8.1: Admin endpoints: force GC, show goroutine dumps, trigger snapshot, clear visited.
* FR8.2: Config reloading (hot reload) of non-breaking parameters.

---

## 6. Non-functional requirements (NFR)

### NFR1 - Performance

* Support steady crawling of **≥ 100 requests/sec** (aggregate) on a decently provisioned single machine when tuned (hardware dependent).
* Search latency: median < 100ms for small-medium indexes (≤100k docs).

### NFR2 - Reliability

* Resume after crash within bounded time; no duplication of indexing across restarts.
* Graceful shutdown: finish in-flight fetches or abort cleanly within configured timeout.

### NFR3 - Scalability

* Vertical scale: tune concurrency, but not horizontally distributed in core design.
* Storage scalability: index should handle tens to hundreds of thousands of documents before needing sharding.

### NFR4 - Maintainability

* Clear package separation, unit tests, and CI checks (`go test`, `-race`, `golangci-lint`).
* Interfaces for pluggable components (Fetcher, Parser, Storage, Indexer).

### NFR5 - Security & Privacy

* Sanitize fetched content (avoid executing scripts, isolate storage).
* Default to reasonable crawl limits to avoid accidental DDoS.

---

## 7. System architecture (high level)

### Components

* **API Server** - HTTP handlers, auth, and admin endpoints.
* **Crawl Manager** - orchestrates crawl sessions, lifecycle, and policies.
* **Frontier Manager (Queue)** - persistent, prioritized queue with resume.
* **Fetcher Pool** - goroutine pool fetching URLs, honoring rate-limits and timeouts.
* **Parser Pool** - extractors that parse HTML into tokens and links.
* **Deduper / Normalizer** - URL normalization and visited tracking.
* **Indexer** - serial or sharded index builder writing to embedded DB.
* **Storage** - key-value DB (bbolt or Badger) to store posting lists, doc metadata, frontier, and visited set.
* **Rate Limiter** - per-host token buckets or leaky buckets.
* **Circuit Breaker** - per-host failure detector throttling retries.
* **Metrics & Logging** - Prometheus client and structured logger.

### Data flow (sequence)

1. Operator POSTs seeds to API.
2. Crawl Manager pushes canonicalized seeds into Frontier.
3. Fetcher goroutines pull from Frontier, request pages, and push results to Parser channel.
4. Parsers extract tokens and links. Tokens go to Indexer; links go back to Frontier through Deduper.
5. Indexer serializes updates to Storage. Metrics updated throughout.

---

## 8. Data model (representative)

### Page / Document

```
DocID (uint64)
URL (string)         // canonical
Title (string)
Snippet (string)
Status (int)         // HTTP status
FetchedAt (timestamp)
ContentLength (int)
TermCount (int)
Metadata (map[string]string)
```

### Inverted Index (storage model)

* Key: `term` → Value: posting list serialized (list of (DocID, termFreq, positions...))
* Doc store: DocID → serialized Page metadata
* Global stats: `N` (num docs), docLength map, termDocFreq map

### Frontier & Visited

* Frontier entries: (URL, priority/depth, seedID)
* Visited set: canonical URL hash → DocID or visited timestamp

---

## 9. APIs (contract)

### Control & Management

* `POST /crawl` - body: `{ seeds: [url], options: { maxDepth, maxPages, concurrency, perHostRate } }` → 202 Accepted `{sessionID}`
* `POST /control` - body: `{ sessionID, action: "pause"|"resume"|"stop"|"snapshot" }` → 200 `{status}`
* `GET /status?sessionID=...` → 200 `{ state, queued, workersActive, pagesCrawled, errors }`

### Search

* `GET /search?q=term&limit=10&offset=0&rank=bm25`
* Response: `[{url, title, score, snippet, docID}]`

### Metrics & Health

* `GET /metrics` → Prometheus format or JSON
* `GET /healthz` → 200 OK if healthy

### Admin (protected)

* `POST /admin/snapshot` → trigger snapshot
* `GET /admin/goroutines` → goroutine dump (dev only)
* `POST /admin/clearVisited` → wipe visited (dangerous)

---

## 10. Concurrency & synchronization patterns

* **Worker pools** using goroutines and buffered channels per stage. Each pool controlled by `sync.WaitGroup`.
* **Indexing**: single-threaded indexer recommended (avoid locking), or sharded indexer with consistent hashing + independent locks.
* **Visited set**: `sync.RWMutex` + in-memory map for hot checks, with periodic persistence to DB. Or `sync.Map` for concurrency ease.
* **Rate limiting**: per-host `chan struct{}` token buckets controlled by tickers or time.After.
* **Cancellation**: root `context.Context` for sessions. All goroutines should select on `ctx.Done()`.

---

## 11. Fault tolerance & error handling

* Retries: exponential backoff with capped attempts.
* Circuit breaker: open after N consecutive failures, grace period, then trial requests.
* Throttling: when system metrics (CPU, memory) exceed thresholds, reduce concurrency (backpressure).
* Poisoned items: if parsing fails repeatedly, tag and drop after configurable threshold.
* DB corruption: maintain periodic snapshots and write-ahead logs (optional).

---

## 12. Security considerations

* Rate limit API endpoints to avoid abuse (API key + per-key quotas).
* Sanitize user input and validate seed URLs.
* Limit allowed schemes (http/https only).
* Run pprof and admin endpoints behind auth or dev-only flag.
* Never execute or render fetched HTML; treat it as data only.
* Monitor disk usage to avoid filling disk with large indices.

---

## 13. Testing strategy

### Unit tests

* URL normalization functions.
* Visited set under concurrent access.
* Tokenizer and ranker correctness (TF-IDF/BM25).
* Rate limiter behavior.

### Integration tests

* Use `httptest.Server` with routes that simulate slow, flaky, and normal responses.
* Start a small crawler session against the test server; assert dedupe, frontier behavior, index contents.

### System tests

* Simulate real websites (a few hundred pages) and measure throughput and memory.
* Crash/restart test to confirm resume works.

### Concurrency tests

* `go test -race` for hot paths.
* Tests that intentionally add sleeps and flakiness to reproduce deadlock and race scenarios.

---

## 14. Performance & profiling

* Use `pprof` for CPU and memory profiles.
* Identify hot functions (tokenization, posting list updates) and optimize (e.g., batch writes, pooling).
* Monitor GC pause times and consider tuning `GOGC` or indexing strategies if allocation heavy.

---

## 15. Deployment & Docker

### Container

* Use multi-stage Dockerfile:

  * Build the Go binary in `golang:1.x` image.
  * Copy binary into small base (`scratch` or `gcr.io/distroless/static`) for runtime.
* Expose port (default `8080`).
* Mount a volume for persistent DB files.
* Environment variables or mounted config for runtime params.

### Compose (local dev)

* `docker-compose` with service:

  * `crawler` (the app)
  * optional `admin` UI (if built)
* Volume for index DB and logs.

### CI/CD

* Lint, test (`-race`), build binary, produce container image, run integration smoke tests.
* Tag images with git SHA.

---

## 16. Observability & ops

* Metrics: Prometheus metrics for pages/sec, error counts, queue length, per-host latency, memory/GC.
* Logs: structured JSON including request IDs/session IDs and URL hashes.
* Alerts: disk usage, high error rate, worker starvation, long queue growth.
* Backups: periodic DB snapshot and export for offline analysis.

---

## 17. Extensibility & future work

* Distributed mode: coordinator + worker nodes (gRPC), consistent hashing for index sharding.
* Advanced ranking: phrase queries, proximity scoring, language-aware tokenizers, embeddings.
* Politeness improvements: full robots.txt & sitemap parsing, crawl-delay respect.
* UI: interactive graph, crawl visualizer, index explorer.

---

## 18. Acceptance criteria (how to know it's done)

* End-to-end crawl of seeds → index built → search returns relevant results.
* Resume works: stop process, restart with `resume=true`, continue crawl without duplicate indexing.
* Test suite: unit + integration tests pass (including `-race`).
* Container image builds and runs with documented Docker commands.
* Basic metrics, health checks, and safe shutdown implemented.

---

## 19. Glossary

* **Frontier**: queue of URLs to be fetched.
* **Posting list**: list of documents containing a term.
* **DocID**: internal numeric identifier for a document.
* **Politeness**: per-host constraints to avoid hammering a server.
* **Tokenization**: transforming text into searchable terms.

---

## 20. Milestones (practical roadmap to reach final boss)

1. Skeleton & CLI toy (single URL).
2. Simple concurrent crawler (fetch + parse + in-memory index).
3. API server + control endpoints.
4. Persistent index + frontier (bbolt).
5. Ranking (TF-IDF), snippets, and search endpoint improvements.
6. Per-host rate limiting, circuit breaker, backoff.
7. Metrics, health checks, profiling.
8. Dockerization + CI + integration tests.
9. Production hardening: snapshots, admin tools, and optional UI.

---

## Appendix: Recommended tech choices (opinionated)

* Language: Go 1.22+.
* DB: `go.etcd.io/bbolt` for simplicity or Badger for performance.
* HTML parsing: `golang.org/x/net/html`.
* Logging: `zerolog` or `logrus` (structured).
* Metrics: Prometheus client for Go.
* Router: standard `net/http` for core; migrate to `chi`/`gin` in iteration 2 if needed.