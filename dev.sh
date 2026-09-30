#!/bin/bash
export HMAC_KEY=0000000000000000000000000000000000000000000000000000000000000000
export ALLOW_DRAFT=true

go run ./cmd/api &
API_PID=$!

cd web && npm run dev

kill $API_PID 2>/dev/null
wait $API_PID 2>/dev/null
