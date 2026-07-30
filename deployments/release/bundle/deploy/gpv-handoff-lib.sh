#!/usr/bin/env bash

# gpv_handoff_prepare pre-creates the fixed GPV RPC durable while the old app
# is still running. The new durable starts at the old consumer's AckFloor+1,
# so every message published during the stop/start window is retained.
#
# The caller owns DC=(docker compose ...). Any non-zero result is a hard gate:
# callers must return before stopping or recreating the old app.
gpv_handoff_prepare() {
  "${DC[@]}" run --rm --no-deps gpv-handoff
}
