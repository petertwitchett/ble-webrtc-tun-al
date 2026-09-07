# BLE WebRTC Tunnel — Master System Architecture & Knowledge Base

> **Location:** `research/knowledge/PROJECT_KNOWLEDGE_BASE.md`  
> **Mandatory Maintenance Rule:** Whenever any code, architecture, configuration, API, or protocol behavior in this project is added, modified, or refactored, this document **must** be updated synchronously to reflect the current ground truth.

---

## 1. Executive Summary & Purpose

The **BLE WebRTC Tunnel** (`ble-webrtc-tun-al`) is a high-performance, covert anti-censorship tunneling system and VPN engineered specifically to bypass extreme state-level internet blocking and Deep Packet Inspection (DPI) in Iran.

### The Threat Model & Censorship Mechanics
During severe network crackdowns and national internet blackouts in Iran:
- **International Gateway Severing:** Traffic to foreign IP addresses, standard VPN protocols (OpenVPN, WireGuard, IPsec), Shadowsocks, and common TLS/WebSocket proxies (V2Ray/VMess/VLESS/Trojan) is actively throttled or outright severed at the national gateway (TIC).
- **Whitelisted Intranet (National Information Network / NIN):** Only domestic Iranian IP ranges, government websites, domestic banking portals, and domestic messenger applications are permitted to transmit data.
- **DPI & SNI Filtering:** State firewalls employ Deep Packet Inspection to inspect TLS ClientHello SNI headers, detect non-conforming protocol handshakes, and throttle UDP streams that fail to match recognized audio/video codecs.

### The Infiltration Vector: Bale Messenger
This system exploits the state-approved domestic messaging application **"Bale" (بله - `bale.ai` / `ble.ir`)**:
1. **Unblocked Infrastructure:** Bale's messaging WebSocket servers (`*.bale.ai`) and WebRTC Selective Forwarding Units (SFUs) (`meet.bale.ai`) are hosted within Iranian domestic datacenters and operate on the state's highest-tier whitelist, remaining completely unblocked and high-speed even during complete international blackouts.
2. **Covert WebRTC Audio Channel:** Rather than establishing a raw, suspicious proxy connection, this project establishes legitimate one-to-one voice/video calls between two Bale accounts over Bale's LiveKit SFU servers.
3. **Opus RTP Camouflage:** Tunneled IP and TCP/UDP packets are disguised as 48kHz Opus audio RTP packets, complete with valid RTP sequence numbers, logical timestamp increments, and Opus DTX comfort-noise frames. To the state DPI and Bale's SFU, the connection is indistinguishable from an ordinary human voice conversation.
4. **QUIC Over Opus:** On top of the audio RTP datagram stream, an encrypted QUIC connection (`quic-go`) runs natively, providing loss recovery, congestion control, and stream multiplexing.

---

