# LivePoll

LivePoll lets a signed-in host create and manage polls, share public voting links, and watch votes arrive live. Its signed-in dashboard includes an overview, searchable and paginated poll list, poll creation preview, live results, saved-vote analytics with CSV export, profile/settings, and a poll assistant. The frontend is React/Vite, the API is Go/Gin, MongoDB stores durable poll and vote records, and Redis maintains result counters and fans out live events through Server-Sent Events (SSE).

## Local development

1. Start MongoDB and Redis from the project root:

   ```powershell
   docker compose up -d
   ```
2. Create `backend/.env` with `PORT`, `MONGODB_URI`, `MONGODB_DATABASE`, `REDIS_ADDR` (or a hosted `REDIS_URL`), `JWT_SECRET` (at least 32 characters), and `FRONTEND_ORIGINS=http://localhost:5173`. `REDIS_URL` takes precedence and supports hosted TLS Redis endpoints. To enable conversational AI, also set `OPENAI_API_KEY` and optionally `OPENAI_MODEL` (defaults to `gpt-5-mini`). Keep the API key in the backend environment only; never add it to the frontend. Do not commit this file.
3. In one PowerShell window run the API:

   ```powershell
   cd backend
   go run .
   ```
4. In another PowerShell window run the frontend:

   ```powershell
   cd frontend
   npm run dev
   ```
5. Open `http://localhost:5173`, sign up, create a poll, and open its share link in a second browser session. Vote in one session and observe the result update in the other without refreshing. The authenticated dashboard is available at `/dashboard`; analytics are calculated from saved MongoDB votes.

The Poll Assistant uses built-in, intent-based guidance when `OPENAI_API_KEY` is not configured. With a backend key configured, it uses the OpenAI Responses API, includes recent chat turns for follow-up context, and may use the signed-in user's poll aggregates to answer results questions. The key is never sent to the browser.

For a deployed frontend, set `VITE_API_URL` to the backend API base URL ending in `/api`. Configure the backend's `FRONTEND_ORIGINS` as a comma-separated list of exact origins.

## API

- `POST /api/auth/register`, `POST /api/auth/login`
- `GET /api/auth/me` (JWT required; returns the signed-in profile without password data)
- `GET /api/polls` (JWT required; supports `page`, `limit` (max 100), `status=all|active|closed`, `sort=newest|oldest|votes`, and `q` search), `POST /api/polls` (JWT required)
- `GET /api/dashboard/summary` (JWT required; real poll and saved-vote totals plus recent polls)
- `GET /api/analytics` (JWT required; supports optional `from`, `to` dates in `YYYY-MM-DD`, `pollId`, and `status`; date windows are limited to one year)
- `POST /api/assistant/chat` (JWT required; returns either a labeled AI response or built-in guidance, depending on backend configuration)
- `PATCH /api/polls/:id` (JWT required; owner only; edits the question while open), `PATCH /api/polls/:id/close` and `DELETE /api/polls/:id` (JWT required; owner only)
- `GET /api/polls/:id`, `POST /api/polls/:id/votes`, `GET /api/polls/:id/events` (public)
- `GET /api/health`

Poll question length is 5–180 characters; option count is 2–8, each 1–80 characters, with duplicates rejected. A poll's public ID and each option ID are random ObjectID strings. MongoDB stores metadata in `polls` and accepted votes in `votes`. A unique compound index on `(poll_id, voter_id)` makes one-vote-per-participant enforcement atomic at the database layer. The browser creates a UUID in local storage as its participant identity; this prevents accidental repeat votes in that browser, not determined abuse or identity sharing.

Redis stores option totals in a hash and publishes vote events on a poll-specific channel. The SSE endpoint subscribes before sending a fresh Redis snapshot, then streams events and keepalives. Redis Pub/Sub can drop messages while disconnected, so reconnecting clients receive a new snapshot. If a counter hash is entirely missing, the API rebuilds it from MongoDB's durable vote records. MongoDB accepts the vote before Redis increments it; if Redis is unavailable the vote stays durable and a missing cache can be rebuilt, though an existing partial cache may need an operator rebuild.

## Checks

```powershell
cd backend
gofmt -w main.go auth.go dashboard.go poll.go dashboard_test.go poll_test.go
go test ./...
go vet ./...
cd ..\frontend
npm run lint
npm run build
```

## Deployment and submission checklist

- [ ] Deploy the Go API and React frontend; configure MongoDB, Redis, JWT secret, CORS origins, and `VITE_API_URL` in the hosting environment.
- [ ] Verify signup, poll creation, sharing, voting, and cross-browser live updates on the public site.
- [ ] Push the project to a public GitHub repository.
- [ ] Record the required 3–5 minute walkthrough, including the toughest challenge and an honest description of AI assistance.
- [ ] Email the public repository, deployed URL, and video URL to `devhiring@hclguvi.com`.

Deployment, publication, video recording, and the submission email still need the account access and URLs supplied by the project owner. No links are claimed here.

## Demo video outline (3–5 minutes)

- **0:00–0:30:** Introduce LivePoll and its React, Gin, MongoDB, and Redis roles.
- **0:30–1:30:** Sign in, create a poll, and copy its public link.
- **1:30–2:30:** Open the link in a separate browser profile, vote, and show the first session's result changing without a refresh.
- **2:30–3:30:** Explain the difficult part: preserving accepted votes in MongoDB while keeping Redis counters and SSE updates in step, plus the unique index that rejects duplicate participant votes.
- **3:30–4:00:** Explain AI use honestly. For example, describe where Codex helped inspect or implement code, what you checked yourself, and any changes you made after reviewing its output. Use your own accurate account.
- **4:00–4:30:** Mention limitations: participant IDs are browser-local and are not strong anti-abuse identity; Redis Pub/Sub reconnects recover using a fresh snapshot.

