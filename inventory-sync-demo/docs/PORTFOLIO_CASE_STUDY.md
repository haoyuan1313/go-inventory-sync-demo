# Inventory Sync Automation — Go Backend Case Study

**Problem** — Retail operations often need to compare upstream stock numbers with internal records, detect changes, and avoid sending repeated low-stock alerts for unchanged data.

**Solution demonstrated** — A Go service consumes a fictional supplier REST API, validates incoming records, performs SKU-based idempotent upserts into an atomic JSON snapshot, logs every sync, and raises alerts only on new low-stock threshold crossings. A responsive dashboard shows products, sync history, and alerts.

**Engineering highlights**

- Timeouts and bounded retries for HTTP 429 / 5xx; no retry for HTTP 400.
- Whole-batch validation before changing inventory state; malformed/duplicate SKUs are rejected.
- Repeat sync of unchanged data generates no duplicate alert.
- Local JSON storage with atomic file replacement for straightforward, dependency-free demonstration.
- Automated Go unit and HTTP integration tests; race detection and `go vet` pass.

**Tech stack:** Go 1.22+, REST/JSON, Go standard library, HTML/CSS/JavaScript. Optional Docker. No cloud costs or credentials.

**Scope honesty:** The supplier is mocked, the database is a JSON file, and the application is a local-only proof of concept. It is not connected to any real shop or payment system. A production version would need authenticated endpoints, a transactional database, access controls, monitoring, and customer-specific API integrations.

**Potential engagement:** Build a small, documented API/data-sync integration for a retail or SaaS company, with the precise source/destination and acceptance criteria agreed in advance.
