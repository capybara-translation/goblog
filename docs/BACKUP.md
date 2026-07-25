# DB Backup (SQLite → S3, daily)

A mechanism that uploads a consistent snapshot of the SQLite DB behind goblog running on Lightsail to S3 once a day. RPO is 24 hours (the design accepts up to one day of data loss in the worst case).

## Architecture

```
systemd timer (daily at 18:00 UTC)
  └─ goblog-backup.service (oneshot)
       └─ /opt/goblog/bin/backup-db.sh
            1. Check the DB exists → take a read-only consistent snapshot with sqlite3 .backup
            2. Verify with integrity_check + a check that the posts table exists
            3. Compress with gzip -9
            4. aws s3 cp (PutObject permission only)
            5. Send a heartbeat to CloudWatch (success=1 / failure=0)
S3:         Public Block + encryption + Versioning + Lifecycle (30-day expiry)
Monitoring: CloudWatch Alarm (TreatMissingData=breaching) → SNS → email
```

Monitoring is entirely within AWS. **SNS delivers notifications; CloudWatch Alarm is the one that detects.** The script sends the metric `BackupSuccess=1` on success and `0` on failure. Setting the alarm to `TreatMissingData=breaching` lets it detect not only "failure (value 0)" but also "didn't run at all — instance stopped, timer never fired (no data point)" (this is the crux of the dead-man's switch). Note that SNS alone cannot detect the latter.

Related files:

| File | Role |
|---|---|
| `scripts/backup-db.sh` | The backup script itself. Deployed to `/opt/goblog/bin/` by `make deploy` |
| `deploy/goblog-backup.service` | oneshot service |
| `deploy/goblog-backup.timer` | Daily schedule |
| `deploy/backup.env.example` | Config template (deployed to `/etc/goblog/backup.env`) |

## Why not a plain `cp`

goblog runs in rollback journal mode (not WAL), so the live file can be mid-write at any given instant. A `cp` could capture that torn state and save an unrecoverable image. SQLite's online backup API (`.backup`) takes a transactionally consistent snapshot even while the app is running. Because the script opens the DB **read-only**, the backup process never mutates the production DB.

---

## Setup

### 1. Install `sqlite3` / the `aws` CLI on the server

`sqlite3` is available via apt. For the AWS CLI, **do not use apt's `awscli`** (on Ubuntu 24.04 there's no candidate — `E: Package 'awscli' has no installation candidate` — and it's an old v1 anyway). Use AWS's official v2 installer instead (it's a plain binary, so it also plays nicely with systemd sandboxing).

```bash
sudo apt update && sudo apt install -y sqlite3 unzip

cd /tmp
ARCH=$(uname -m)   # x86_64 or aarch64
curl "https://awscli.amazonaws.com/awscli-exe-linux-${ARCH}.zip" -o awscliv2.zip
unzip -q awscliv2.zip
sudo ./aws/install            # installs to /usr/local/aws-cli. Use --update to re-run
aws --version                # success if it prints aws-cli/2.x
```

### 2. Create and harden the S3 bucket

The bucket name must be globally unique (e.g. `my-goblog-backups`). CLI examples below.

```bash
BUCKET=my-goblog-backups
REGION=ap-northeast-1

# Create (omit --create-bucket-configuration only if the region is us-east-1)
aws s3api create-bucket --bucket "$BUCKET" --region "$REGION" \
  --create-bucket-configuration LocationConstraint="$REGION"

# Block Public Access (all four settings)
aws s3api put-public-access-block --bucket "$BUCKET" \
  --public-access-block-configuration \
  BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true

# Default encryption (SSE-S3 / AES256)
aws s3api put-bucket-encryption --bucket "$BUCKET" \
  --server-side-encryption-configuration \
  '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'

# Versioning (insurance against overwrite, accidental deletion, tampering)
aws s3api put-bucket-versioning --bucket "$BUCKET" \
  --versioning-configuration Status=Enabled

# Lifecycle (current version 30 days, noncurrent versions 7 days, abort incomplete MPU after 7 days)
aws s3api put-bucket-lifecycle-configuration --bucket "$BUCKET" \
  --lifecycle-configuration '{
    "Rules":[{
      "ID":"expire-backups","Status":"Enabled",
      "Filter":{"Prefix":"goblog/"},
      "Expiration":{"Days":30},
      "NoncurrentVersionExpiration":{"NoncurrentDays":7},
      "AbortIncompleteMultipartUpload":{"DaysAfterInitiation":7}
    }]
  }'
```

> For longer retention, change the Lifecycle rule to "transition to `GLACIER_IR` after 30 days → expire after 365 days".

### 3. Create a least-privilege IAM user

Lightsail doesn't support EC2-style IAM instance roles, so you need an IAM user with programmatic access keys. Scope its permissions to just **S3 PutObject** and **CloudWatch PutMetricData** (even if the key leaks, it can't enumerate, fetch, or delete existing backups; CloudWatch access is also limited to a specific namespace).

First, save the policy as `policy.json`:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "GoblogBackupPutOnly",
      "Effect": "Allow",
      "Action": "s3:PutObject",
      "Resource": "arn:aws:s3:::my-goblog-backups/goblog/*"
    },
    {
      "Sid": "GoblogBackupMetric",
      "Effect": "Allow",
      "Action": "cloudwatch:PutMetricData",
      "Resource": "*",
      "Condition": { "StringEquals": { "cloudwatch:namespace": "Goblog/Backup" } }
    }
  ]
}
```

> `cloudwatch:PutMetricData` doesn't support resource-level restriction (`Resource: "*"` is required), so the `Condition` clause scopes it down to the namespace instead, to keep the privilege minimal.

Create the user, attach the policy, and issue an access key:

```bash
aws iam create-user --user-name goblog-backup
aws iam put-user-policy --user-name goblog-backup \
  --policy-name goblog-backup --policy-document file://policy.json
