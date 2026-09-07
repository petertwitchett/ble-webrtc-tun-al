#!/usr/bin/env python3
"""
traffic_sniffer.py — Advanced Stealth Sniffer & Traffic Analyzer for Bale Web App

Features:
- Launches a native Google Chrome browser instance without automation flags,
  bypassing Cloudflare Turnstile, anti-bot scripts, and web protections.
- Connects via Chrome DevTools Protocol (CDP) to track:
    1. All HTTP/HTTPS requests & responses (headers, bodies, status codes).
    2. All downloaded JavaScript sources from web.bale.ai (saving them to research/captured_traffic/js_sources/).
    3. All WebSocket traffic (handshake headers, cookies, token payloads, sent/received frames).
    4. Decodes binary Protobuf payloads in real-time, extracting RPC services, method names, and strings.
    5. Saves full session tokens (access_token, user_id, device UUIDs).
- Logs everything in real time to both terminal and structured files under research/captured_traffic/.
"""

import os
import sys
import json
import time
import re
import base64
import signal
import asyncio
import subprocess
import urllib.request
import urllib.error
from datetime import datetime
from pathlib import Path

# Dependency check
try:
    import websockets
except ImportError:
    print("[!] Missing 'websockets' package. Installing via pip...")
    subprocess.check_call([sys.executable, "-m", "pip", "install", "websockets"])
    import websockets

# Configuration
CHROME_PATH = "/usr/bin/google-chrome"
CDP_PORT = 9222
TARGET_URL = "https://web.bale.ai"
PROFILE_DIR = Path("/tmp/bale_sniffer_chrome_profile")
OUTPUT_DIR = Path(__file__).resolve().parent / "captured_traffic"
JS_OUTPUT_DIR = OUTPUT_DIR / "js_sources"

# Ensure output directories exist
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
JS_OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

# Output log files
HTTP_LOG = OUTPUT_DIR / "api_calls.jsonl"
HTTP_SUMMARY_LOG = OUTPUT_DIR / "api_summary.txt"
WS_FRAMES_LOG = OUTPUT_DIR / "websocket_frames.jsonl"
WS_READABLE_LOG = OUTPUT_DIR / "websocket_readable.txt"
SESSION_LOG = OUTPUT_DIR / "session_info.json"

# Terminal ANSI colors
GREEN = "\033[92m"
BLUE = "\033[94m"
YELLOW = "\033[93m"
CYAN = "\033[96m"
MAGENTA = "\033[95m"
RED = "\033[91m"
BOLD = "\033[1m"
RESET = "\033[0m"

# State tracking
pending_requests = {}      # requestId -> request info
js_request_ids = {}        # requestId -> (url, filename)
captured_tokens = {}       # token_type -> value
ws_connections = {}        # requestId -> ws_url
saved_js_hashes = set()


def extract_printable_strings(data_bytes: bytes, min_len: int = 3) -> list:
    """Extract human-readable ASCII/UTF-8 strings from binary Protobuf frames."""
    printable = []
    current = bytearray()
    for b in data_bytes:
        if 32 <= b <= 126:  # Printable ASCII
            current.append(b)
        else:
            if len(current) >= min_len:
                try:
                    s = current.decode("utf-8", errors="ignore")
                    printable.append(s)
                except Exception:
                    pass
            current = bytearray()
    if len(current) >= min_len:
        try:
            s = current.decode("utf-8", errors="ignore")
            printable.append(s)
        except Exception:
            pass
    return printable


