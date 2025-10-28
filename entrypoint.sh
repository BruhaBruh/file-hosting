#!/bin/sh
[ ! -f /app/configs/config.yaml ] && mkdir -p /app/configs && cp /app/configs.default/* /app/configs/ || true
[ ! -f /app/public/index.html ] && mkdir -p /app/public && cp /app/public.default/* /app/public/ || true
exec ./app
