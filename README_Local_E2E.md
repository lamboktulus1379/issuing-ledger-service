# Enterprise Ledger Service: Local End-to-End Execution Guide

This document is the **centralized source of truth** for running, tracing, and debugging the entire Ledger System locally. 

By following this guide step-by-step, you will observe the complete lifecycle of a transaction across every infrastructure component:
`HTTP Request -> Go API -> Redis (Idempotency) -> MariaDB (State) -> Debezium (CDC) -> Kafka (Event Broadcast) -> Grafana/Tempo (Observability)`.


---
## Step 1: Start the Entire Infrastructure Ecosystem

The root `docker-compose.yml` is the Compose orchestrator. It includes the state, CDC, observability, and app modules and creates the shared `ledger_network`.

Open a terminal in the repository root and run:
```bash
docker compose up -d
```

**Verify everything is running:**
```bash
docker ps
```
*You should see the MariaDB, MySQL, Redis, Kafka, Kafka Connect, Prometheus, Grafana, and OpenTelemetry Collector containers.*

After changing Collector or Tempo configuration files, force those services to reload their bind-mounted configuration:

```bash
docker compose up -d --force-recreate tempo otelcollector app
```

Tempo must log listeners on `0.0.0.0:4317` and `0.0.0.0:4318`. If it logs `127.0.0.1`, the Collector container cannot reach Tempo and Grafana's Tempo service dropdown remains empty.

To start only the state stores:

```bash
docker compose -f infrastructure/docker/docker-compose-state.yml up -d
```

The modules use the shared `ledger_network`, so service names remain consistent when the root orchestrator is used.

### Docker Database Hostname Troubleshooting

When the Go application runs inside Docker, do not configure the database host as `localhost` or `127.0.0.1`. Inside the app container, `localhost` points back to the app container itself, not to MariaDB. On IPv6-enabled systems it may resolve to `[::1]`, producing errors such as:

```text
dial tcp [::1]:3306: connect: connection refused
```

Use the Compose service name instead:

```text
MYSQL_HOST=mariadb
MYSQL_PORT=3306
MYSQL_DATABASE=ledger
MYSQL_USER=root
MYSQL_PASSWORD=rootpassword
```

The application Compose module already supplies these values. Check the rendered environment without printing secrets:

```bash
docker compose config | grep -E 'MYSQL_HOST|MYSQL_PORT|MYSQL_DATABASE|MYSQL_USER'
```

Check that the app and MariaDB share `ledger_network`:

```bash
docker network inspect ledger_network
```

From the app container, verify DNS and TCP reachability:

```bash
docker compose exec app getent hosts mariadb
docker compose exec app sh -c 'echo > /dev/tcp/mariadb/3306'
```

If the second command is unavailable in the image, use the application logs and MariaDB logs instead:

```bash
docker logs --tail 100 ledger_db
docker logs --tail 100 <app-container-name>
```

If a container exits with code 137, Docker killed it, commonly because of an out-of-memory condition. Check resource usage and the container state:

```bash
docker stats --no-stream
docker inspect <container-name> --format '{{.State.OOMKilled}} {{.State.ExitCode}}'
```

The same rule applies to Redis. An error such as `dial tcp [::1]:6379: connect: connection refused` means the app is using the `localhost` default from `config.json`. The Docker app must use the Redis service name and password:

```text
REDIS_HOST=ledger-redis
REDIS_PORT=6379
REDIS_USERNAME=default
REDIS_PASSWORD=your_redis_password
```

Verify Redis DNS and authentication from the app container:

```bash
docker compose exec app getent hosts ledger-redis
docker compose exec redis redis-cli -a your_redis_password PING
```

The expected Redis response is `PONG`. The authorization endpoint depends on Redis for idempotency locks, so the application now fails startup when Redis is unavailable instead of accepting requests that return HTTP 500.

Before testing authorization, apply the ledger migrations so `accounts`, `ledger_entries`, `ledger_transactions`, and `outbox_events` exist. The outbox migration is `infrastructure/persistence/migrations/0002_20260925_create_outbox_events.up.sql`. A missing `outbox_events` table will cause the transaction to roll back after the account locks and balance updates.

---

## Step 2: Verify Kafka Connect Health

Kafka Connect is a JVM application and takes 30-60 seconds to fully boot and expose its REST API. You must wait for it to be ready before proceeding.

Run this command periodically until it returns a `200 OK` (or valid JSON showing version info):
```bash
curl -s http://localhost:8083/
```
**Expected Output:**
```json
{"version":"2.4.0","commit":"...","kafka_cluster_id":"..."}
```

