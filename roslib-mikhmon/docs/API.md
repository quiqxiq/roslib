# roslib-mikhmon HTTP API

REST + SSE service di atas paket `mikrotik/*` dan `workflows/`. Entry point:
`cmd/server/main.go`. Layer ini di-bind ke `127.0.0.1:8080` secara default
(local-only — auth ditunda).

## Response envelope

Semua endpoint pakai bentuk envelope konsisten:

```json
// Success single
{ "data": { "id": "*1", "name": "user01" } }

// Success list
{ "data": [ ... ], "meta": { "count": 47 } }

// Error
{ "error": { "code": "NOT_FOUND", "message": "...", "path": "/api/v1/..." } }
```

Error `code` stabil — UI bisa branch dengan aman:

| Code | HTTP | Trigger |
|---|---|---|
| `NOT_FOUND` | 404 | `mikrotik.ErrNotFound` |
| `INVALID_ARGUMENT` | 400 | `mikrotik.ErrInvalidArgument` |
| `AMBIGUOUS` | 409 | `mikrotik.ErrAmbiguous` |
| `VALIDATION` | 400 | `c.ShouldBindJSON` gagal (detail di `error.details`) |
| `TIMEOUT` | 504 | `context.DeadlineExceeded` |
| `CANCELED` | 499 | `context.Canceled` |
| `INTERNAL` | 500 | uncategorized + panic recovery |
| `METHOD_NOT_ALLOWED` | 405 | wrong HTTP method untuk path |

Setiap response include header `X-Request-ID` (16-char hex) untuk korelasi
ke server log (`request_id` field).

## REST endpoints

Prefix: **`/api/v1`**. Health probe: `GET /healthz`.

### Hotspot — User

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/hotspot/users` | — | `[HotspotUser]` (query `?profile=`, `?comment=` post-filter) |
| GET | `/hotspot/users/count` | — | `{count: N}` |
| GET | `/hotspot/users/by-name/:name` | — | `HotspotUser` |
| GET | `/hotspot/users/:id` | — | `HotspotUser` |
| POST | `/hotspot/users` | `HotspotUserCreateRequest` | 201 + `{id: "*N"}` |
| PUT | `/hotspot/users/:id` | `HotspotUserUpdateRequest` | 204 |
| PATCH | `/hotspot/users/:id/disabled` | `{value: bool}` | 204 |
| PATCH | `/hotspot/users/:id/expiry` | `{value: "jan/06/2099 14:32:01"}` | 204 |
| PATCH | `/hotspot/users/:id/mac` | `{value: "AA:BB:CC:..."}` | 204 |
| POST | `/hotspot/users/:id/reset-counters` | — | 204 |
| POST | `/hotspot/users/:id/reset-usage` | — | 204 |
| DELETE | `/hotspot/users/:id` | — | 204 (cascade: script + scheduler + user) |
| POST | `/hotspot/users/bulk-delete` | `{ids: ["*1","*2"]}` | `{succeeded: [...], failed: {id: msg}}` |

### Hotspot — Profile / Active / Server / Host / Cookie / Binding

| Path | Methods |
|---|---|
| `/hotspot/profiles` | GET (`?cache=true&ttl=60s`), POST |
| `/hotspot/profiles/by-name/:name` | GET |
| `/hotspot/profiles/:id` | GET, PUT, DELETE (body opsional `{name}` untuk cascade scheduler) |
| `/hotspot/active` | GET (`?server=`), GET `/count` |
| `/hotspot/active/:id` | GET, DELETE (kick — workflows.KickActive) |
| `/hotspot/servers` | GET (`?cache=true`) |
| `/hotspot/hosts` | GET, DELETE `/:id` |
| `/hotspot/cookies` | GET (`?user=`), GET `/count`, DELETE `/:id` |
| `/hotspot/bindings` | GET, GET `/count`, GET `/:id`, DELETE `/:id` (cascade) |
| `/hotspot/bindings/:id/type` | PATCH `{type: "regular"\|"bypassed"\|"blocked"}` |
| `/hotspot/bindings/:id/disabled` | PATCH `{value: bool}` |

### System

| Path | Methods |
|---|---|
| `/system/identity` | GET (`?cache=true`) |
| `/system/resource` | GET |
| `/system/routerboard` | GET |
| `/system/clock` | GET |
| `/system/reboot` | POST (destructive!) |
| `/system/shutdown` | POST (destructive!) |
| `/system/logging/by-prefix/:prefix` | GET → `{count: N}` |
| `/system/logging/hotspot-disk` | POST `{prefix: "->"}` |
| `/system/scripts` | GET (`?name=`, `?comment=`, `?owner=`, `?source=`), POST |
| `/system/scripts/:id` | GET, PUT, DELETE |
| `/system/schedulers` | GET (`?name=`), GET `/count`, POST |
| `/system/schedulers/:id` | PUT, DELETE |

### Network

| Path | Methods |
|---|---|
| `/network/interfaces` | GET |
| `/network/queues` | GET (`?name=`, `?dynamic=false`), DELETE `/:id` |
| `/network/pools` | GET (`?cache=true`) |
| `/network/arp` | GET `?mac=` (wajib), DELETE `/:id` |
| `/network/dhcp-leases` | GET (`?mac=`), GET `/count`, DELETE `/:id` |

### PPP

| Path | Methods |
|---|---|
| `/ppp/secrets` | GET, POST |
| `/ppp/secrets/by-name/:name` | GET |
| `/ppp/secrets/:id` | PUT, DELETE |
| `/ppp/secrets/:id/disabled` | PATCH `{value: bool}` |
| `/ppp/profiles` | GET |
| `/ppp/profiles/by-name/:name` | GET |
| `/ppp/active` | GET, DELETE `/:id` |

### Log

| Path | Methods |
|---|---|
| `/log` | GET (`?topics=hotspot,info,debug`) |

---

## SSE endpoints (Server-Sent Events)

Response `Content-Type: text/event-stream`. Setiap event:

```
event: <type>
data: <json>
                  ← blank line separator
