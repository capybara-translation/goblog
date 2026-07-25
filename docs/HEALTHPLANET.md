# Health Planet integration (weight/blood pressure data sync)

A mechanism that fetches weight, body fat percentage, blood pressure, and pulse from Tanita's Health Planet API once a day at 0:00 JST and idempotently upserts them into goblog's `health_records` table. Because it fetches over a 30-day window and upserts via `UNIQUE(measured_at, metric)`, late manual entries (backfilled a day or more later) are automatically absorbed by the next sync.

The whole feature is gated by the `HEALTHPLANET_ENABLED` environment variable. When unset (disabled by default), the daily sync is a no-op (exit 0), the admin UI is hidden, and the OAuth API routes are not registered.

## Architecture

```
Admin UI (/admin/healthplanet)
  └─ "連携する" (Connect) → Health Planet authorization → /admin/healthplanet/success?code=
       └─ "連携を完了する" (Complete linking) → POST /api/v1/healthplanet/exchange
            └─ Token exchange → saved to healthplanet_tokens

systemd timer (daily at 0:00 JST, always installed)
  └─ goblog-hpsync.service (oneshot)
       └─ /opt/goblog/bin/hpsync run
            0. If HEALTHPLANET_ENABLED != true, skip (exit 0)
            1. Load the token from healthplanet_tokens
            2. Refresh → save the updated expires_at
            3. Fetch innerscan / sphygmomanometer over the most recent 30-day window
            4. Upsert into health_records (UNIQUE(measured_at, metric))
            5. On failure or imminent token expiry, exit≠0 → journal / CloudWatch
```

The redirect URI for the OAuth authorization flow is `{BASE_URL}/admin/healthplanet/success`. The code is deliberately not exchanged until the admin explicitly clicks the "連携を完了する" (Complete linking) button — this prevents a linking attack (code injection) where an attacker tricks the blog admin into consuming the attacker's own authorization code.

Related files:

| File | Role |
|---|---|
| `cmd/hpsync/main.go` | CLI entry point (`run` / `auth` subcommands) |
| `internal/healthplanet/client.go` | Health Planet API client (OAuth, innerscan, sphygmomanometer) |
| `internal/service/health_sync_service.go` | Daily sync logic (token refresh, fetch, upsert, expiry warnings) |
| `internal/service/health_planet_admin_service.go` | Admin-facing OAuth flow (authorization URL generation, code exchange, status check) |
| `internal/http/handlers_healthplanet.go` | HTTP handlers (status / auth-url / exchange) |
| `internal/repo/health_record_repo.go` | Upsert into the `health_records` table |
| `internal/repo/healthplanet_token_repo.go` | Load / save for the single-row `healthplanet_tokens` table |
| `web-admin/src/pages/HealthPlanet.tsx` | Admin UI: shows link status, starts the authorization flow |
| `web-admin/src/pages/HealthPlanetSuccess.tsx` | Admin UI: the "連携を完了する" (Complete linking) screen after redirect |
| `deploy/goblog-hpsync.service` | oneshot service unit |
| `deploy/goblog-hpsync.timer` | Daily schedule (0:00 JST, Persistent=true) |
| `deploy/healthplanet.env.example` | Config template (deployed to `/etc/goblog/healthplanet.env`) |
| `scripts/hpsync-run.sh` | CloudWatch dead-man's-switch wrapper (optional) |

---

## Setup

### 1. Register the app with Health Planet

