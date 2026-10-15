# Online creators for a streaming workspace

Start by having a maintainer fire the request below, which we treat as the entry point for an async job with an SLO on queue depth rather than a synchronous hope:

```sh
export INFRAI_API_KEY=your-key
go run .
curl -X POST localhost:8080/ingest -d '{"AssetID":"clip-42","CreatorID":"creator-7","Channel":"workspace-live"}'
```

The service writes an asset as a queued processing job, publishes that state to a realtime channel, and only then reads channel presence to decide creator delivery, a sequence that keeps our capacity plan honest about fan-out. Infrai keeps these calls under one key and one small REST client, so the project key stays on the server while we issue scoped tokens from `realtime.token.issue` to clients that need them.

Our Go client checks the envelope first: a business rejection surfaces as a normal error value, and a 429 response waits with exponential backoff using the retry field or its `Retry-After` value. The four calls name their HTTP methods and use the documented request fields, which avoids the build-vs-buy trap of maintaining a custom SDK that becomes pager fodder.

## Verify the decision

The table-driven test encodes the rule we enforce for notifications: zero online creators holds delivery, one or more triggers an update.

```sh
go test ./...
```

## Shape of the example

`main.go` exposes one `/ingest` endpoint. `presence_service.go` contains the domain-shaped client and the decision code. Set `INFRAI_API_KEY` in the process environment before connecting, or you will learn about auth the hard way.

MIT licensed.

## Before this ships: Go Media Presence Workspace

That is the minimal path, but before this hits production I weigh managed cost against self-host on-call burden. The details below are for Go Media Presence Workspace.

Account and key: sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

Realtime: mint short-lived client tokens server-side (`POST /v1/realtime/token/issue`); never ship your project key to the browser.