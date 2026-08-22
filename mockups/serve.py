#!/usr/bin/env python3
"""Tiny preview server used only for the static design review."""
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import os

ROOT = Path(__file__).resolve().parents[1]
os.chdir(ROOT)

class Handler(SimpleHTTPRequestHandler):
    def end_headers(self):
        self.send_header("Cache-Control", "no-store")
        self.send_header("X-Content-Type-Options", "nosniff")
        super().end_headers()

    def log_message(self, fmt, *args):
        print(f"[saga-mockups] {self.address_string()} {fmt % args}", flush=True)

ThreadingHTTPServer(("127.0.0.1", 3000), Handler).serve_forever()