Register an app as a **Web Application** at [https://www.healthplanet.jp/](https://www.healthplanet.jp/). Register your blog's domain, and set the redirect URI to `{BASE_URL}/admin/healthplanet/success` (e.g. `https://blog.example.com/admin/healthplanet/success`). Note the client_id and client_secret you receive.

> **Note**: the CLI fallback (`hpsync auth`) redirects to Tanita's own `success.html`, so it doesn't need to be registered separately from the blog domain — but for production use, standardizing on the admin UI flow is recommended.

### 2. Deploy the linking config file (a single source shared by goblog and hpsync)

```bash
sudo install -D -o goblog -g goblog -m 600 \
  deploy/healthplanet.env.example /etc/goblog/healthplanet.env
sudo nano /etc/goblog/healthplanet.env
# Fill in HEALTHPLANET_ENABLED / _CLIENT_ID / _CLIENT_SECRET / DATABASE_PATH
```

`/etc/goblog/healthplanet.env` is the **single source read by both** goblog itself (loaded via `goblog.service`'s `EnvironmentFile=-`) and hpsync (`goblog-hpsync.service`). Rotating a secret only requires updating this one file. `DATABASE_PATH` is for hpsync; on the goblog side, the inline `Environment=` directives in the unit take precedence (last one wins), so having it duplicated there is harmless.

```bash
sudo systemctl restart goblog
```

If the "連携する" (Connect) button appears on the admin page at `/admin/healthplanet`, the feature is enabled.

> **Note for existing servers**: if your already-installed `/etc/systemd/system/goblog.service` doesn't have an `EnvironmentFile=-/etc/goblog/healthplanet.env` line (i.e. it was set up before this feature existed), add it **before** the block of inline `Environment=` entries, then run `sudo systemctl daemon-reload && sudo systemctl restart goblog`. The repo's `deploy/goblog.service` already has it added.

### 3. Distribute the binary

```bash
make deploy
```

`make deploy` places `bin/hpsync` and `scripts/hpsync-run.sh` into `/opt/goblog/bin/`.

### 4. Install the systemd unit

```bash
sudo cp deploy/goblog-hpsync.service deploy/goblog-hpsync.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now goblog-hpsync.timer
```

It's fine to have the timer always installed even when `HEALTHPLANET_ENABLED != true`. In that case `hpsync run` just prints "skipping" and exits 0 with no side effects.

### 5. Authorize from the admin UI

Open the admin page at `/admin/healthplanet` in a browser:

1. Click "連携する" (Connect) → you're redirected to the Health Planet authorization screen
2. Log in to Health Planet and approve access
3. You're redirected back to the blog at `/admin/healthplanet/success?code=…`
4. Click the "連携を完了する" (Complete linking) button → `POST /api/v1/healthplanet/exchange` is called and the token is saved

Once the admin page shows "最終リフレッシュ: …" (Last refresh: …), authorization is complete.

---

## Verification

Once the initial authorization is complete from the admin UI, immediately trigger a manual sync to confirm the refresh works correctly. If `journalctl` shows no "token refresh failed" warning, redirect_uri matching is fine.

```bash
# Run a manual sync
sudo systemctl start goblog-hpsync.service

# Check the log ("Sync complete." means success. If "token refresh failed" appears,
# a redirect_uri mismatch is likely — re-authorize from the admin UI to align the redirect_uri)
journalctl -u goblog-hpsync -n 20

# Next scheduled run of the timer
systemctl list-timers goblog-hpsync.timer

# Confirm records landed in the DB
sqlite3 /var/lib/goblog/goblog.db \
  'SELECT metric, COUNT(*) FROM health_records GROUP BY metric;'
```

On success, the "トークン最終リフレッシュ" (Token last refreshed) timestamp on the `/admin/healthplanet` admin page also updates (`healthplanet_tokens.updated_at`).

---

## Monitoring (optional)

You can monitor whether `hpsync run` succeeds using a CloudWatch dead-man's switch. The wrapper (`hpsync-run.sh`) sends the metric `SyncSuccess` (success=1 / failure=0) on every run, and a CloudWatch Alarm detects both "failure (value 0)" and "didn't run at all — timer never fired, instance stopped (no data point)". **SNS delivers notifications; CloudWatch Alarm is the one that detects.** `TreatMissingData=breaching` is what catches the latter — this is the crux of the dead-man's switch (SNS alone cannot detect "it never ran").

> **Important**: don't set up the alarm while `HEALTHPLANET_ENABLED != true`. While disabled, `hpsync run` exits 0 every time, so `SyncSuccess=1` keeps being reported and the alarm never fires (it looks healthy when it isn't). Enable monitoring only after the service is actually running.

### 1. Create a least-privilege IAM user

Lightsail doesn't support EC2-style IAM instance roles, so you need an IAM user with programmatic access keys. Scope its permissions to just **`cloudwatch:PutMetricData` for the specific namespace** (even if the key leaks, nothing beyond sending metrics is possible).

First, save the policy as `policy.json`:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "GoblogHPSyncMetric",
      "Effect": "Allow",
      "Action": "cloudwatch:PutMetricData",
      "Resource": "*",
      "Condition": { "StringEquals": { "cloudwatch:namespace": "Goblog/HPSync" } }
    }
  ]
}
```

> `cloudwatch:PutMetricData` doesn't support resource-level restriction (`Resource: "*"` is required), so the `Condition` clause scopes it down to the namespace instead, to keep the privilege minimal.

Create the user, attach the policy, and issue an access key (run with admin credentials):

```bash
aws iam create-user --user-name goblog-hpsync
aws iam put-user-policy --user-name goblog-hpsync \
  --policy-name goblog-hpsync --policy-document file://policy.json
