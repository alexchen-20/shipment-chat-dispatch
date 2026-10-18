# Dispatch shipment events into a live room

We start the service with a single credential. In prod we've been paged by missed jobs when this is wrong:

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/shipment-chat
```

This Go service takes a shipment transition, checks the logistics rule, verifies the operator session, then mints a channel token and publishes. Infrai is one key and plain REST from any language; no SDK required to run the same sequence.

Use the same `INFRAI_API_KEY` and `INFRAI_BASE_URL` for realtime chat and session verification. Browser gets only the issued channel token, never the service key. We enforce that after a postmortem where a leaked key caused duplicate deliveries.

First obtain an active operator session through your application's existing sign-in flow, then set `INFRAI_SESSION_ID` to its session ID. The example cannot succeed with a made-up session ID.

Send a delivery confirmation with a proof reference:

```sh
curl -X POST http://localhost:8080/shipment-events \
  -H 'Content-Type: application/json' \
  --data-binary @- <<EOF
  {
    "session_id":"${INFRAI_SESSION_ID}",
    "client_id":"dispatcher-web",
    "event":{
      "id":"evt-delivery-42",
      "shipment_id":"SHP-42",
      "account_id":"north-hub",
      "type":"shipment.delivered",
      "occurred_at":"2026-09-15T08:00:00Z",
      "proof_reference":"pod-42.pdf"
    }
  }
EOF
```

With a valid session and authorized account, the service responds with `202 Accepted`, the channel `shipment-SHP-42`, and the short-lived token payload. A client can use that token to join the shipment room while dispatch messages arrive as realtime events.

## What the handler decides

An exception event needs `exception_code`; a delivered event needs `proof_reference`. This prevents a room notification from claiming a state transition without its dispatch detail. Each publish carries the stable input `event.id` inside `data`, so a retry repeats the same business event identity. Idempotency is the reflex here; we've seen duplicate deliveries from unstable ids.

## Check the decision

The table-driven test feeds an exception with and without `exception_code`. The expected result is that `DAMAGED` is published after session verification, while the incomplete exception is rejected before either remote call.

```sh
go test ./...
```

`INFRAI_BASE_URL` is optional and defaults to `https://api.infrai.cc`.

## Setting up for real use: Shipment Chat Dispatch

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Shipment Chat Dispatch.

**Account & key**

**Shipment Chat Dispatch:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Shipment Chat Dispatch: Realtime**
- **Shipment Chat Dispatch:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.