aws iam create-access-key --user-name goblog-backup
# -> Note the AccessKeyId / SecretAccessKey from the output (the secret is shown only this once)
```

You'll record the issued access key ID / secret into the env file in a later step.

### 4. Monitoring (CloudWatch Alarm + SNS)

Build an AWS-native dead-man's switch. **SNS (notification) and CloudWatch Alarm (detection) are needed as a pair.**

```bash
# (a) Create an SNS topic for notifications and subscribe an email address
TOPIC_ARN=$(aws sns create-topic --name goblog-backup-alerts --query TopicArn --output text)
aws sns subscribe --topic-arn "$TOPIC_ARN" --protocol email \
  --notification-endpoint you@example.com

# A confirmation email will arrive, but don't click the "Confirm subscription" link —
# instead copy the value of the Token= parameter from the link URL and confirm via the CLI:
aws sns confirm-subscription --topic-arn "$TOPIC_ARN" \
  --token "<the Token= value from the link in the email>" \
  --authenticate-on-unsubscribe true

# (b) An alarm that fires when "no success heartbeat arrives within 24h, or a failure (0) arrives".
#     TreatMissingData=breaching is the key to catching "it never ran at all".
aws cloudwatch put-metric-alarm \
  --alarm-name goblog-backup-missing \
  --alarm-description "Daily goblog DB backup did not succeed" \
  --namespace Goblog/Backup --metric-name BackupSuccess \
  --statistic Minimum --period 86400 \
  --evaluation-periods 1 --datapoints-to-alarm 1 \
  --threshold 1 --comparison-operator LessThanThreshold \
  --treat-missing-data breaching \
  --alarm-actions "$TOPIC_ARN" --ok-actions "$TOPIC_ARN"
```

> The metric uses `--unit Count`, with a period of 1 day (86400 seconds). Because it's `Minimum < 1`, the alarm fires on either "failure (0)" or "no data point", and sends an OK notification once it recovers (next success = 1). Keep the region consistent with the env file (SNS/CloudWatch/metrics must all be in the same region).

> **Why `--authenticate-on-unsubscribe true`**: the unsubscribe link at the bottom of SNS notification emails (and on the Confirm completion page) unsubscribes with a single unauthenticated GET — no confirmation screen. An email provider's link-scanning feature visiting that link automatically is enough to unsubscribe, silently killing the monitoring (this actually happened while building the Health Planet monitoring). Setting this flag at confirmation time makes AWS authentication mandatory to unsubscribe. Existing subscriptions confirmed via the email link have `ConfirmationWasAuthenticated` set to `false` in `aws sns get-subscription-attributes`, so if this concerns you, unsubscribe and re-subscribe using the CLI method above.

### 5. Deploy the config file

```bash
sudo install -D -o goblog -g goblog -m 600 deploy/backup.env.example /etc/goblog/backup.env
sudo nano /etc/goblog/backup.env   # fill in S3_BUCKET / AWS keys / region
```

### 6. Deploy and enable the script and unit files

```bash
# make deploy places the script at /opt/goblog/bin/backup-db.sh
make deploy

