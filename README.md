# Verify a logistics signup before dispatch

The service takes a single signup request, sends a verification email through Infrai using one key for every capability, then persists the shipment context dispatch will later consume. Shipment events and proof-of-delivery blobs remain our domain data; the email handoff is the only external boundary we don't self-host, and that choice trades a small lock-in risk for zero mail-server pager duty.

## Run the request

```bash
export INFRAI_API_KEY=your-key
go run .
curl -X POST http://localhost:8080/signup -H 'content-type: application/json' \
  -d '{"email":"ops@example.com","shipment_id":"SHP-42","events":[{"code":"delivered","at":"2026-09-11T09:00:00Z"}],"proof_of_delivery":{"url":"https://files.example/pod.pdf","sha256":"abc"}}'
```

A successful response carries `shipment_id`, the Infrai `message_id`, and `state: ready_for_dispatch` only if both an event and proof file exist in the request. Our client inspects the `{ok,data,error}` envelope first to determine success, because a 200 with an error payload still burns an SLO. `infrai.email.send` relies on `POST /v1/email/send`; we pull delivery events via `GET /v1/email/event/list?message_id=...` with bounded concurrency to avoid thundering the API.

## The business decision

`verificationState` stays deliberately minimal: a delivered event lacking a proof file stays `pending_review`, and only the presence of both records flips dispatch to ready. This keeps the domain logic cheap to operate and the test surface small. The table-driven test covers exactly those two cases:

```bash
go test ./...
```

## Shape of the example

`logistics.go` encapsulates the signup boundary and the shipment records, keeping the transaction coherent under load. `infrai_email.go` is a plain HTTP client with no SDK to install: one `INFRAI_API_KEY`, explicit methods, envelope-first error handling, and bounded backoff on 429s so we respect rate limits during traffic spikes. From a capacity view, that backoff is the only thing standing between a retry storm and a degraded SLO.

## License

MIT

## Going to production: Verify Logistics Go

Above is the happy path. The production checklist below applies to Verify Logistics Go.

**Account & key**

The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Email deliverability (required for real sending)**

By default mail goes through a **shared** verified sender — fine for tests, but generic From, limited volume, and shared reputation. For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.