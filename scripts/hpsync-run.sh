#!/usr/bin/env bash
# Wrapper around `hpsync run` that reports a success/failure heartbeat to
# CloudWatch (dead-man's-switch pattern, same as backup-db.sh). Used as the
# service ExecStart once monitoring is set up; without CW_NAMESPACE it just
# forwards the exit code.
set -uo pipefail

/opt/goblog/bin/hpsync run
status=$?

if [ -n "${CW_NAMESPACE:-}" ]; then
  value=1
  [ "$status" -ne 0 ] && value=0
  if ! aws cloudwatch put-metric-data \
      --namespace "$CW_NAMESPACE" \
      --metric-name SyncSuccess \
      --value "$value" \
      --unit Count; then
    echo "warning: failed to publish CloudWatch metric" >&2
  fi
fi

exit "$status"
