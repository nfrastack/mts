## 1.0.0 2026-10-18 <code at nfrastack dot com>

   ### Added
    - Initial release
    - Filter mode: stdin -> stdout with strftime format plus %.S/%.s/%.T
    - incremental (-i) and since-start (-s) timestamps
    - -m flag for compat (accepted, changes nothing)
    - UTC (-u/--utc) and ISO 8601/RFC3339 (--iso8601/--rfc3339) output
    - timezone override (--tz/--timezone, $TIMEZONE) via system zoneinfo
    - --prefix, --output/--truncate, -q/--quiet routing options
    - exec mode (mts -- cmd) separate stdout/stderr stamping, signal forwarding, child exit code propagation
    - privilege drop (--user) for child
    - output file permissions (mode, ownership)