def detect_bale_tokens(text: str):
    """Scan text or headers for Bale access tokens, user IDs, or API keys."""
    updated = False

    # JWT access token
    jwt_match = re.search(r"access_token=([A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*)", text)
    if not jwt_match:
        jwt_match = re.search(r"(eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9\.[A-Za-z0-9-_=]+\.[A-Za-z0-9-_.+/=]*)", text)
    if jwt_match:
        token = jwt_match.group(1)
        if captured_tokens.get("access_token") != token:
            captured_tokens["access_token"] = token
            print(f"\n{BOLD}{GREEN}[★ AUTH TOKEN DETECTED]{RESET} access_token: {token[:35]}...{RESET}")
            updated = True

    # User ID
    uid_match = re.search(r'"user_id"\s*:\s*([0-9]{8,12})', text)
    if not uid_match:
        uid_match = re.search(r'uid=([0-9]{8,12})', text)
    if uid_match:
        uid = int(uid_match.group(1))
        if captured_tokens.get("user_id") != uid:
            captured_tokens["user_id"] = uid
            print(f"{BOLD}{GREEN}[★ USER ID DETECTED]{RESET} {uid}{RESET}")
            updated = True

    # Web API Key
    key_match = re.search(r'([A-F0-9]{64})', text)
    if key_match:
        key = key_match.group(1)
        if captured_tokens.get("web_api_key") != key:
            captured_tokens["web_api_key"] = key
            print(f"{BOLD}{GREEN}[★ API KEY DETECTED]{RESET} {key}{RESET}")
            updated = True

    # App Version
    app_v = re.search(r'(?:appVersion|app_version|App_version)[:=]\s*["\']?([0-9]{5,7}|[0-9]+\.[0-9]+\.[0-9]+)["\']?', text)
    if app_v:
        v = app_v.group(1)
        if captured_tokens.get("app_version") != v:
            captured_tokens["app_version"] = v
            print(f"{BOLD}{GREEN}[★ APP VERSION DETECTED]{RESET} {v}{RESET}")
            updated = True

    if updated:
        with open(SESSION_LOG, "w") as f:
            json.dump(captured_tokens, f, indent=2)


