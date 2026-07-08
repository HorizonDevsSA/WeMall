# EcoCash Service — GraphQL Operations

GraphQL operations for the EcoCash payment integration.  
All operations are sent to the API gateway at `POST /graphql`.  
Operations marked **BUYER** require a buyer JWT in `Authorization: Bearer <token>`.  
Operations marked **SELLER** require a seller JWT.  
Operations marked **ADMIN** require an admin JWT.

---

## Operations

| File | Type | Role | Description |
|---|---|---|---|
| `ecocash_charge_mutation.graphql` | Mutation | BUYER | Charge customer EcoCash wallet (MER) |
| `ecocash_lookup_query.graphql` | Query | BUYER | Re-confirm transaction status from EcoCash |
| `ecocash_get_transaction_query.graphql` | Query | BUYER | Fetch a transaction by internal ID |
| `ecocash_transactions_by_order_query.graphql` | Query | BUYER | All transactions for an order |
| `ecocash_refund_mutation.graphql` | Mutation | BUYER | Refund a completed charge (REF) |
| `ecocash_reverse_mutation.graphql` | Mutation | BUYER | Reverse a pending charge (REV) |
| `ecocash_request_payout_mutation.graphql` | Mutation | SELLER | Queue a manual seller payout |
| `ecocash_get_payout_query.graphql` | Query | SELLER | Fetch a payout record |
| `ecocash_update_payout_status_mutation.graphql` | Mutation | ADMIN | Mark payout PAID/FAILED after settlement |

---

## Typical buyer checkout flow

```
1. checkout(input: ...) → Order { id, total, currency }
2. ecocashCharge(orderId, msisdn, amountCents, currency)
      → EcoCashTransaction { status: PENDING, clientCorrelator }
3. [customer completes USSD PIN on their phone]
4. EcoCash POSTs to notifyUrl webhook → service calls LookupTransaction
5. NATS publishes wemall.payment.succeeded or wemall.payment.failed
6. Poll ecocashLookup(orderId, clientCorrelator) if no webhook arrives within 60 s
```

## Refund / reversal flow

```
# Transaction is still PENDING (customer never entered PIN / timed out):
ecocashReverse(originalTxnId, reason) → REV

# Transaction is SUCCESS (settled):
ecocashRefund(originalTxnId, amountCents?, reason) → REF
```

## Transaction statuses

| Status | Meaning |
|---|---|
| `PENDING` | Waiting for customer USSD PIN or EcoCash confirmation |
| `SUCCESS` | Charge settled, funds received |
| `FAILED` | Declined — see `ecocashStatusMsg` for reason |
| `REFUNDED` | Full or partial refund processed |
| `REVERSED` | Pre-settlement reversal completed |
| `MANUAL_REVIEW` | Transient error — in ops queue for investigation |

## EcoCash sandbox PIN test matrix

| PIN | Scenario | Expected `status` |
|---|---|---|
| `0000` | Success | `SUCCESS` |
| `1111` | Insufficient funds | `FAILED` |
| `2222` | Invalid PIN | `FAILED` |
| `9999` | Limit exceeded | `FAILED` |

## Notes

- `amountCents` is always in the **smallest currency unit** — cents for USD, equivalent for ZWG.  
  Example: `$10.50 USD` = `1050`.
- `msisdn` accepts `07XXXXXXXX`, `263XXXXXXXXX`, or `2637XXXXXXXX` — normalised to E.164 internally.
  Stored and returned as a masked value (`263773***456`) for PII compliance.
- Payouts are **manual_bulk only** — EcoCash's EIP is a collections API with no B2C disbursement endpoint.
  The `ecocashRequestPayout` mutation queues a record; the finance team executes the transfer externally.
