# message_broker

Shared library implementing the transactional outbox/inbox pattern. It is not a running service. Its own migration table is `outbox_schema_migrations`; `CreateTable` runs it on cold start of the bookings and authentication HTTP Lambdas.

## Outbox flow

1. **Write**: A command handler writes domain data and an outbox record atomically in one DB transaction.
2. **Relay** (`OUTBOX_MODE`):
   - `polling` (production Lambda, Helm default): a scheduled job reads unsent rows (`emitted_at IS NULL`) from the `outbox` table. In production this runs every 10 minutes, so an event can take up to 10 minutes to arrive.
   - `replication` (the default when `OUTBOX_MODE` is unset in bookings and authentication; `--set config.outboxMode=replication` tries it in dev): tails the outbox table with PostgreSQL logical replication (`pgoutput`) through a replication slot. This needs `wal_level=logical`; the Frostmoln provider cannot apply parameter groups, so production uses `polling`. Polling vs replication is not decided yet; both stay supported.
3. **Publish**: the relay publishes the event to an SNS topic (LocalStack locally, AWS in production).
4. **Consume**: bookings, journal and event_ledger consume SQS queues subscribed to the topics. Consumers are idempotent (an `inbox` table, `ON CONFLICT DO NOTHING`).

```
bookings / authentication
  -> [outbox table]
  -> poller (Lambda, scheduled) or pg logical replication -> SNS topic
                                                  |
                                            SQS queue(s)
                                                  |
                                  bookings (users) / journal / event_ledger consumers
```

Tests: testify + testcontainers. CI builds and tests this module before the services.