aws iam create-access-key --user-name goblog-hpsync
# -> Note the AccessKeyId / SecretAccessKey from the output (the secret is shown only this once)
```

> If you've already set up monitoring for the DB backup (docs/BACKUP.md), you can skip creating a dedicated user and instead widen the existing `goblog-backup` user's policy Condition to `"cloudwatch:namespace": ["Goblog/Backup", "Goblog/HPSync"]`, reusing the same access key.

### 2. Add CloudWatch credentials to `/etc/goblog/healthplanet.env`

```bash
CW_NAMESPACE=Goblog/HPSync
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
AWS_DEFAULT_REGION=ap-northeast-1
```

If `CW_NAMESPACE` is unset, the wrapper won't send metrics and will just forward the exit code (this variable is the monitoring on/off switch).

### 3. Switch `goblog-hpsync.service`'s `ExecStart` to the wrapper

```ini
# ExecStart=/opt/goblog/bin/hpsync run
ExecStart=/opt/goblog/bin/hpsync-run.sh
```

```bash
sudo systemctl daemon-reload
```

The wrapper uses the `aws` CLI. If it's not installed on the server, install it with AWS's official v2 installer (do not use apt's `awscli`; the steps are shared with "Install `sqlite3` / the `aws` CLI on the server" in docs/BACKUP.md).

### 4. Create the SNS topic and CloudWatch Alarm (run with admin credentials)

```bash
# (a) Create an SNS topic for notifications and subscribe an email address
TOPIC_ARN=$(aws sns create-topic --name goblog-hpsync-alerts --query TopicArn --output text)
aws sns subscribe --topic-arn "$TOPIC_ARN" --protocol email \
  --notification-endpoint you@example.com

# A confirmation email will arrive, but don't click the "Confirm subscription" link —
# instead copy the value of the Token= parameter from the link URL and confirm via the CLI:
aws sns confirm-subscription --topic-arn "$TOPIC_ARN" \
  --token "<the Token= value from the link in the email>" \
  --authenticate-on-unsubscribe true

# If a goblog-backup-alerts topic already exists from backup monitoring, feel free to
# reuse that TOPIC_ARN instead of creating a new one (no reason to split notifications going to the same email)

# (b) An alarm that fires when "no success heartbeat arrives within 24h, or a failure (0) arrives"
aws cloudwatch put-metric-alarm \
  --alarm-name goblog-hpsync-missing \
  --alarm-description "Daily Health Planet sync did not succeed" \
  --namespace Goblog/HPSync --metric-name SyncSuccess \
  --statistic Minimum --period 86400 \
  --evaluation-periods 1 --datapoints-to-alarm 1 \
  --threshold 1 --comparison-operator LessThanThreshold \
  --treat-missing-data breaching \
  --alarm-actions "$TOPIC_ARN" --ok-actions "$TOPIC_ARN"
