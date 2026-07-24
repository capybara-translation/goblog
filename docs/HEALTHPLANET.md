# Health Planet 連携（体重・血圧データ同期）

タニタ Health Planet API から体重・体脂肪率・血圧・脈拍を毎日 0:00 JST に取得し、goblog の `health_records` テーブルへ冪等 upsert する仕組み。30 日窓での取得と `UNIQUE(measured_at, metric)` による upsert により、手入力の遅延登録（翌日以降のバックフィル）も次回同期で自動吸収される。

機能全体は `HEALTHPLANET_ENABLED` 環境変数でゲートされる。未設定（デフォルト: 無効）の場合、日次同期は no-op（exit 0）、管理画面 UI は非表示、OAuth 系 API ルートは未登録。

## 構成

```
管理画面 (/admin/healthplanet)
  └─ 「連携する」→ Health Planet 認可 → /admin/healthplanet/success?code=
       └─ 「連携を完了する」→ POST /api/v1/healthplanet/exchange
            └─ トークン交換 → healthplanet_tokens に保存

systemd timer (毎日 0:00 JST、常設)
  └─ goblog-hpsync.service (oneshot)
       └─ /opt/goblog/bin/hpsync run
            0. HEALTHPLANET_ENABLED != true なら skip (exit 0)
            1. healthplanet_tokens からトークンをロード
            2. リフレッシュ → expires_at を更新保存
            3. innerscan / sphygmomanometer を直近30日窓で取得
            4. health_records に upsert（UNIQUE(measured_at, metric)）
            5. 失敗・トークン失効間近は exit≠0 → journal / CloudWatch
```

OAuth 認可フローの redirect URI は `{BASE_URL}/admin/healthplanet/success`。「連携を完了する」ボタンを明示的に押すまで code を交換しないのは、攻撃者の code をブログ管理者に踏ませる紐付け攻撃（code injection）を防ぐ意図による。

関連ファイル:

| ファイル | 役割 |
|---|---|
| `cmd/hpsync/main.go` | CLI エントリポイント（`run` / `auth` サブコマンド） |
| `internal/healthplanet/client.go` | Health Planet API クライアント（OAuth・innerscan・sphygmomanometer） |
| `internal/service/health_sync_service.go` | 日次同期ロジック（トークン更新・取得・upsert・失効警告） |
| `internal/service/health_planet_admin_service.go` | 管理画面向け OAuth フロー（認可 URL 生成・code 交換・状態確認） |
| `internal/http/handlers_healthplanet.go` | HTTP ハンドラー（status / auth-url / exchange） |
| `internal/repo/health_record_repo.go` | `health_records` テーブルへの upsert |
| `internal/repo/healthplanet_token_repo.go` | `healthplanet_tokens` 1 行テーブルの load / save |
| `web-admin/src/pages/HealthPlanet.tsx` | 管理画面: 連携状態表示・認可フロー開始 |
| `web-admin/src/pages/HealthPlanetSuccess.tsx` | 管理画面: redirect 後の「連携を完了する」画面 |
| `deploy/goblog-hpsync.service` | oneshot サービス unit |
| `deploy/goblog-hpsync.timer` | 日次スケジュール（0:00 JST, Persistent=true） |
| `deploy/healthplanet.env.example` | 設定テンプレート（`/etc/goblog/healthplanet.env` に配置） |
| `scripts/hpsync-run.sh` | CloudWatch dead-man's-switch ラッパー（任意） |

---

## セットアップ手順

### 1. Health Planet でアプリ登録

