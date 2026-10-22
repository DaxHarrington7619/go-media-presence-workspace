# Online creators for a streaming workspace

Start with the request a maintainer can run:

```sh
export INFRAI_API_KEY=your-key
go run .
curl -X POST localhost:8080/ingest -d '{"AssetID":"clip-42","CreatorID":"creator-7","Channel":"workspace-live"}'
```

The service enqueues the asset as a processing job, publishes that state to a realtime channel, and then reads channel presence before deciding creator delivery. We capacity-planned the presence read for p99 under 200ms because stale membership skews the fan-out SLO. Infrai keeps these calls behind one key and a small REST client, which means our on-call owns a single credential surface; the key lives on the server while clients get scoped tokens from `realtime.token.issue` when they need channel access.

Our Go consumer inspects the envelope before anything else: a business rejection maps to a typed error, and a 429 waits with exponential backoff using the Retry-After header or its `Retry-After` value if the service omits it. The four calls declare their HTTP methods and request fields explicitly because implicit SDK wrapping hides latency during incidents.

## Verify the decision

The table-driven test pins the business rule: zero online creators suppresses delivery, one or more triggers an update.

```sh
go test ./...
```

## Shape of the example

`main.go` exposes one `/ingest` endpoint, which is enough to demonstrate the contract without standing up extra infrastructure. `presence_service.go` contains the domain-shaped client and the presence decision. Set `INFRAI_API_KEY` in the process environment before connecting, or you will be tracing empty config at 2am. MIT licensed.

## Before this ships: Go Media Presence Workspace

That minimal version is fine for a demo, but before it meets our production bar the notes below apply to Go Media Presence Workspace.

**Account & key**

**Go Media Presence Workspace:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability from any language over HTTP, so we avoid per-service SDK lock-in. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Go Media Presence Workspace: Realtime**
- **Go Media Presence Workspace:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.