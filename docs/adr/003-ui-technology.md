# ADR-003: React + Vite Embedded in Go Binary

## Status

Accepted

## Date

2025-01-10

## Context

Pulltrace needs a web UI that displays live-updating image pull progress. The UI requirements are:

- Real-time updates via SSE (Server-Sent Events).
- Expandable rows showing per-layer progress within each image pull.
- Filtering and sorting of active pulls.
- Progress bars with download speed and ETA.
- Pod correlation display.
- Works as a single deployment unit (no separate frontend hosting).

We evaluated three approaches:

1. **React + Vite** -- SPA built with Vite, embedded into the Go server binary using `embed.FS`.
2. **HTMX + server-rendered HTML** -- Server generates HTML fragments, HTMX handles partial updates.
3. **Vanilla JavaScript** -- No framework, plain DOM manipulation.

## Decision

Use **React 18 with Vite** for the web UI. The built assets are embedded into the Go server binary using Go's `embed` package, producing a single self-contained binary.

### Build pipeline

1. `cd web && npm run build` produces static assets in `web/dist/`.
2. The Go server uses `//go:embed web/dist` to include these assets at compile time.
3. The server serves the embedded assets at `/` and the API at `/api/`.

### Development workflow

- `cd web && npm run dev` starts Vite's dev server with hot reload.
- Vite proxies `/api` requests to `localhost:8080` (the Go server) during development.

## Rationale

- **React is widely known.** Contributors are more likely to be familiar with React than with HTMX or other alternatives. This lowers the barrier to contribution.
- **Vite builds fast.** Production builds complete in under 5 seconds. The dev server provides instant hot module replacement.
- **Complex client-side state.** The UI manages SSE connections, filtering state, expanded/collapsed rows, computed rates, and ETA calculations. React's component model and hooks (`useState`, `useEffect`, `useRef`) handle this cleanly.
- **Single binary deployment.** Embedding the built assets into the Go binary means Pulltrace installs as two container images (agent and server) with no separate frontend service, CDN, or static file server.
- **Minimal dependency footprint.** The UI has only two runtime dependencies: `react` and `react-dom`. No state management library, CSS framework, or routing library is required.

### Why not HTMX?

HTMX was considered seriously because it would eliminate the JavaScript build step entirely. However:

- SSE handling with HTMX requires either `hx-sse` (which replaces entire DOM fragments) or custom JavaScript. The per-layer expandable detail rows and progress bar animations would require significant custom JS regardless.
- Filtering and sorting on the client side (to avoid server round-trips for every keystroke) would require custom JavaScript state management that HTMX does not provide.
- The complexity savings from avoiding a JS build step are offset by the complexity of managing interactive state in server-rendered fragments.

### Why not vanilla JavaScript?

Vanilla JS was rejected because the UI has enough interactive state (dozens of concurrently updating progress bars, expandable sections, filtering) that manual DOM management would be error-prone and hard to maintain. React's declarative model is a better fit.

## Consequences

- Contributors to the UI need Node.js and npm installed for development.
- The Go build process must run `npm run build` before compiling the server binary. This is handled in the Makefile and Dockerfile.
- The embedded UI is static once compiled. UI changes require rebuilding the server binary.
- React 18 adds approximately 130 KB (gzipped: ~42 KB) to the served assets. This is acceptable for a cluster-internal tool.
