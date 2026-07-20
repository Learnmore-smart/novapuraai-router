#!/usr/bin/env python3
"""
Minimal OpenAI-compatible mock upstream for end-to-end testing of new-api gateway.

Implements just enough of the OpenAI Chat Completions + Models APIs for the
new-api relay adapter to accept the channel and proxy a real request through it:
  - GET  /v1/models
  - POST /v1/chat/completions   (non-streaming + streaming)

Run: python3 mock_openai_server.py 8787
"""
import json
import sys
import time
import uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class Handler(BaseHTTPRequestHandler):
    server_version = "MockOpenAI/1.0"

    def _send(self, code: int, payload: dict, ctype: str = "application/json"):
        body = json.dumps(payload).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_json(self):
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length) if length > 0 else b""
        try:
            return json.loads(raw.decode("utf-8")) if raw else {}
        except Exception:
            return {}

    def do_GET(self):
        if self.path == "/v1/models":
            self._send(200, {
                "object": "list",
                "data": [
                    {"id": "gpt-3.5-turbo", "object": "model", "created": 1677610602, "owned_by": "mock"},
                    {"id": "gpt-4o-mini", "object": "model", "created": 1715367049, "owned_by": "mock"},
                    {"id": "gpt-4o", "object": "model", "created": 1715367049, "owned_by": "mock"},
                ],
            })
            return
        self._send(404, {"error": {"message": f"not found: {self.path}", "type": "invalid_request_error"}})

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self._send(404, {"error": {"message": f"not found: {self.path}", "type": "invalid_request_error"}})
            return

        body = self._read_json()
        model = body.get("model", "gpt-3.5-turbo")
        messages = body.get("messages", [])
        stream = body.get("stream", False)
        user_msg = ""
        for m in reversed(messages):
            if m.get("role") == "user":
                user_msg = m.get("content", "")
                break

        reply = f"Hello from MockOpenAI! You said: {user_msg!r}. (model={model})"
        created = int(time.time())

        if stream:
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.send_header("Cache-Control", "no-cache")
            self.send_header("Connection", "keep-alive")
            self.end_headers()
            chat_id = f"chatcmpl-{uuid.uuid4().hex[:12]}"
            chunks = [
                {"choices": [{"delta": {"role": "assistant"}, "index": 0}]},
                {"choices": [{"delta": {"content": reply}, "index": 0}]},
                {"choices": [{"delta": {}, "index": 0, "finish_reason": "stop"}],
                 "usage": {"prompt_tokens": 9, "completion_tokens": 12, "total_tokens": 21}},
            ]
            for ch in chunks:
                ch["id"] = chat_id
                ch["object"] = "chat.completion.chunk"
                ch["model"] = model
                ch["created"] = created
                self.wfile.write(f"data: {json.dumps(ch)}\n\n".encode("utf-8"))
                self.wfile.flush()
                time.sleep(0.02)
            self.wfile.write(b"data: [DONE]\n\n")
            self.wfile.flush()
            return

        self._send(200, {
            "id": f"chatcmpl-{uuid.uuid4().hex[:12]}",
            "object": "chat.completion",
            "created": created,
            "model": model,
            "choices": [{
                "index": 0,
                "message": {"role": "assistant", "content": reply},
                "finish_reason": "stop",
            }],
            "usage": {"prompt_tokens": 9, "completion_tokens": 12, "total_tokens": 21},
        })

    def log_message(self, fmt, *args):
        sys.stderr.write(f"[mock-openai] {self.address_string()} - {fmt % args}\n")


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8787
    srv = ThreadingHTTPServer(("127.0.0.1", port), Handler)
    print(f"[mock-openai] listening on http://127.0.0.1:{port}", flush=True)
    try:
        srv.serve_forever()
    except KeyboardInterrupt:
        srv.shutdown()
