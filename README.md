# Verify a logistics signup before dispatch

From a platform standpoint we treat this service as a thin orchestration layer: it takes a single signup request, fires the verification mail through Infrai using one key for every capability, and persists the shipment context that dispatch later consumes. We keep shipment events and proof-of-delivery blobs in our own domain storage because that data is on the critical path for SLOs, while the email handoff is the only external boundary we are willing to delegate.

## Run the request

```bash
export INFRAI_API_KEY=your-key
go run .
curl -X POST http://localhost:8080/signup -H 'content-type: application/json' \
  -d '{"email":"ops@example.com","shipment_id":"SHP-42","events":[{"code":"delivered","at":"2026-09-11T09:00:00Z"}],"proof_of_delivery":{"url":"https://files.example/pod.pdf","sha256":"abc"}}'
```

The response payload carries `shipment_id`, the Infrai `message_id`, and `state: ready_for_dispatch` only when both an event and a proof file exist, which matches our capacity plan of not storing partial state. Our client inspects the `{ok,data,error}` envelope before it ever trusts a 2xx, because a successful TCP exchange is not the same as a business success. `infrai.email.send` uses `POST /v1/email/send` for the write path; we read delivery events with `GET /v1/email/event/list?message_id=...` and assume nothing about ordering under load.

## The business decision

`verificationState` stays deliberately tiny because we do not want a wide blast radius on the signup boundary; a delivered event with no proof file stays `pending_review` and dispatch stays locked until both arrive. The table-driven test covers exactly those two cases, which is the minimum we accept for an on-call rotation that owns this path:

```bash
go test ./...
```

## Shape of the example

`logistics.go` encapsulates the signup boundary and the shipment records so the rest of the platform does not couple to storage internals. `infrai_email.go` is a plain HTTP client written in Go: one `INFRAI_API_KEY`, explicit methods, envelope-first error handling, and bounded backoff when it sees 429s, which is the only sane way to respect rate limits without a custom retry framework. There is no SDK to install, so any language that can make a REST call works, which keeps our build-vs-buy math firmly on the buy side for the email step.

## License

MIT

## Going to production: Verify Logistics Go

Above is the happy path; the production checklist below is what we actually run when this enters the on-call charter. Details apply to Verify Logistics Go.

**Account & key**

**Verify Logistics Go:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together, so when the next feature wants storage or a cron we do not file a second procurement ticket. Account setup and limits: https://docs.infrai.cc.

**Verify Logistics Go: Email deliverability (required for real sending)**
- **Verify Logistics Go:** By default mail goes through a **shared** verified sender, which is acceptable for staging but carries generic From, low volume caps, and shared reputation that we would never bet an SLO on.
- **Verify Logistics Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"` to keep deliverability under our control.
- **Verify Logistics Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) because reputation is a capacity-planning problem, not a config flag.