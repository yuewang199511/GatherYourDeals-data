#!/bin/sh
set -e

# Run init non-interactively using credentials from env vars.
# On first deploy this creates the admin account.
# On subsequent deploys initCmd detects the existing admin and exits cleanly.
if [ -n "$GYD_ADMIN_USERNAME" ] && [ -n "$GYD_ADMIN_PASSWORD" ]; then
    printf '%s\n%s\n%s\n' \
        "$GYD_ADMIN_USERNAME" \
        "$GYD_ADMIN_PASSWORD" \
        "$GYD_ADMIN_PASSWORD" \
        | gatheryourdeals init
else
    echo "ERROR: GYD_ADMIN_USERNAME and GYD_ADMIN_PASSWORD must be set" >&2
    exit 1
fi

# Replace shell with the server process so SIGTERM reaches Go directly.
exec gatheryourdeals serve