## 2. End-to-End System Workflow

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                                 RESTRICTED CLIENT (IRAN)                                │
│                                                                                        │
│  ┌─────────────────────────┐          ┌─────────────────────────────────────────────┐  │
│  │   User Application      │          │         Client Dashboard / Web UI           │  │
│  │  (Browser / Telegram)   │          │          (http://localhost:6681)            │  │
│  └────────────┬────────────┘          └──────────────────────┬──────────────────────┘  │
│               │                                              │                         │
│               │ SOCKS5 (:1080) / HTTP (:8080)                │ Phone Login + OTP       │
│               ▼                                              ▼                         │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │ RoutingEngine & BypassEngine                                                     │  │
│  │   - Iranian APNIC CIDRs → Direct Local Route                                     │  │
│  │   - Blocked International Targets → QUIC Artery Pool                             │  │
│  └────────────────────────────────────┬─────────────────────────────────────────────┘  │
│                                       │                                                │
│                                       ▼                                                │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │ Multi-Artery Virtual Pool (Orchestrator)                                         │  │
│  │   - Manages multiple client Bale accounts (Ch 0, Ch 1, ...)                      │  │
│  │   - Tracks RTT / Loss / State (ACTIVE, SHADOW, QUARANTINED, REVIVING)            │  │
│  └────────────────────────────────────┬─────────────────────────────────────────────┘  │
│                                       │                                                │
│                                       ▼                                                │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │ QUIC over Opus Layer (quicconn + rtpconn)                                        │  │
│  │   - QUIC datagrams wrapped into Opus RTP packets (Payload Type 111, 48kHz)       │  │
│  │   - XChaCha20-Poly1305 Obfuscation (dcconn)                                      │  │
│  │   - Comfort Noise Frames (0xF8 0xFF 0xFE) during silence                         │  │
│  └────────────────────────────────────┬─────────────────────────────────────────────┘  │
│                                       │                                                │
│               Bale Signaling          │ WebRTC Media Streams                           │
│               (WSS / Proto)           │ (SRTP / UDP)                                   │
└───────────────────────┬───────────────┴──────────────────────┬─────────────────────────┘
                        │                                      │
                        ▼                                      ▼
        ┌───────────────────────────────┐      ┌───────────────────────────────┐
        │       Bale Signaling          │      │     Bale LiveKit SFU          │
        │   (wss://api.bale.ai/...)     │      │       (meet.bale.ai)          │
        │  * Whitelisted Domestic IP    │      │  * Whitelisted Domestic IP    │
        └───────────────┬───────────────┘      └───────────────┬───────────────┘
                        │                                      │
                        │ Signaling Push                       │ Media Forwarding
                        │                                      │
┌───────────────────────┴──────────────────────────────────────┴─────────────────────────┐
│                                 UNRESTRICTED SERVER (ABROAD)                            │
│                                                                                        │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │ Server WebRTC / LiveKit Listener                                                 │  │
│  │   - Accepts incoming Bale calls from paired client accounts                      │  │
│  │   - Joins room, receives Opus audio track, strips RTP headers                    │  │
│  └────────────────────────────────────┬─────────────────────────────────────────────┘  │
│                                       │                                                │
│                                       ▼                                                │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │ QUIC Server (quicconn)                                                           │  │
│  │   - Terminates QUIC session over OpusPacketConn                                  │  │
│  │   - Demultiplexes streams                                                        │  │
│  └────────────────────────────────────┬─────────────────────────────────────────────┘  │
│                                       │                                                │
│                                       ▼                                                │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │ TCP Proxy Forwarder                                                              │  │
│  │   - Dials target host/port directly onto unrestricted foreign internet           │  │
│  └────────────────────────────────────┬─────────────────────────────────────────────┘  │
│                                       │                                                │
│                                       ▼                                                │
│                        ┌─────────────────────────────┐                                 │
│                        │     Open Free Internet      │                                 │
│                        │ (Google, YouTube, X, etc.)  │                                 │
│                        └─────────────────────────────┘                                 │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

### Operational User Phases
1. **Server Provisioning:**
   - Deploy `bin/server` to an unblocked host (Clever Cloud, VPS abroad, or Docker).
   - Access the Server Web UI (e.g. `http://server-ip:6680`).
   - Enter one or more Iranian phone numbers registered on Bale.
   - Receive SMS OTPs, verify them via the UI, acquiring Bale JWT access tokens. These accounts enter `SERVER` role in the server's SQLite database (`server.db`).
2. **Client Provisioning:**
   - Run `bin/client` locally on the restricted machine.
   - Access the Client Web UI at `http://localhost:6681`.
   - Enter separate Iranian phone numbers registered on Bale, receive SMS OTPs, and verify them. These accounts enter `CLIENT` role in `client.db`.
3. **Pairing:**
   - Link Client Account(s) to Server Account(s).
   - Auto-pairing or manual pairing creates `Pairing` entities tied to an `OwnerID`.
   - The client syncs pairings to the server database via the REST sync protocol (`/api/remote/sync/...`).
4. **Tunnel Initiation:**
   - User clicks **"Start Tunnel"** in the Client UI.
   - Client initializes the `TunnelManager`:
     - Opens local SOCKS5 proxy on `127.0.0.1:1080` and HTTP proxy on `127.0.0.1:8080`.
     - For each paired account, connects to Bale's signaling WebSocket, sends a call invitation, joins the LiveKit SFU room, binds Pion WebRTC to an Opus track, and establishes a QUIC connection.
     - The `Artery` orchestrator aggregates active channels into a resilient virtual connection pool.
     - Split-tunneling actively diverts domestic Iranian traffic directly while tunneling blocked international traffic.

---

## 3. Component Deep Dive

### 3.1. Bale Platform Protocol & Reverse Engineering (`internal/bale`)

Bale is an enterprise-grade messenger built on custom Protobuf RPC over WebSockets and gRPC-Web via Envoy proxies.

#### A. Client Emulation & Metadata Synchronization (`constants.go`, `extractor.go`)
Bale silently silences or disconnects clients that advertise outdated versions.
- **Client Constants:**
  - `app_version`: `"169491"` (corresponds to production release `web@5.5.1+169491`).
  - `browser_version`: `"151.0.0.0"`.
  - `web_api_key`: `"C28D46DC4C3A7A26564BFCC48B929086A95C93C98E789A19847BEE8627DE4E7D"`.
  - `bale_ws_url`: `"wss://next-ws.bale.ai/ws/"`.
  - `bale_grpc_base`: `"https://next-ws.bale.ai"`.
- **Dynamic JS Bundle Extraction:** `ExtractConstants` and `fetchLatestAppVersion` scrape `https://web.bale.ai/`, discover the active JavaScript bundles, and dynamically extract current build numbers and API keys. The scraper targets `SENTRY_RELEASE={id:"web@...+(\\d{5,7})"}`, `appversion:String("(\\d{5,7})")`, and `buildNumber:(\\d{5,7})`.
- **Persistence:** These constants are cached in the SQLite `settings` table and updated on startup and via `/api/bale/constants/sync`.

#### B. Phone Authentication & OTP (`auth.go`, `internal/api/bale_login.go`)
- **Start Auth:** `StartPhoneAuth(phone)` dispatches a gRPC-Web request to `/bale.auth.v1.Auth/StartPhoneAuth`.
  - **Required Headers:** Must include paired twins: `language: fa`, `mt_language: fa`, `app_version: 169491`, `mt_app_version: 169491`, `browser_type: 1`, `mt_browser_type: 1`, `browser_version: 151.0.0.0`, `mt_browser_version: 151.0.0.0`, `os_type: 4`, `mt_os_type: 4`, `session_id`, `mt_session_id`, `x-grpc-web: 1`, `content-type: application/grpc-web+proto`.
  - **Cookie Clarification:** `StartPhoneAuth` does NOT require an `access_token` cookie when correct version and paired headers are sent; attempting to inject stale or expired cookies causes Envoy to return `401 Unauthorized`.
- **Verify Auth:** `ValidateCode(txHash, code)` validates the 6-digit SMS code against `/bale.auth.v1.Auth/ValidateCode` and extracts the final user JWT token, `UserID`, and `AccessHash` from `Set-Cookie` or field 4 of the response body.

#### C. WebSocket Signaling & Call Negotiation (`client.go`)
- Connects to Bale's WSS signaling server (`wss://next-ws.bale.ai/ws/`).
- **Protobuf RPC Dispatch & Metadata Envelope:**
  Every RPC dispatched over WebSocket wraps the request in an envelope containing a 12-entry metadata message (Field 4):
  `app_version`, `browser_type`, `browser_version`, `os_type`, `session_id`, `mt_app_version`, `mt_browser_type`, `mt_browser_version`, `mt_os_type`, `mt_session_id`, `language: fa`, `mt_language: fa`.
  Omitting the `mt_*` twins or language fields causes the gateway to drop or reject RPCs.
- **Call Disconnection (`DiscardCall`):**
  - Schema: Field 1 (int64 `call_id`), Field 3 (int32 `reason: 3` = `CALLDISCARDREASON_HANGUP`), Field 4 (12-entry metadata), Field 5 (seq).
  - Teardown: `router.ForceEndCall` triggers session context cancellation and dispatches `DiscardCall` to prevent server accounts from remaining locked in `IN_CALL` state on Bale's backend.
- **SDP Exchange Over Bale Chat:** When direct SFU signaling requires SDP negotiation, offers and answers are sent as direct text messages disguised with headers:
  - `BLETUN:O:<base64-sdp>` — SDP Offer
  - `BLETUN:A:<base64-sdp>` — SDP Answer
- **Control Commands:**
  - `BLETUN:PING` / `BLETUN:PONG` — Heartbeats
  - `BLETUN:END` — Graceful session termination
  - `BLETUN:ENDCALL` & `BLETUN:ENDCALL_ACK` — Emergency call termination to break hanging server sessions.
- **Forensic Fingerprint Wiping:** Immediately after a signaling message is acknowledged, `client.go` issues `DeleteMessage` RPCs to Bale's servers to wipe the chat history, ensuring user chat logs remain empty.

---

### 3.2. Covert Media Transport: Opus & QUIC (`internal/rtpconn`, `internal/quicconn`, `internal/livekit`)

#### A. The 1:1 Track-to-QUIC Architectural Invariant
> **CRITICAL RULE:** Each QUIC connection **MUST** bind to exactly **ONE** WebRTC Opus audio track. Multi-path aggregation occurs **ABOVE** QUIC at the Artery Orchestrator level, **NEVER** below it.  
> *Rationale:* Striping datagrams from a single QUIC connection across multiple tracks induces sub-transport packet reordering. Reordering triggers QUIC's loss detection, causing the Congestion Window (CWND) to permanently collapse to minimum.

#### B. Pion WebRTC Audio Track & Zero-Sleep Pacing (`rtpconn.go`)
- Uses Pion WebRTC with `TrackLocalStaticSample` set to `webrtc.MimeTypeOpus` (Payload Type 111, clock rate 48,000Hz, stereo channels).
- **Logical Timestamp Increments:** Calls to `WriteFrame()` immediately invoke `WriteSample()` with a sample duration of 20ms. Pion increments the 48kHz RTP timestamp by exactly `960` samples per call. The SFU sees mathematically perfect timestamp progression regardless of the physical delivery rate.
- **Comfort Noise Injection:** An idle background loop checks `trackLastWrite`. If no user data was transmitted in the last 20ms, it injects a 3-byte Opus DTX comfort noise frame (`[]byte{0xF8, 0xFF, 0xFE}`) to keep the SFU room alive without causing buffer bloat.

#### C. QUIC Over RTP Bridge (`quicconn/opus_packet_conn.go`)
- Implements Go's `net.PacketConn` via `OpusPacketConn`.
- Provides static pseudo-addresses (`opus://local:0`, `opus://remote:0`).
- `WriteTo(datagram)` pushes pre-sized QUIC datagrams (≤ 1060 bytes) directly to `rtpconn.WriteFrame`.
- `ReadFrom(p)` blocks on incoming RTP packets from `rtpconn.ReadPacket` and returns them as QUIC datagrams.
- Uses mutual self-signed ECDSA certificates generated on-the-fly (`tls.Certificate`) with ALPN `ble-quic`.

#### D. Payload Obfuscation (`internal/dcconn/obfuscate.go`)
- Optional XChaCha20-Poly1305 authenticated encryption scrambles raw payloads before RTP transmission, preventing passive heuristic signatures.

---

### 3.3. Multi-Artery Virtual Pool & Orchestration (`internal/artery`, `cmd/client/orchestrator.go`)

The client groups multiple Bale accounts into parallel "arteries" within a virtual connection pool.

#### A. Artery Lifecycle State Machine
```
   (Connect) ──→ SHADOW ──→ ACTIVE ──→ QUARANTINED ──→ REVIVING ──→ ACTIVE
                             │              │
                             ▼              ▼
                           DEAD           DEAD  (Permanent token failure)
```
- **`ACTIVE`:** Operational; streams proxy connections.
- **`SHADOW`:** Connected and authenticated warm standby; handles background PINGs, ready for instantaneous failover.
- **`QUARANTINED`:** Demoted due to latency spikes or error thresholds; isolated from active routing while background health probes run.
- **`REVIVING`:** Background goroutine performs complete teardown and re-dial.
- **`DEAD`:** Terminal state (e.g. revoked token).

#### B. Telemetry & Load Scheduling
- **EWMA RTT:** Exponentially Weighted Moving Average smoothed RTT calculated per artery.
- **Sliding-Window Loss:** Tracks packet success/failure ratios over the last 100 samples.
- **Least-Loaded / Minimum RTT Scheduling:** Incoming proxy requests are routed to the artery with the lowest latency and fewest active streams.

#### C. Staggered Dynamic Session Refresh
Bale SFU rooms occasionally degrade after extended durations.
- **Quorum-Safety Guard:** Prevents simultaneous teardown. At least 2 active channels must exist before a channel is allowed to refresh (unless total channel count is 1).
- **Jittered Expiration:** Refreshes are scheduled at `60 minutes + RandomJitter(0..60 minutes)` to spread re-authentication events uniformly.

#### D. Fast Online Connection Policy & Dynamic Artery Scaling (`cmd/client/main.go`)
Historically, the client proxy listeners (SOCKS5/HTTP) did not open until *all* configured account pairs had finished connecting. In multi-pair setups (e.g. 6 pairs), this introduced noticeable startup latency and blocked user internet access if any single pair experienced slow signaling.
- **Instant First-Artery Proxy Activation:** As soon as pair #1 completes its WebRTC handshake and registers its independent QUIC session in the pool (`RegisterWithIndex`), `proxyOnce.Do` immediately starts the SOCKS5 (:10909) and HTTP (:9095) listeners, and `orchOnce.Do` launches the Artery Orchestrator. The user gains functional internet access within seconds.
- **Sequential Rate-Limit Safe Dialing:** Subsequent pairs continue connecting sequentially in the background. Sequential dialing avoids simultaneous ICE/DTLS negotiations that trigger Bale/LiveKit rate limits.
- **Zero-Downtime Dynamic Loop Expansion:** As each subsequent artery joins (`ActiveCount() > 1`), it is immediately registered into `tunnelPool`. The P2C (Power of Two Choices) + WRR scheduler automatically starts routing new outbound proxy streams across all active arteries. Bandwidth, throughput, and failover resilience scale smoothly and dynamically without tearing down or interrupting in-flight SOCKS streams.
- **Continuous Real-Time Telemetry:** Background metric collection starts immediately on channel #1, streaming per-artery health, byte counters, and EWMA latency to the client Web UI as each artery comes online.

---

### 3.4. Traffic Routing, DNS & Split-Tunneling (`internal/router`, `internal/dns`, `cmd/client/routing.go`)

To conserve tunnel bandwidth and provide low-latency access to domestic Iranian services, traffic is classified at the proxy layer before entering the tunnel.

#### A. High-Velocity Iranian CIDR Trie (`bypass.go`, `iran_cidrs.go`)
- Compiles thousands of APNIC-assigned Iranian IPv4 CIDR blocks into a fast in-memory subnet lookup trie.
- **Rule:** If the resolved target IP belongs to an Iranian CIDR or an admin-defined bypass domain, it bypasses the WebRTC tunnel and dials directly via the local network interface.

#### B. Protected Invariant: Bale Endpoints
- Domains matching `*.bale.ai` or `*.ble.ir` **never bypass the tunnel**, preventing accidental routing loops or broken signaling paths.

#### C. Application-Level DNS Resolver (`internal/dns/resolver.go`)
- Iranian ISPs frequently tamper with UDP 53 DNS queries (DNS poisoning/hijacking).
- `AppResolver` provides configurable primary and secondary upstream DNS resolvers (e.g. `1.1.1.1`, `8.8.8.8`).
- `InstallAppDNS` attaches this resolver to Bale and LiveKit HTTP/WebSocket dialers.

---

### 3.5. Database & Synchronization Architecture (`internal/db`, `internal/accounts`, `internal/sync`)

#### A. CGO-Free SQLite (`glebarez/sqlite`)
- Single-binary deployment without requiring C toolchains on deployment servers.
- Schemas:
  - `accounts`: Bale credentials, roles (`CLIENT`, `SERVER`), status, and metadata.
  - `pairings`: Client-to-server 1:1 mapping with `owner_id`.
  - `connection_logs`: Historical session statistics, bytes transferred, termination reasons.
  - `events`: Append-only log for event-sourced synchronization.
  - `settings`: Key-value configuration store.
  - `admin_users`: Web panel login credentials.

#### B. Event-Sourced Sync
- Modifications to accounts or pairings append immutable records to the `events` table.
- Clients periodically poll `/api/sync/pull` or push changes to remote servers via `/api/remote/sync/push`, ensuring seamless multi-panel coordination.

---

### 3.6. Embedded Management Dashboard (`web/`, `internal/webui`, `internal/api`)

- **Tech Stack:** React 18, TypeScript, Vite, Tailwind CSS, Lucide icons.
- **Single-Binary Embedding:** `npm run build` generates static assets in `web/dist`, which are synced to `internal/webui/dist` and compiled into the Go binary using `//go:embed dist/*`.
- **Key Features:**
  - Phone OTP Login Modal (direct Bale login).
  - Multi-Channel Artery Visualizer (live RTT, packet loss, bandwidth meters).
  - Account Manager & Auto-Pairing Tool.
  - DNS & Split-Tunneling Routing Configuration.
  - Web Terminal (`/api/terminal`) and Live WebSocket Log Streaming (`/api/logs/ws`).
  - Remote Server Sync & Database Backup/Restore.

---

## 4. Repository Structure

```
ble-webrtc-tun-al/
├── cmd/
│   ├── client/                  # Client entrypoint, proxy listeners, orchestrator, routing
│   │   ├── main.go              # Client bootstrap, CLI flags, tunnel lifecycle
│   │   ├── orchestrator.go      # Layer 1-3 recovery matrix, session refresh engine
│   │   └── routing.go           # SOCKS5/HTTP interception, DNS resolution, bypass routing
│   ├── server/                  # Server entrypoint, proxy forwarder, WebRTC listener
│   │   ├── main.go              # Server bootstrap, call responder, QUIC listener
│   │   └── proxy.go             # Bidirectional TCP streaming to open internet
│   └── migrate/                 # DB migration CLI (.env.tokens → SQLite)
├── internal/
│   ├── accounts/                # Account manager, pairing logic, health checks
│   ├── admin/                   # Legacy admin panel (deprecated)
│   ├── api/                     # REST API server (auth, accounts, pairings, sync, bale login)
│   ├── artery/                  # Multi-Artery Virtual Pool, state machine, EWMA telemetry
│   ├── bale/                    # Reverse-engineered Bale WebSocket, gRPC-Web auth, Protobuf
│   ├── config/                  # Environment & settings loader
│   ├── db/                      # GORM + SQLite models, database migrations, events
│   ├── dcconn/                  # DataChannel adapter & XChaCha20-Poly1305 obfuscator
│   ├── dns/                     # Anti-poisoning application DNS resolver
│   ├── livekit/                 # LiveKit SFU transport wrapper & token handling
│   ├── logger/                  # Component-based logging with disk persistence
│   ├── pool/                    # Connection pooling
│   ├── proxy/                   # TCP forwarding primitives
│   ├── quicconn/                # QUIC-over-Opus net.PacketConn adapter (OpusPacketConn)
│   ├── router/                  # Iranian CIDR trie, domain bypass engine, router state
│   ├── rtpconn/                 # Pion Opus track bridge, logical RTP timestamping, DTX comfort noise
│   ├── sync/                    # Event-sourced client ↔ server sync engine
│   ├── transport/               # WebRTC transport abstractions
│   └── webui/                   # Go embed root for compiled frontend
├── web/                         # React + TypeScript + Vite + Tailwind CSS source
├── research/
│   ├── traffic_sniffer.py       # Stealth Chrome CDP sniffer (captures WS frames, API calls, JS source)
│   ├── captured_traffic/        # Real-time logs of captured requests, WS frames, and JS sources
│   │   ├── api_calls.jsonl      # All HTTP/HTTPS requests and responses with headers/bodies
│   │   ├── api_summary.txt      # Clean summary list of touched endpoints
│   │   ├── websocket_frames.jsonl # Complete JSON lines of every sent/received WS packet
│   │   ├── websocket_readable.txt # Decoded Protobuf strings, RPC methods, and chat traffic
│   │   ├── js_sources/          # Raw downloaded JS source bundles from web.bale.ai
│   │   └── session_info.json    # Extracted JWT tokens, user IDs, and client metadata
│   └── knowledge/               # Living project documentation & architecture notes
│       ├── README.md            # Index pointing to master documentation
│       └── PROJECT_KNOWLEDGE_BASE.md  # ← THIS FILE (Master Knowledge Base)
├── database_backup/             # DB synchronization utilities & backup scripts
├── build_web.sh                 # Headless Vite build & Go sync script
├── Makefile                     # Build & deployment targets
└── .gitignore                   # Excludes binaries (client, server, *.test, *.out)
```

---

## 5. Build, Run & Deployment Instructions

### 5.1. Building from Source

```bash
# 1. Build everything (React frontend + Go embed + client/server/migrate binaries)
make build

# Or build individual components:
make build-web      # Builds web/dist and syncs to internal/webui/dist
make build-client   # Builds bin/client
make build-server   # Builds bin/server
make build-migrate  # Builds bin/migrate
```

### 5.2. Running the Server (Foreign VPS / Clever Cloud)

```bash
# Start server binary (API on :6680 or PORT env)
./bin/server

# In Docker:
docker-compose up -d
```

### 5.3. Running the Client (Restricted Host in Iran)

```bash
# Run client (Web UI available at http://localhost:6681)
./bin/client

# Once connected:
# SOCKS5 Proxy: 127.0.0.1:1080
# HTTP Proxy:   127.0.0.1:8080
```

### 5.4. Running the Protocol Sniffer & Traffic Analyzer

```bash
# Launch stealth Google Chrome with CDP packet logging & JS scraping:
python3 research/traffic_sniffer.py
```

---

## 6. Troubleshooting & Operational Gotchas

| Symptom | Root Cause | Solution |
| :--- | :--- | :--- |
| **Bale OTP Fails with HTTP 405 (`StartPhoneAuth`)** | Dynamic regex extractor matched `https://assets.bale.ai/configs.json` in Bale JS bundles instead of gRPC host, setting `bale_grpc_base` to asset CDN returning `405 Not Allowed nginx/1.23.1`. | Tightened regexes in `internal/bale/extractor.go`, added guards in `SetBaleGRPCBase` and `SetLiveKitOrigin` to reject CDN/json URLs, enabled dynamic `PUT`/`POST` on `/api/bale/constants`, and added in-memory auto-reload to `handleSettings`. |
| **Bale OTP Fails (`StartPhoneAuth` header/version)** | Stale `app_version` or missing paired `mt_*` and `language: fa` headers in gRPC-Web request. | Updated `app_version` to `169491` and added full header set in `internal/bale/auth.go`. Stale cookies are no longer injected. |
| **Silent Disconnects / Stale Calls** | Bale client app version is deprecated by Bale servers. | The system scrapes `SENTRY_RELEASE` build number dynamically. Alternatively, trigger `/api/bale/constants/sync` in the UI. |
| **QUIC CWND Collapses to Minimum** | Sub-transport packet reordering occurred due to multi-track striping under a single QUIC connection. | Enforce the 1:1 track-to-QUIC rule. Multi-path must only be managed via the `Artery` orchestrator. |
| **LiveKit SFU Drops Session** | Audio track was silent for >30 seconds. | Ensure `minimalOpusSilence` (`0xF8 0xFF 0xFE`) comfort-noise injection is running every 20ms in `rtpconn.go`. |
| **Server Marked as Busy / Hanging Calls** | Call was not discarded on Bale's backend or client abruptly disconnected. | `router.ForceEndCall` now executes `cancelFn` and calls `baleClient.DiscardCall` with the complete 12-entry metadata envelope. |
| **QUIC Handshake Fails over Opus Track (`context deadline exceeded`)** | Obfuscation secret mismatch between server and client (e.g. server had `OBFUSCATION_SECRET` env var set, while client had empty secret). Server encrypted Opus RTP payloads with XChaCha20-Poly1305, client sent raw QUIC packets. | Enabled dynamic sync and storage of `obfuscation_secret` in DB settings via `/api/sync/snapshot` and `/api/remote/sync-from-server`. Updated both client and server to dynamically look up `obfuscation_secret` from DB settings, env, and config. |
| **End Call / Disconnect Hangs Server Account in Call** | Server worker captured `expectedCallerID = 0` before user paired accounts; server discarded `BLETUN:ENDCALL` when `expectedCallerID == 0`; `handleSFUProxy` blocked on `listener.Accept` for 90s without listening for messages or context cancel; `apiSrv.OnForceEndCall` was unwired returning 501. | Updated `runSessionLoopDB` to resolve caller ID dynamically via `:id` suffix, DB pairing, or active session; unblocked `listener.Accept` immediately upon `ctx.Done()`; synchronized active session with `sessionDone` chan; wired `apiSrv.OnForceEndCall` to call `callRouter.ForceEndCall` and reset all stuck accounts in DB; updated client `tm.Stop()` to automatically send `BLETUN:ENDCALL` upon disconnect. |

---

## 7. Mandatory Maintenance Protocol for Developers & AI Agents

Whenever a change is introduced to this repository:
1. **Locate the Impacted Section:** Check Section 3 (Component Deep Dive), Section 4 (Repository Structure), or Section 5 (Deployment).
2. **Update the Specification:** Document any changed structs, APIs, protocols, or behavioral patterns.
3. **Record in Change Log:** Add a dated entry to the table below summarizing what changed and why.

### Change Log
| Date (Local) | Author / Agent | Scope / Target | Summary of Changes |
| :--- | :--- | :--- | :--- |
| 2026-09-07 | Antigravity AI | Repository Root | Initialized master architecture document & knowledge base. |
| 2026-09-07 | Antigravity AI | `.gitignore` | Configured comprehensive ignore rules for Go binaries (`/bin/`, `/client`, `/server`, `cmd/*/*`, `*.test`, `*.out`, `*.exe`) while preserving source files. |
| 2026-09-07 | Antigravity AI | `research/traffic_sniffer.py` | Implemented stealth Google Chrome CDP sniffer capturing HTTP calls, WebSocket frames (decoded Protobuf), JS bundles from `web.bale.ai`, and auth tokens. |
| 2026-09-07 | Antigravity AI | `internal/bale/`, `internal/api/`, `internal/router/`, `cmd/server/` | Resolved Bale OTP login, WebSocket push event drops, and DiscardCall failure: updated constants to `169491` / `151.0.0.0`, expanded `buildMetadata()` to 12-key envelope (`mt_*` and `language: fa`), added paired headers to gRPC-Web calls, removed stale cookie injection from `StartPhoneAuth`, wired session cancel/discard callbacks in `router.ForceEndCall`, and added unit test suites. |
| 2026-09-07 | Antigravity AI | `internal/bale/`, `internal/api/` | Fixed HTTP 405 on `StartPhoneAuth`: resolved upstream URL corruption where `assets.bale.ai/configs.json` was scraped as `bale_grpc_base`. Added strict property regexes (`reBaleGRPC`, `reBaleWS`), defensive URL guards in `SetBaleGRPCBase` / `SetLiveKitOrigin`, dynamic PUT/POST support for `/api/bale/constants`, and live reload in `/api/settings`. |
| 2026-09-07 | Antigravity AI | `internal/api/`, `cmd/server/`, `cmd/client/` | Fixed QUIC handshake failure (XChaCha20 obfuscation mismatch) and Call Disconnect / Teardown deadlock: added `obfuscation_secret` sync via snapshot and settings APIs, resolved dynamic obfuscation on both client and server; fixed `runSessionLoopDB` to dynamically detect caller ID and immediately ACK/terminate calls upon `BLETUN:ENDCALL`; added listener unblocking on session context cancellation in `handleSFUProxy`; wired `apiSrv.OnForceEndCall` to force terminate all active router sessions and reset DB statuses; updated client `tm.Stop()` to automatically send `BLETUN:ENDCALL` on disconnect. |
| 2026-09-07 | Antigravity AI | `cmd/client/main.go` | Implemented Fast Online Connection Policy & Dynamic Artery Scaling: activated SOCKS5 (:10909), HTTP (:9095) proxies, and Artery Orchestrator immediately upon first account pair connection (`proxyOnce.Do` / `orchOnce.Do`). Enabled real-time telemetry streaming and dynamic zero-downtime routing loop expansion across subsequent connecting arteries via P2C+WRR while preserving sequential rate-limit safe dialing. |
| 2026-09-07 | Antigravity AI | `web/src/`, `internal/api/`, `cmd/client/` | Unified Disconnect & Server Call Termination: merged `Stop()` and `ForceEndCall()` into `StopAndEndCalls()` returning per-account termination results via `/api/tunnel/stop`. Synchronized Dashboard UI so clicking DISCONNECT immediately activates the ENDING status animation on the END CALLS button. Kept END CALLS button visible as a live status indicator (ENDED ✓ or RETRY END CALLS on failure) with click-to-retry enabled if any server accounts fail to disconnect. |


