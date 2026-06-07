# DB バックアップ（SQLite → S3、日次）

Lightsail 上で稼働する goblog の SQLite DB を、毎日 1 回 S3 に整合性の取れたスナップショットとしてアップロードする仕組み。RPO は 24 時間（最悪 1 日分のデータ損失を許容する設計）。

## 構成

```
systemd timer (毎日 18:00 UTC)
  └─ goblog-backup.service (oneshot)
       └─ /opt/goblog/bin/backup-db.sh
            1. DB の存在確認 → sqlite3 .backup で read-only 一貫スナップショット
            2. integrity_check + posts テーブル存在チェックで検証
            3. gzip -9 圧縮
            4. aws s3 cp（PutObject 権限のみ）
            5. CloudWatch に heartbeat 送信（成功=1 / 失敗=0）
S3:         Public Block + 暗号化 + Versioning + Lifecycle(30日失効)
監視:       CloudWatch Alarm (TreatMissingData=breaching) → SNS → メール
```

監視は AWS 完結。**SNS は通知の配信役で、検知役は CloudWatch Alarm** という分担。スクリプトは成功時にメトリクス `BackupSuccess=1`、失敗時に `0` を送る。Alarm を `TreatMissingData=breaching` にすることで、「失敗（値 0）」だけでなく「そもそも走らなかった＝インスタンス停止・timer 不発（データ点なし）」も検知できる（これが dead-man's switch の肝）。SNS 単独では後者を検知できない点に注意。

関連ファイル:

| ファイル | 役割 |
|---|---|
| `scripts/backup-db.sh` | バックアップ本体。`make deploy` で `/opt/goblog/bin/` に配置される |
| `deploy/goblog-backup.service` | oneshot サービス |
| `deploy/goblog-backup.timer` | 日次スケジュール |
| `deploy/backup.env.example` | 設定テンプレート（`/etc/goblog/backup.env` に配置） |

## なぜ単純な `cp` ではないのか

goblog はロールバックジャーナルモード（WAL ではない）で動作するため、稼働中のファイルは任意の瞬間に書き込み途中の可能性がある。`cp` はその torn な状態を取り込み、復元不能なイメージを保存しうる。SQLite のオンラインバックアップ（`.backup`）はアプリ稼働中でもトランザクション的に一貫したスナップショットを取る。スクリプトは DB を **read-only** で開くため、バックアップ処理が本番 DB を書き換えることはない。

---

## セットアップ手順

### 1. サーバに `sqlite3` / `aws` CLI を用意

```bash
sudo apt update && sudo apt install -y sqlite3 awscli
```

### 2. S3 バケットを作成・堅牢化

バケット名はグローバルで一意（例: `my-goblog-backups`）。以下は CLI 例。

```bash
BUCKET=my-goblog-backups
REGION=ap-northeast-1

# 作成（us-east-1 の場合のみ --create-bucket-configuration を省く）
aws s3api create-bucket --bucket "$BUCKET" --region "$REGION" \
  --create-bucket-configuration LocationConstraint="$REGION"

# Block Public Access（4項目すべて）
aws s3api put-public-access-block --bucket "$BUCKET" \
  --public-access-block-configuration \
  BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true

# Default encryption（SSE-S3 / AES256）
aws s3api put-bucket-encryption --bucket "$BUCKET" \
  --server-side-encryption-configuration \
  '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'

# Versioning（上書き・誤削除・改ざんへの保険）
aws s3api put-bucket-versioning --bucket "$BUCKET" \
  --versioning-configuration Status=Enabled

# Lifecycle（現行30日・旧版7日・未完了MPU7日中止）
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

> 長期保管したい場合は Lifecycle を「30 日で `GLACIER_IR` 移行 → 365 日で失効」に変更する。

### 3. 最小権限の IAM ユーザーを作成

Lightsail は EC2 のような IAM インスタンスロールに非対応なので、プログラム用アクセスキーを持つ IAM ユーザーが必要。権限は **S3 への PutObject** と **CloudWatch への PutMetricData** のみに絞る（漏洩しても既存バックアップの列挙・取得・削除は不可。CloudWatch も指定 namespace への送信のみ）。

まずポリシーを `policy.json` として保存:

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

> `cloudwatch:PutMetricData` は resource-level 制限に非対応（`Resource: "*"` 必須）なので、`Condition` で namespace を縛って最小権限化する。

ユーザー作成・ポリシー付与・アクセスキー発行:

```bash
aws iam create-user --user-name goblog-backup
aws iam put-user-policy --user-name goblog-backup \
  --policy-name goblog-backup --policy-document file://policy.json
