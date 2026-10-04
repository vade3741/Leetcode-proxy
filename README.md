# LeetCode Merchant Verification Proxy

A production-grade reverse-engineered API microservice written in Go. Bridges undocumented upstream GraphQL endpoints into structured, low-latency REST endpoints for candidate technical assessments.

## Quick Start
cp .env.example .env
go run cmd/server/main.go

## Endpoints
- GET /healthz
- GET /api/v1/daily
- GET /api/v1/user/{username}

## Testing
go test -v ./internal/platform -short