# deploy the unit files
sudo cp deploy/goblog-backup.service deploy/goblog-backup.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now goblog-backup.timer
```

### 7. Verify (dry run)

Note that verification commands fall into two different execution contexts.

**(a) Run on the server** (systemd/local operations; no AWS credentials needed, or they come via the env file):

```bash
sudo systemctl start goblog-backup.service          # run manually
journalctl -u goblog-backup --no-pager -n 30        # check the log ("backup uploaded: ...")
systemctl list-timers goblog-backup.timer           # next scheduled run

# Verify the monitoring: load the env file and deliberately trigger a failure to send metric 0
# (if the alarm → email arrives within a few minutes, the dead-man's switch is working)
sudo bash -c 'set -a; . /etc/goblog/backup.env; set +a; DB_PATH=/nonexistent /opt/goblog/bin/backup-db.sh'; echo "exit=$?"
```

**(b) Run with admin credentials** (the same admin-level permissions used to create the bucket/IAM/SNS in steps 2–4; typically your local machine):

```bash
# Confirm objects showed up. The backup IAM user only has PutObject and no
# ListBucket, so this returns AccessDenied with the server's credentials (by design).
aws s3 ls s3://my-goblog-backups/goblog/ --recursive

# Check the metric. The backup user only has PutMetricData and no GetMetricStatistics,
# so this also requires admin credentials.
# Note: `date -u -d '1 day ago'` assumes GNU date (Linux).
#       On macOS/BSD, use `date -u -v-1d +%FT%TZ` instead.
aws cloudwatch get-metric-statistics --namespace Goblog/Backup --metric-name BackupSuccess \
  --start-time "$(date -u -d '1 day ago' +%FT%TZ)" --end-time "$(date -u +%FT%TZ)" \
  --period 86400 --statistics Minimum Maximum
```

> Why the split: the IAM user from step 3 has **least privilege** — only `s3:PutObject` + `cloudwatch:PutMetricData`. It deliberately has no read permissions (List/Get), so that even if the key leaks it can't be used to enumerate, fetch, or delete existing backups. As a result, the read-only verification commands must be run separately with admin credentials.

---

## Restore procedure

> **A backup you've never restored isn't a backup.** After setting this up, do a full dry run once into a directory separate from production.

> **Credential note**: downloading the object in step 1 (`aws s3 cp` = `s3:GetObject`) **requires admin credentials**. The backup IAM user only has `s3:PutObject` and no Get/List, so it returns AccessDenied with the server's credentials (by design). If you keep admin credentials locally, download locally and `scp` to the server; if you want to do everything on the server, use admin credentials there temporarily. Step 3 (swapping the file in) is done on the server.

```bash
# 1. Fetch the object (with admin credentials — GetObject is required)
aws s3 cp s3://my-goblog-backups/goblog/2026/06/goblog-<ts>.db.gz /tmp/
gunzip /tmp/goblog-<ts>.db.gz

# 2. Verify integrity (must say "ok")
sqlite3 /tmp/goblog-<ts>.db 'PRAGMA integrity_check;'

# 3. Roll it into production (on the server: stop → swap → start)
sudo systemctl stop goblog
sudo install -o goblog -g goblog -m 644 /tmp/goblog-<ts>.db /var/lib/goblog/goblog.db
sudo systemctl start goblog
```

---

## Operational notes

- **Assumes a single instance.** If you scale to multiple instances, run the backup on only one of them (duplicate uploads are harmless but cost extra).
- **Storage cost**: a personal blog's SQLite file is only a few MB. After gzip it's even smaller — even 30 days of retention costs a few to a few dozen yen per month in S3.
- **If you want RPO down to seconds**: consider migrating to Litestream (continuous SQLite→S3 replication, OSS, free). This assumes switching to WAL mode (a DSN change in `internal/db/db.go`) and running a long-lived process.
- **Belt and suspenders**: Lightsail's automatic snapshots (whole-disk, daily, one click from the GUI) are a good code-free addition on top of this.
