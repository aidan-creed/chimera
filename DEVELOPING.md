# Contributing to Chimera

## Quick Start

### Prerequisites
- Go 1.23+
- Node 20+
- Docker & Docker Compose
- EVE Online account

### First Time Setup
```bash
# Clone and setup
git clone https://github.com/jjckrbbt/chimera
cd chimera

# Backend
cd backend
cp .env.example .env
# Edit .env with your ESI credentials
docker compose up -d
go run cmd/server/main.go

# Frontend
cd ../frontend
npm install
npm run dev
```

Visit http://localhost:5173

## Project Structure
```
chimera/
├── backend/          # Go API server
│   ├── cmd/server/   # Main entry point
│   ├── internal/     # Private application code
│   └── configs/      # YAML configs for data pipelines
├── frontend/         # React TypeScript app
└── embeddings/       # Python AI service
```

## Development Workflow

- **Backend changes**: Hot reload with `air` (install: `go install github.com/cosmtrek/air@latest`)
- **Frontend changes**: Vite auto-reloads
- **Database migrations**: `goose -dir backend/migrations postgres "..." up`

## ESI Rate Limits

- Respect ESI cache headers
- Max 150 req/sec burst, 20 req/sec sustained
- Use `esi_cache` table for caching

## Getting Help

- Check existing issues
- Ask in discussions
- Join our Discord: [link]

## Code Style

- **Go**: `gofmt` and `golangci-lint`
- **TypeScript**: `eslint` and `prettier`
- Write tests for new features

---

**First contribution ideas**: Check issues labeled `good-first-issue`