[https://www.healthplanet.jp/](https://www.healthplanet.jp/) でアプリを **Web アプリケーション** として登録する。ブログのドメインを登録し、redirect URI には `{BASE_URL}/admin/healthplanet/success` を設定する（例: `https://blog.example.com/admin/healthplanet/success`）。取得した client_id と client_secret を控える。

> **注意**: CLI フォールバック (`hpsync auth`) はタニタ自身の `success.html` に redirect するため、blog ドメインとは別に登録する必要はないが、本番運用では管理画面フローに統一することを推奨する。

### 2. 連携設定ファイルを配置（goblog と hpsync の単一ソース）

```bash
sudo install -D -o goblog -g goblog -m 600 \
  deploy/healthplanet.env.example /etc/goblog/healthplanet.env
sudo nano /etc/goblog/healthplanet.env
# HEALTHPLANET_ENABLED / _CLIENT_ID / _CLIENT_SECRET / DATABASE_PATH を記入
```

`/etc/goblog/healthplanet.env` は goblog 本体（`goblog.service` の `EnvironmentFile=-` で読み込み）と hpsync（`goblog-hpsync.service`）の**両方が読む単一ソース**。シークレットのローテーションはこのファイル 1 つの更新で済む。`DATABASE_PATH` は hpsync 用で、goblog 本体側では unit 内の inline `Environment=` 指定が後勝ちで優先されるため、重複していても安全。

```bash
sudo systemctl restart goblog
```

管理画面 `/admin/healthplanet` に「連携する」ボタンが現れれば有効化できている。

> **既存サーバでの注意**: インストール済みの `/etc/systemd/system/goblog.service` に `EnvironmentFile=-/etc/goblog/healthplanet.env` の行が無い場合（この機能より前に構築したサーバ）は、inline の `Environment=` 群より**前**に追記してから `sudo systemctl daemon-reload && sudo systemctl restart goblog` を実行する。リポジトリの `deploy/goblog.service` には追加済み。

### 3. バイナリを配布

```bash
make deploy
```

`make deploy` で `bin/hpsync` と `scripts/hpsync-run.sh` が `/opt/goblog/bin/` に配置される。

### 4. systemd unit を常設

```bash
sudo cp deploy/goblog-hpsync.service deploy/goblog-hpsync.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now goblog-hpsync.timer
```

timer は `HEALTHPLANET_ENABLED != true` のときも常設で問題ない。その場合 `hpsync run` は "skipping" を出力して exit 0 するだけで副作用はない。

### 5. 管理画面から認可

ブラウザで管理画面 `/admin/healthplanet` を開く:

1. 「連携する」をクリック → Health Planet の認可画面にリダイレクトされる
2. Health Planet でログイン・アクセス許可を承認する
3. ブログの `/admin/healthplanet/success?code=…` にリダイレクトされる
4. 「連携を完了する」ボタンをクリック → `POST /api/v1/healthplanet/exchange` が呼ばれトークンが保存される

管理画面に「最終リフレッシュ: …」が表示されれば認可完了。

---

## 動作確認

管理画面から初回認可が完了したら、すぐに手動で同期を走らせてリフレッシュが正常に動作することを確認する。`journalctl` に "token refresh failed" の警告が出ていなければ、redirect_uri の一致検証は問題ない。

```bash
# 手動で同期を実行
sudo systemctl start goblog-hpsync.service

# ログを確認（"Sync complete." が出ていれば成功。"token refresh failed" が出ていた場合は
# redirect_uri の不一致が疑われるため、管理画面から再認可して redirect_uri を揃えること）
journalctl -u goblog-hpsync -n 20

# timer の次回発火時刻
systemctl list-timers goblog-hpsync.timer

# DB に記録されているか確認
sqlite3 /var/lib/goblog/goblog.db \
  'SELECT metric, COUNT(*) FROM health_records GROUP BY metric;'
```

成功すると管理画面 `/admin/healthplanet` の「トークン最終リフレッシュ」タイムスタンプも更新される（`healthplanet_tokens.updated_at`）。

---

## 監視（任意）

BACKUP.md の CloudWatch dead-man's-switch と同じパターンで、`hpsync run` の成否を監視できる。

**IAM ポリシー**: `cloudwatch:PutMetricData` を namespace `Goblog/HPSync` に限定（手順の詳細は `docs/BACKUP.md` の「最小権限の IAM ユーザー」を参照、namespace / metric を読み替えること）。

**設定方法**:

1. `/etc/goblog/healthplanet.env` に CloudWatch 認証情報を追記:

   ```bash
   CW_NAMESPACE=Goblog/HPSync
   AWS_ACCESS_KEY_ID=...
   AWS_SECRET_ACCESS_KEY=...
   AWS_DEFAULT_REGION=ap-northeast-1
   ```

2. `goblog-hpsync.service` の `ExecStart` を wrapper に切り替え:

   ```ini
   # ExecStart=/opt/goblog/bin/hpsync run
   ExecStart=/opt/goblog/bin/hpsync-run.sh
   ```

   ```bash
   sudo systemctl daemon-reload
   ```

3. SNS トピックと CloudWatch Alarm を作成（BACKUP.md の「監視」手順を参照。namespace を `Goblog/HPSync`、metric を `SyncSuccess` に読み替える）。

> **重要**: `HEALTHPLANET_ENABLED != true` のままアラームを設定しないこと。無効運用中の `hpsync run` は exit 0 で終了するため `SyncSuccess=1` が継続して報告され、アラームは発報しない（正しく見えてしまう）。監視はサービスが実際に稼働してから有効化する。

---

## 再認可（復旧手順）

トークンが失効または認可が切れた場合は再認可が必要。ログには以下のどちらかが出る:

- `Error: healthplanet re-authorization required: ...` — refresh に失敗しトークンも期限切れ
- `Error: healthplanet token expiring soon: ...` — refresh が有効期間を延長しなかった警告（7 日以内に失効）

**通常の再認可（管理画面が使える場合）:**

管理画面 `/admin/healthplanet` の「再認可する」をクリックし、手順「5. 管理画面から認可」を再度行う。

**CLI フォールバック（管理画面が使えない場合）:**

```bash
sudo -u goblog bash -c 'set -a; . /etc/goblog/healthplanet.env; set +a; /opt/goblog/bin/hpsync auth'
```

表示された URL をブラウザで開き、Health Planet で許可後に address bar の `code=` パラメータをコピーしてプロンプトに貼り付ける。タニタの `success.html` に redirect されるため、blog ドメインへのアクセスは不要（サーバへの接続が SSH のみでも完結する）。

---

## 運用上の注意

- **レート制限**: Health Planet API は 60 リクエスト/時。日次同期は innerscan + sphygmomanometer の 2 コール、管理画面の操作を加えても余裕がある。連続で手動実行を繰り返すと上限に近づく場合があるので注意。
- **同期窓外のデータ**: 取得窓は直近 30 日。それより古い過去データは自動同期されない。初期データが必要な場合は Health Planet の手動エクスポートを使い、DB に直接 insert する。
- **トークンの平文保存**: `healthplanet_tokens` テーブルにアクセストークン・リフレッシュトークンが平文で保存される。DB は S3 バックアップに含まれるため、バックアップも同様に平文トークンを含む点に留意すること（バックアップへのアクセス制御は docs/BACKUP.md 参照）。
- **トークン有効期限**: 30 日。日次同期で毎回リフレッシュするため通常は自動延長されるが、API 仕様上ローテーションなし（verified 2026-07 実機検証）。リフレッシュが有効期間を延長しない状況が続くと 7 日前から journal にエラーが出る（dead-man's switch 設定済みの場合はメールで通知される）。その場合は早めに再認可すること。
- **ブログ上の表示**: 現時点では `health_records` に蓄積されるのみで、ブログの公開ページへの表示は未実装。