```

Keepalive `: keepalive` di-emit tiap **15 detik** kalau tidak ada event —
cegah reverse proxy timeout.

### Broker fan-out

1 stream router → N SSE client subscriber. Buka saat client pertama
connect, tutup saat last client disconnect (refcount). Hemat connection
ke router + listener registration.

### Endpoints

| Path | Backend | Event type | Payload contoh |
|---|---|---|---|
| `GET /stream/hotspot/active` | `hot.ActiveStream` (Follow) | `change` | `{id, user, address, mac_address, server, uptime, bytes_in, bytes_out, dead}` |
| `GET /stream/hotspot/active?mode=follow-only` | `hot.ActiveStreamFollowOnly` | `change` | same |
| `GET /stream/ppp/active` | `ppp.ActiveStream` | `change` | `{id, name, service, caller_id, address, uptime, dead}` |
| `GET /stream/log?topics=hotspot,info` | `syslog.LogStream` (FollowOnly) | `log` | `{id, time, topics, message}` |
| `GET /stream/system/resource?interval=1s` | `sys.MonitorResource` | `resource` | `SystemResourceResponse` |
| `GET /stream/network/interfaces/:name/traffic` | `net.InterfaceTrafficStream` | `traffic` | `{name, rx_bits_per_sec, tx_bits_per_sec, rx_packets_per_sec, tx_packets_per_sec}` |
| `GET /stream/network/interfaces/stats?interval=2s` | `net.InterfaceStatsStream` | `stats` | `{id, name, type, rx_byte, tx_byte, running, disabled, ...}` |
| `GET /stream/network/queues/stats?interval=1s` | `net.QueueStatsStream` | `stats` | `{id, name, target, bytes, packets, rate, max_limit}` |

### Pattern client (JavaScript)

```js
const es = new EventSource('/api/v1/stream/system/resource?interval=2s');
es.addEventListener('resource', (e) => {
  const data = JSON.parse(e.data);
  console.log('cpu=' + data.cpu_load + ' uptime=' + data.uptime);
});
es.onerror = () => { es.close(); };
```

### Pattern client (curl)

```bash
curl -N http://127.0.0.1:8080/api/v1/stream/network/interfaces/ether1/traffic
# event: traffic
# data: {"name":"ether1","rx_bits_per_sec":1008,"tx_bits_per_sec":1448,...}
```

---

## Konfigurasi (.env)

```bash
# Router (wajib)
ROSLIB_ROUTER_ADDRESS=192.168.88.1:8728
ROSLIB_ROUTER_USERNAME=admin
ROSLIB_ROUTER_PASSWORD=secret

# HTTP server
HTTP_BIND=127.0.0.1:8080
HTTP_READ_TIMEOUT=10s
HTTP_IDLE_TIMEOUT=60s
HTTP_SHUTDOWN_GRACE=10s
CORS_ALLOWED_ORIGINS=*
```

---

## Middleware chain

1. `Recovery` — panic → 500 envelope + log stack trace.
2. `RequestID` — set `X-Request-ID` (echoed di response header + log).
3. `Logger` — log per request: method, path, status, latency, request_id.
4. `CORS` — permissive default; production override via env.

---

## Quickstart

```bash
cp .env.example .env  # edit ROSLIB_ROUTER_*
go run ./cmd/server

# Terminal lain:
curl -s http://127.0.0.1:8080/api/v1/system/identity | jq
# {"data":{"name":"MikroTik"}}

curl -s http://127.0.0.1:8080/healthz
# {"data":{"status":"ok"}}
```

Graceful shutdown: kirim SIGINT (Ctrl+C) atau SIGTERM. Server akan:
1. Stop accept connection baru.
2. Tunggu in-flight request selesai (max `HTTP_SHUTDOWN_GRACE`).
3. `mgr.CloseAll()` → tutup roslib connection.
4. `influxCli.Close()` (kalau enabled).
