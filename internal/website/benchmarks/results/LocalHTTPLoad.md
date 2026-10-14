# Local website HTTP load samples

Captured 2026-10-14 on macOS 26, Apple M4 Pro. The server was the locally built
`cmd/cully-web` binary using `dist/site`, produced by `node site/build.mjs` (the
same build step used by `Dockerfile.website`), from patch revision `513a901`,
with a fresh temporary data directory. Both routes returned HTTP 200 before
load was run.

```sh
go build -o /tmp/cully-web ./cmd/cully-web
node site/build.mjs
CULLY_WEB_SITE_DIR="$PWD/dist/site" CULLY_WEB_DATA_DIR="/tmp/cully-web-data" \
  CULLY_WEB_LISTEN=127.0.0.1:18089 /tmp/cully-web
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:18089/
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:18089/install.sh
ab -n 10000 -c 32 http://127.0.0.1:18089/
ab -n 10000 -c 32 http://127.0.0.1:18089/install.sh
```

Three sequential ApacheBench samples were run per route. Each had 10,000
completed requests, 0 failed requests, and 0 non-2xx responses.

| Route | Sample | Requests/s | p50 | p95 | p99 | Max |
|---|---:|---:|---:|---:|---:|---:|
| `/` | 1 | 23,246.41 | 1 ms | 2 ms | 3 ms | 5 ms |
| `/` | 2 | 24,740.41 | 1 ms | 2 ms | 2 ms | 4 ms |
| `/` | 3 | 22,495.97 | 1 ms | 2 ms | 3 ms | 38 ms |
| `/install.sh` | 1 | 27,594.66 | 1 ms | 1 ms | 2 ms | 3 ms |
| `/install.sh` | 2 | 23,902.17 | 1 ms | 1 ms | 3 ms | 48 ms |
| `/install.sh` | 3 | 28,494.41 | 1 ms | 1 ms | 2 ms | 4 ms |

These local results depend on hardware and host load. The container build copies
`install.sh` into the built site before the Go server serves it. Starting the
server directly against `site/` omits that build step and correctly produces a
404, so the load test uses `dist/site`.