class TrafficSniffer:
    def __init__(self):
        self.cmd_id = 0
        self.ws = None
        self.sessions = {}  # sessionId -> targetInfo
        self.chrome_process = None
        self.pending_body_cmd = {}  # cmd_id -> filename

    def next_id(self):
        self.cmd_id += 1
        return self.cmd_id

    async def send_cmd(self, method: str, params: dict = None, session_id: str = None):
        if not self.ws:
            return None
        msg_id = self.next_id()
        payload = {"id": msg_id, "method": method, "params": params or {}}
        if session_id:
            payload["sessionId"] = session_id
        await self.ws.send(json.dumps(payload))
        return msg_id

    def launch_chrome(self):
        """Launch genuine Chrome instance with anti-automation flags removed."""
        if not os.path.exists(CHROME_PATH):
            raise FileNotFoundError(f"Chrome binary not found at {CHROME_PATH}")

        print(f"{BOLD}{BLUE}[1/4] Launching Google Chrome...{RESET}")
        cmd = [
            CHROME_PATH,
            f"--remote-debugging-port={CDP_PORT}",
            f"--user-data-dir={PROFILE_DIR}",
            "--no-first-run",
            "--no-default-browser-check",
            "--start-maximized",
            "--disable-blink-features=AutomationControlled",
            "--flag-switches-begin",
            "--flag-switches-end",
            TARGET_URL
        ]
        self.chrome_process = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        print(f"      Chrome PID: {self.chrome_process.pid} (Debugging Port: {CDP_PORT})")

    async def wait_for_cdp(self, max_retries: int = 30):
        """Wait until Chrome's CDP endpoint responds."""
        print(f"{BOLD}{BLUE}[2/4] Connecting to Chrome DevTools Protocol...{RESET}")
        url = f"http://127.0.0.1:{CDP_PORT}/json/version"
        for i in range(max_retries):
            try:
                with urllib.request.urlopen(url, timeout=1) as resp:
                    if resp.status == 200:
                        data = json.loads(resp.read().decode("utf-8"))
                        return data.get("webSocketDebuggerUrl")
            except Exception:
                await asyncio.sleep(0.5)
        raise TimeoutError("Failed to connect to Chrome CDP endpoint.")

    async def run(self):
        self.launch_chrome()
        browser_ws_url = await self.wait_for_cdp()
        print(f"      Browser Debugger URL: {browser_ws_url}")

        print(f"{BOLD}{BLUE}[3/4] Attaching network sniffer & target monitors...{RESET}")
        async with websockets.connect(browser_ws_url, max_size=100 * 1024 * 1024) as ws:
            self.ws = ws

            # Enable target auto-attaching to monitor all tabs, frames, and workers
            await self.send_cmd("Target.setDiscoverTargets", {"discover": True})
            await self.send_cmd("Target.setAutoAttach", {
                "autoAttach": True,
                "waitForDebuggerOnStart": False,
                "flatten": True
            })

            print(f"\n{BOLD}{GREEN}[4/4] SNIFFER IS ACTIVE AND RECORDING!{RESET}")
            print(f"───────────────────────────────────────────────────────────────────────")
            print(f" {BOLD}Target:{RESET} {TARGET_URL}")
            print(f" {BOLD}Outputs:{RESET}")
            print(f"   • API Calls:       {HTTP_LOG}")
            print(f"   • API Summary:     {HTTP_SUMMARY_LOG}")
            print(f"   • WebSocket Log:   {WS_FRAMES_LOG}")
            print(f"   • Human Readable:  {WS_READABLE_LOG}")
            print(f"   • Saved JS Files:  {JS_OUTPUT_DIR}/")
            print(f"   • Session Info:    {SESSION_LOG}")
            print(f"───────────────────────────────────────────────────────────────────────")
            print(f"{YELLOW}Please interact with the opened Chrome browser (log in, test calls, disconnect).")
            print(f"All transmitted packets, RPC calls, and JS sources will be captured below.{RESET}\n")

            while True:
                try:
                    msg_str = await ws.recv()
                    msg = json.loads(msg_str)
                    await self.handle_cdp_message(msg)
                except asyncio.CancelledError:
                    break
                except Exception as e:
                    # Ignore transient decode issues
                    pass

    async def handle_cdp_message(self, msg: dict):
        method = msg.get("method")
        params = msg.get("params", {})
        session_id = msg.get("sessionId")

        # 1. Target attached
        if method == "Target.attachedToTarget":
            sub_session_id = params.get("sessionId")
            target_info = params.get("targetInfo", {})
            self.sessions[sub_session_id] = target_info

            # Enable Network with maximum buffer sizes
            await self.send_cmd("Network.enable", {
                "maxTotalBufferSize": 100_000_000,
                "maxResourceBufferSize": 50_000_000,
                "maxPostDataSize": 50_000_000
            }, session_id=sub_session_id)

            # Enable Page domain
            await self.send_cmd("Page.enable", {}, session_id=sub_session_id)

            # Inject anti-detection override
            stealth_js = """
                Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
                window.chrome = window.chrome || { runtime: {} };
            """
            await self.send_cmd("Page.addScriptToEvaluateOnNewDocument", {"source": stealth_js}, session_id=sub_session_id)

        # 2. HTTP Request Will Be Sent
        elif method == "Network.requestWillBeSent":
            req_id = params.get("requestId")
            req = params.get("request", {})
            url = req.get("url", "")
            url_lower = url.lower()

            record = {
                "timestamp": datetime.now().isoformat(),
                "requestId": req_id,
                "type": "request",
                "method": req.get("method"),
                "url": url,
                "headers": req.get("headers", {}),
                "postData": req.get("postData", "")
            }
            pending_requests[req_id] = record

            # Detect auth tokens or parameters in request
            detect_bale_tokens(url)
            detect_bale_tokens(json.dumps(req.get("headers", {})))
            if req.get("postData"):
                detect_bale_tokens(req.get("postData"))

            # Check if this is a JS source file on Bale
            if "web.bale.ai" in url and (url.endswith(".js") or "/static/js/" in url or "chunk" in url):
                filename = url.split("?")[0].split("/")[-1]
                if not filename.endswith(".js"):
                    filename += ".js"
                js_request_ids[req_id] = (url, filename, session_id)

            # Console output for interesting Bale endpoints
            if any(k in url_lower for k in ["bale.ai", "ble.ir", "meet", "auth", "startphoneauth", "validatecode", "turn"]):
                print(f"{CYAN}[REQ]{RESET} {req.get('method')} {url[:90]}")
                with open(HTTP_SUMMARY_LOG, "a") as f:
                    f.write(f"[{datetime.now().strftime('%H:%M:%S')}] REQ {req.get('method')} {url}\n")

        # 3. HTTP Response Received
        elif method == "Network.responseReceived":
            req_id = params.get("requestId")
            resp = params.get("response", {})
            url = resp.get("url", "")
            status = resp.get("status")
            mime = resp.get("mimeType", "")

            # Log tokens from response headers
            detect_bale_tokens(json.dumps(resp.get("headers", {})))

            record = pending_requests.get(req_id, {})
            record.update({
                "status": status,
                "statusText": resp.get("statusText"),
                "responseHeaders": resp.get("headers", {}),
                "mimeType": mime,
                "remoteIPAddress": resp.get("remoteIPAddress")
            })

            # Check if this is JS via mimeType
            if ("javascript" in mime or "ecmascript" in mime) and "web.bale.ai" in url:
                filename = url.split("?")[0].split("/")[-1]
                if not filename.endswith(".js"):
                    filename += ".js"
                js_request_ids[req_id] = (url, filename, session_id)

            # Write to jsonl
            with open(HTTP_LOG, "a") as f:
                f.write(json.dumps(record) + "\n")

            if any(k in url.lower() for k in ["bale.ai", "ble.ir", "meet", "auth"]):
                print(f"{GREEN}[RESP]{RESET} {status} {url[:90]} ({mime})")
                with open(HTTP_SUMMARY_LOG, "a") as f:
                    f.write(f"[{datetime.now().strftime('%H:%M:%S')}] RESP {status} {url}\n")

        # 4. HTTP Loading Finished (fetch body for JS and API calls)
        elif method == "Network.loadingFinished":
            req_id = params.get("requestId")
            if req_id in js_request_ids:
                url, filename, sess_id = js_request_ids[req_id]
                if filename not in saved_js_hashes:
                    saved_js_hashes.add(filename)
                    # Request response body via CDP
                    body_cmd_id = self.next_id()
                    self.pending_body_cmd[body_cmd_id] = filename
                    await self.ws.send(json.dumps({
                        "id": body_cmd_id,
                        "sessionId": sess_id,
                        "method": "Network.getResponseBody",
                        "params": {"requestId": req_id}
                    }))

        # 5. Handle Response Bodies (Saving JS source files)
        elif "result" in msg and "body" in msg.get("result", {}):
            body = msg["result"]["body"]
            is_base64 = msg["result"].get("base64Encoded", False)
            if is_base64:
                try:
                    body = base64.b64decode(body).decode("utf-8", errors="ignore")
                except Exception:
                    pass

            # Detect tokens/constants in JS code
            detect_bale_tokens(body[:5000])

            # Save JS file to disk under its exact filename
            filename = self.pending_body_cmd.pop(msg.get("id"), None)
            if not filename:
                filename = f"source_{len(saved_js_hashes)}.js"

            target_path = JS_OUTPUT_DIR / filename
            with open(target_path, "w", encoding="utf-8") as f:
                f.write(body)
            print(f"{BOLD}{MAGENTA}[JS SAVED]{RESET} {filename} ({len(body):,} bytes)")

        # 6. WebSocket Created
        elif method == "Network.webSocketCreated":
            req_id = params.get("requestId")
            url = params.get("url", "")
            ws_connections[req_id] = url
            print(f"\n{BOLD}{YELLOW}[WS CREATED]{RESET} {url}")
            with open(WS_READABLE_LOG, "a") as f:
                f.write(f"\n{'='*70}\n[{datetime.now().isoformat()}] WEBSOCKET CONNECTED: {url}\n{'='*70}\n")

        # 7. WebSocket Handshake Sent
        elif method == "Network.webSocketWillSendHandshakeRequest":
            req_id = params.get("requestId")
            req = params.get("request", {})
            headers = req.get("headers", {})
            detect_bale_tokens(json.dumps(headers))
            url = ws_connections.get(req_id, "unknown")

            with open(WS_READABLE_LOG, "a") as f:
                f.write(f"\n[WS HANDSHAKE REQUEST] {url}\n")
                for k, v in headers.items():
                    f.write(f"  {k}: {v}\n")

        # 8. WebSocket Handshake Response
        elif method == "Network.webSocketHandshakeResponseReceived":
            req_id = params.get("requestId")
            resp = params.get("response", {})
            status = resp.get("status")
            headers = resp.get("headers", {})
            detect_bale_tokens(json.dumps(headers))

            with open(WS_READABLE_LOG, "a") as f:
                f.write(f"[WS HANDSHAKE RESPONSE] Status {status}\n")
                for k, v in headers.items():
                    f.write(f"  {k}: {v}\n")

        # 9. WebSocket Frame Sent (Client -> Server)
        elif method == "Network.webSocketFrameSent":
            await self.process_ws_frame(params, direction="Client -> Server")

        # 10. WebSocket Frame Received (Server -> Client)
        elif method == "Network.webSocketFrameReceived":
            await self.process_ws_frame(params, direction="Server -> Client")

    async def process_ws_frame(self, params: dict, direction: str):
        req_id = params.get("requestId")
        response = params.get("response", {})
        opcode = response.get("opcode")  # 1 = text, 2 = binary
        payload_raw = response.get("payloadData", "")
        ws_url = ws_connections.get(req_id, "unknown")
        timestamp = datetime.now().strftime("%H:%M:%S.%f")[:-3]

        decoded_strings = []
        raw_bytes = b""

        if opcode == 1:  # Text frame
            text_content = payload_raw
            decoded_strings = [text_content]
            detect_bale_tokens(text_content)
        elif opcode == 2:  # Binary frame (Protobuf)
            try:
                raw_bytes = base64.b64decode(payload_raw)
                decoded_strings = extract_printable_strings(raw_bytes, min_len=3)
                detect_bale_tokens(" ".join(decoded_strings))
            except Exception:
                decoded_strings = ["<Failed to base64 decode binary payload>"]
        else:
            decoded_strings = [f"<Control Frame Opcode {opcode}>"]

        # Filter out noisy keep-alive ping/pong frames unless they contain strings
        is_ping_pong = len(raw_bytes) <= 4 and not decoded_strings
        summary_strings = [s for s in decoded_strings if len(s) >= 3 and not s.isdigit()]

        # Print to terminal
        dir_color = CYAN if direction.startswith("Client") else MAGENTA
        print(f"{dir_color}[WS {direction[:6]}]{RESET} Len: {len(raw_bytes) if opcode==2 else len(payload_raw)}B | Strings: {summary_strings[:4]}")

        # Save structured JSON
        frame_record = {
            "timestamp": datetime.now().isoformat(),
            "requestId": req_id,
            "url": ws_url,
            "direction": direction,
            "opcode": opcode,
            "length": len(raw_bytes) if opcode == 2 else len(payload_raw),
            "payload_base64": payload_raw if opcode == 2 else None,
            "payload_text": payload_raw if opcode == 1 else None,
            "extracted_strings": decoded_strings
        }
        with open(WS_FRAMES_LOG, "a") as f:
            f.write(json.dumps(frame_record) + "\n")

        # Save clean human-readable log
        with open(WS_READABLE_LOG, "a") as f:
            f.write(f"\n[{timestamp}] {direction} (Opcode {opcode}, {frame_record['length']} bytes)\n")
            if summary_strings:
                f.write(f"  RPC / Strings: {' | '.join(summary_strings)}\n")
            if raw_bytes and len(raw_bytes) <= 128:
                f.write(f"  Hex: {raw_bytes.hex()}\n")


async def main():
    sniffer = TrafficSniffer()

    def handle_sigint(sig, frame):
        print(f"\n{YELLOW}[*] Stopping sniffer and saving summary...{RESET}")
        if sniffer.chrome_process:
            try:
                sniffer.chrome_process.terminate()
            except Exception:
                pass
        print(f"{GREEN}[✓] Captured data saved in: {OUTPUT_DIR}{RESET}")
        sys.exit(0)

    signal.signal(signal.SIGINT, handle_sigint)
    signal.signal(signal.SIGTERM, handle_sigint)

    try:
        await sniffer.run()
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    asyncio.run(main())