---

## Step 3: Register the Debezium CDC Connector

Now we tell Debezium to attach to MariaDB, read its binary log (`binlog`), and listen for inserts on the `outbox_events` table.

Run this exact command:
```bash
curl -X POST http://localhost:8083/connectors   -H "Content-Type: application/json"   -d @infrastructure/cdc/outbox-connector.json
```

**Verify the connector is registered successfully:**
```bash
curl -s http://localhost:8083/connectors/issuing-outbox-connector/status | jq
```
*Expected output should show `"state": "RUNNING"` for both the connector and the tasks.*

---

## Step 4: Boot the Go Ledger Service

Open a **NEW terminal tab**. We will boot the Go application and tell it to use the CDC strategy (bypassing the Go polling worker) and to emit OpenTelemetry data to our local Collector.

```bash
export OUTBOX_STRATEGY=cdc
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
go run cmd/api/main.go
```
*Leave this tab open. The server should log that it is listening on port `8080`.*

---

## Step 5: Fire the Core HTTP Request

Open a **NEW terminal tab**. We will simulate a user making a payment authorization. We pass an `Idempotency-Key` to ensure the exact same request isn't processed twice.

```bash
curl -X POST http://localhost:10001/api/authorize   -H "Content-Type: application/json" -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3OTAzNTU0OTMsImlzcyI6IjIiLCJ1c2VyX25hbWUiOiJqb2huZG9lIn0.cipxCF2qOiw8PKVZWKthoTaclIwJiSDIgdghC47LtrA"  -H "Idempotency-Key: req-uuid-5555-9999"   -d '{
  "transaction_id": "txn-cdc-0001",
  "source_account_id": "acc_source_123",
  "destination_account_id": "acc_destination_456",
  "amount": 2550,
    "currency": "USD"
  }' | jq
```

`amount` is expressed in integer minor units, so `2550` means USD 25.50. This endpoint requires both account IDs because it performs a double-entry transfer; `account_id` and `merchant` are not recognized fields.

**Expected Output:**
```json
{
  "status": "success",
  "message": "Payment authorized",
  "transaction_id": "txn_..."
}
```

---

## Step 6: Verify the Transactional Outbox in MariaDB

Let's prove the Go application correctly inserted the state and the outbox event in the same atomic database transaction.

```bash
docker exec -it mariadb mysql -u root -prootpassword ledger_db -e "SELECT * FROM outbox_events ORDER BY created_at DESC LIMIT 1;"
```
*You should see the raw event row waiting in the database.*

---

## Step 7: Verify Event Streaming in Kafka (The Magic Moment)

Because Debezium is running, it should have instantly captured that insert and routed it to Kafka. Let's prove it by tailing the target Kafka topic.

```bash
docker exec -it kafka /opt/kafka/bin/kafka-console-consumer.sh   --bootstrap-server localhost:9092   --topic outbox.event.TransactionAuthorized   --from-beginning   --max-messages 1
```
*You should immediately see the JSON payload emitted by Debezium, proving your CDC architecture works perfectly.*

---

## Step 8: Trace the 50ms SLA in Grafana

We need to prove the transaction met our strict performance requirements. 

1. Open your web browser and navigate to: **http://localhost:3000**
2. **Login:** `admin` / `admin` (or skip password setup).
3. Go to the **Explore** tab (compass icon on the left).
4. Select **Tempo** from the data source dropdown at the top.
5. Go to the **Search** tab inside Tempo, select `ledger-service` from the Service Name dropdown, and click **Run Query**. Generate a fresh request after the services are recreated; old failed exports are not retroactively stored.
6. Click on the Trace ID for your recent request.

**What you will see:**
A complete waterfall graph proving exactly how many milliseconds were spent:
1. Handling the HTTP Request.
2. Acquiring the Redis CAS lock (`Idempotency.AcquireLock`).
3. Executing the MariaDB `FOR UPDATE` transaction (`Usecase.AuthorizePayment`).
4. Releasing the lock (`Idempotency.Complete`).

---

## Step 9: Clean Teardown

Once you are done learning and debugging, clean up the system resources.

1. Go to the terminal running the Go application and press `Ctrl+C` if you ran it on the host.
2. From the repository root, stop the complete Compose project:
```bash
docker compose down --remove-orphans
```

To also delete named volumes and reset local state:

```bash
docker compose down --remove-orphans --volumes
```

If you started only the state module, use:

```bash
docker compose -f infrastructure/docker/docker-compose-state.yml down --remove-orphans
```

The old multi-file CDC/observability command is obsolete. Use the root `docker compose` commands above.
