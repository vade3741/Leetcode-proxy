# LeetCode Merchant Verification Proxy

A production-grade reverse-engineered API microservice written in Go. Bridges undocumented upstream GraphQL endpoints into structured, low-latency REST endpoints for candidate technical assessments and daily challenge feeds.

## Merchant Context & Forward-Deployed Engineering Discovery

### The Merchant Problem
Technical hiring and EdTech platforms process thousands of candidate profiles monthly. To evaluate algorithmic proficiency and problem-solving consistency, recruiting coordinators manually open candidate profiles, record solved problem counts in spreadsheets, and cross-reference contest rankings.
* Labor Overhead: 10–15 minutes spent per candidate manually checking profiles.
* Data Staleness: Static spreadsheet records become obsolete within days as candidates continue solving problems.
* Transcription Errors: Manual data entry introduces human error into candidate scoring.

### Uncovering the Root Problem
The merchant initially requested a browser automation bot (Puppeteer/Playwright) to scrape public profile pages. However, browser automation introduces excessive memory/CPU overhead, high latency (5–15 seconds per page render), and frequent breaks whenever frontend CSS classes change.

Solution: Reverse-engineer the unauthenticated GraphQL endpoint powering the platform's public frontend (https://leetcode.com/graphql) to provide a clean, sub-second REST API tailored for direct ATS integration.

## Architecture & Endpoints

1. Healthcheck: GET /healthz
   Response: {"success":true,"data":{"operational_status":"ready"},"execution_time":"0s"}

2. Candidate Profile Verification: GET /api/v1/user/{username}
   Returns verified problem metrics, contest rankings, and difficulty breakdowns.

3. Active Daily Challenge: GET /api/v1/daily
   Returns metadata and direct challenge links for the active daily problem.

## Solution Limitations & Vulnerability Analysis

1. Upstream Schema Drift: Because internal GraphQL endpoints lack semantic versioning, field renames break JSON unmarshaling without advance notice.
2. Perimeter Controls & Rate Limiting: High-frequency polling using standard HTTP clients risks triggering perimeter bot heuristics (e.g., Cloudflare/DataDome WAF) due to missing browser TLS signatures.
3. Protocol Error Ambiguity: GraphQL backends return HTTP 200 even for execution errors, embedding error details inside a top-level errors array.

## Long-Term Fix & Production Architecture

1. Caching Tier: Introduce Redis caching with Time-to-Live policies (8–24 hours for candidate profile metrics, midnight UTC rollover for daily challenges) and token-bucket circuit breakers to insulate the upstream service.
2. Automated Canary Testing: Schedule synthetic canary tests every 15 minutes to alert engineers to upstream schema shifts before downstream merchant workflows experience downtime.
3. Strategic Resolution: Negotiate official partner API access or migrate candidate assessment to SLA-backed verification providers (such as HackerRank, CodeSignal, or GitHub APIs).

## Quick Start & Testing

Run unit and mocked tests:
go test -v ./internal/platform -short

Run complete suite including live canary check:
go test -v ./internal/platform

Build and start server:
go build -o server cmd/server/main.go
./server