aws iam create-access-key --user-name goblog-backup
# -> 出力の AccessKeyId / SecretAccessKey を控える（シークレットはこの一度しか表示されない）
```

発行したアクセスキー ID / シークレットを後の手順で env ファイルに記入する。

### 4. 監視（CloudWatch Alarm + SNS）

AWS 完結の dead-man's switch を作る。**SNS（通知役）と CloudWatch Alarm（検知役）はセット**で必要。

```bash
# (a) 通知先 SNS トピックを作成し、メールを購読（届いた確認メールで Confirm する）
TOPIC_ARN=$(aws sns create-topic --name goblog-backup-alerts --query TopicArn --output text)
aws sns subscribe --topic-arn "$TOPIC_ARN" --protocol email \
  --notification-endpoint you@example.com

# (b) 「24h 以内に成功 heartbeat が来ない／失敗(0)が来た」で発報するアラーム。
#     TreatMissingData=breaching が「そもそも走らなかった」を捉える肝。
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

> メトリクスは `--unit Count`、period は 1 日（86400 秒）。`Minimum < 1` なので「失敗(0)」でも「データ点なし」でも発報し、復旧（次回成功で 1）すると OK 通知が届く。リージョンは env と揃えること（SNS/CloudWatch/メトリクスは同一リージョン）。

### 5. 設定ファイルを配置

```bash
sudo install -D -o goblog -g goblog -m 600 deploy/backup.env.example /etc/goblog/backup.env
sudo nano /etc/goblog/backup.env   # S3_BUCKET / AWS キー / リージョンを記入
```

### 6. スクリプトと unit を配置・有効化

```bash
# スクリプトは make deploy で /opt/goblog/bin/backup-db.sh に入る
make deploy

# unit を配置
sudo cp deploy/goblog-backup.service deploy/goblog-backup.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now goblog-backup.timer
```

### 7. 動作確認（ドライラン）

```bash
sudo systemctl start goblog-backup.service          # 手動実行
journalctl -u goblog-backup --no-pager -n 30        # ログ確認（"backup uploaded: ..."）
aws s3 ls s3://my-goblog-backups/goblog/ --recursive # オブジェクトが出るか
systemctl list-timers goblog-backup.timer           # 次回発火時刻

# 監視の確認: env を読み込んだ上で意図的に失敗させ、メトリクス 0 を送る
# （数分内にアラーム→メールが届けば dead-man's switch は機能している）
sudo bash -c 'set -a; . /etc/goblog/backup.env; set +a; DB_PATH=/nonexistent /opt/goblog/bin/backup-db.sh'; echo "exit=$?"
# CloudWatch 上のメトリクスを確認（成功/失敗の値が記録されているか）
# 注: `date -u -d '1 day ago'` は GNU date 前提（本番 Ubuntu で実行する想定）。
#     macOS/BSD で試す場合は `date -u -v-1d +%FT%TZ` に置き換える。
aws cloudwatch get-metric-statistics --namespace Goblog/Backup --metric-name BackupSuccess \
  --start-time "$(date -u -d '1 day ago' +%FT%TZ)" --end-time "$(date -u +%FT%TZ)" \
  --period 86400 --statistics Minimum Maximum
```

---

## 復元手順

> **「復元したことのないバックアップ」はバックアップではない。** 導入後に一度、本番とは別のディレクトリで通しで試すこと。

```bash
# 1. 対象を取得
aws s3 cp s3://my-goblog-backups/goblog/2026/06/goblog-<ts>.db.gz /tmp/
gunzip /tmp/goblog-<ts>.db.gz

# 2. 健全性を確認（必ず "ok"）
sqlite3 /tmp/goblog-<ts>.db 'PRAGMA integrity_check;'

# 3. 本番へ反映（停止 → 差し替え → 起動）
sudo systemctl stop goblog
sudo install -o goblog -g goblog -m 644 /tmp/goblog-<ts>.db /var/lib/goblog/goblog.db
sudo systemctl start goblog
```

---

## 運用上の注意

- **単一インスタンス前提**。複数インスタンス化した場合は 1 台だけで実行すること（重複アップロードは無害だがコスト増）。
- **保管コスト**: 個人ブログの SQLite は数 MB 程度。gzip 後はさらに小さく、30 日分でも S3 料金は月数円〜数十円。
- **RPO を秒単位にしたくなったら**: Litestream（SQLite→S3 の連続レプリケーション、OSS・無料）への移行を検討。ただし WAL モード化（`internal/db/db.go` の DSN 変更）と常駐プロセスの運用が前提になる。
- **保険の二重化**: Lightsail の自動スナップショット（ディスク丸ごと日次、GUI から 1 クリック）も併用すると、コード不要の追加の安全網になる。