```

> Because it's `Minimum < 1`, the alarm fires on either "failure (0)" or "no data point", and sends an OK notification once it recovers (next success = 1). Keep the region consistent with the env file (SNS / CloudWatch / metrics must all be in the same region).

> **Why `--authenticate-on-unsubscribe true`**: the unsubscribe link at the bottom of SNS notification emails unsubscribes with a single unauthenticated GET — no confirmation screen. An email provider's link-scanning feature (e.g. phishing inspection) visiting that link automatically is enough to unsubscribe — an "Unsubscribe Confirmation" arrives and monitoring silently stops (this was actually observed happening). Setting this flag at confirmation time makes AWS authentication mandatory to unsubscribe, preventing accidents via the link.
>
> **An ALARM state right after creation is expected**: since there's no metric data yet, `TreatMissingData=breaching` puts it into ALARM within a few minutes. It's fine to leave it as-is until the first successful sync metric (step 5) arrives and it returns to OK. If you'd rather avoid getting notified twice, create this alarm after confirming the happy path in step 5 instead.

### 5. Verify the monitoring

```bash
# Happy path: manually sync on the server → a metric is sent
sudo systemctl start goblog-hpsync.service

# Check the metric (with admin credentials — the hpsync user only has PutMetricData, no read permission)
# Note: `date -u -d '1 day ago'` assumes GNU date (Linux). On macOS/BSD, use `date -u -v-1d +%FT%TZ` instead.
aws cloudwatch get-metric-statistics --namespace Goblog/HPSync --metric-name SyncSuccess \
  --start-time "$(date -u -d '1 day ago' +%FT%TZ)" --end-time "$(date -u +%FT%TZ)" \
  --period 86400 --statistics Minimum Maximum

# Failure path: deliberately trigger a failure on the server, sending metric 0
# (if the alarm → email arrives within a few minutes, the dead-man's switch is working.
#   Blanking out client_id makes hpsync exit 1 with a config error, reproducing a failure with no side effects)
sudo bash -c 'set -a; . /etc/goblog/healthplanet.env; set +a; HEALTHPLANET_CLIENT_ID= /opt/goblog/bin/hpsync-run.sh'; echo "exit=$?"
```

---

## Re-authorization (recovery procedure)

Re-authorization is needed if the token has expired or authorization has been revoked. One of the following will appear in the logs:

- `Error: healthplanet re-authorization required: ...` — the refresh failed and the token has also expired
- `Error: healthplanet token expiring soon: ...` — a warning that the refresh didn't extend the validity period (expiring within 7 days)

**Normal re-authorization (when the admin UI is available):**

Click "再認可する" (Re-authorize) on the `/admin/healthplanet` admin page and repeat step "5. Authorize from the admin UI" above.

**CLI fallback (when the admin UI is unavailable):**

```bash
sudo -u goblog bash -c 'set -a; . /etc/goblog/healthplanet.env; set +a; /opt/goblog/bin/hpsync auth'
```

Open the displayed URL in a browser, approve access on Health Planet, then copy the `code=` parameter from the address bar and paste it into the prompt. Because this redirects to Tanita's own `success.html`, no access to the blog domain is needed — it works even if the only access to the server is via SSH.

---

## Operational notes

- **Rate limiting**: the Health Planet API allows 60 requests/hour. The daily sync makes 2 calls (innerscan + sphygmomanometer), leaving plenty of headroom even with admin UI operations added on top. Repeated manual runs in quick succession can approach the limit, so be mindful of that.
- **Data outside the sync window**: the fetch window is the most recent 30 days. Older historical data is not synced automatically. If you need historical data, use Health Planet's manual export and insert it into the DB directly.
- **Tokens stored in plaintext**: the `healthplanet_tokens` table stores the access token and refresh token in plaintext. Because the DB is included in the S3 backup, the backup likewise contains the plaintext tokens — keep this in mind (see docs/BACKUP.md for access control on the backup).
- **Token expiry**: 30 days. Because the daily sync refreshes it every time, it's normally extended automatically, but per the API's behavior there's no rotation (verified against the real API, 2026-07). If refreshing stops extending the validity period, errors start appearing in the journal 7 days before expiry (if the dead-man's switch is configured, you'll get an email). In that case, re-authorize promptly.
- **Display on the blog**: `/health` shows a line chart (with period switching), and posts display a daily-average badge for the day specified via `health_date` (set in the editor). Both are gated by `HEALTHPLANET_ENABLED`.
