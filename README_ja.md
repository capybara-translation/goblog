# goblog
[![CI](https://github.com/capybara-translation/goblog/actions/workflows/ci.yml/badge.svg)](https://github.com/capybara-translation/goblog/actions/workflows/ci.yml)
goblog is a simple blog system written in Go.

## 開発環境のセットアップ

### 1. リポジトリのクローン

```bash
git clone https://github.com/capybara-translation/goblog.git
cd goblog
```

### 2. 依存関係のインストール

```bash
make deps
# または
go mod download
```

### 3. 環境変数の設定（オプション）

開発環境では `.env` ファイルで環境変数を管理できます：

```bash
# .env.example をコピーして .env を作成
cp .env.example .env

# .env ファイルを編集して設定をカスタマイズ
# 例：ポート番号やブログタイトルを変更
```

**注意:** `.env` ファイルは `.gitignore` に含まれているため、Git にコミットされません。

### 4. データベースのセットアップとテストデータ投入

```bash
# データベースをリセットしてテストデータ投入
make reset

# または個別に実行
make clean  # データベースを削除
make seed   # テストデータを投入
```

これにより、動作確認用のテストユーザーとテスト記事が作成されます。

**テストユーザー:**
- ユーザー名: `admin`
- パスワード: `password`

### 5. サーバーの起動

**方法1: .env ファイルを使用（推奨）**

```bash
# .env ファイルを作成して設定
cp .env.example .env
# .env ファイルを編集して設定をカスタマイズ

# サーバーを起動（.env から自動的に読み込まれます）
make run
# または
go run cmd/goblog/main.go
```

**方法2: 環境変数を直接指定**

```bash
# 環境変数を指定して起動
PORT=8000 BLOG_TITLE="開発ブログ" go run cmd/goblog/main.go
```

ブラウザで http://localhost:8080 にアクセスして確認できます。


**利用可能な環境変数:**

| 環境変数 | 説明 | デフォルト値 |
|---------|------|-------------|
| `PORT` | サーバーのポート番号 | `8080` |
| `SECURE_COOKIE` | Cookie の Secure フラグ（HTTPS環境では `true` に設定） | `false` |
| `PASSWORD_POLICY` | パスワードポリシー（`NONE` または `STRONG`） | `NONE` |
| `TRUSTED_PROXIES` | 信頼するプロキシのIP/CIDR（カンマ区切り、X-Forwarded-For用） | *（空）* |
| `DATABASE_PATH` | データベースファイルのパス | `data/goblog.db` |
| `BLOG_TITLE` | ブログのタイトル（ヘッダーやページタイトルに表示） | `goblog` |
| `BASE_URL` | サイトのベースURL（サイトマップ等で使用） | `http://localhost:{PORT}` |
| `UPLOAD_DIR` | アップロードファイルの保存先ディレクトリ | `data/uploads` |
| `MAX_UPLOAD_SIZE` | アップロードファイルの最大サイズ（バイト） | `5242880`（5MB） |
| `TZ` | タイムゾーン（例: `Asia/Tokyo`, `UTC`, `America/New_York`）<br>日付表示に使用される | システムのタイムゾーン |

**パスワードポリシーについて:**

- `NONE`: 制限なし（開発/テスト環境向け）
- `STRONG`: 厳格なポリシー（本番環境向け）
  - 最小15文字
  - 大文字を1文字以上含む
  - 小文字を1文字以上含む
  - 数字を1文字以上含む
  - 記号を1文字以上含む

※ 大文字小文字を区別しません（`none`/`NONE`/`None`、`strong`/`STRONG`/`Strong` すべて有効）

**タイムゾーンについて:**

日付表示は ISO 8601 形式（YYYY-MM-DD）にタイムゾーン略称を付けた形式で表示されます：
- 例: `2024-12-26 (JST)`, `2024-12-25 (UTC)`, `2024-12-26 (EST)`

TZ 環境変数を設定することで、ブログの執筆者のタイムゾーンで日付を表示できます：

```bash
# Asia/Tokyo タイムゾーンで起動
TZ=Asia/Tokyo make run

# または .env ファイルに設定
echo "TZ=Asia/Tokyo" >> .env
```

**注意:**
- 本番環境（HTTPS対応サーバー）では必ず `SECURE_COOKIE=true` を設定してください。これにより Cookie が HTTPS 接続でのみ送信されるようになります。
- 本番環境では `PASSWORD_POLICY=STRONG` を設定することを強く推奨します。
- リバースプロキシ（nginx等）の背後で運用する場合、`TRUSTED_PROXIES` にプロキシのIPを設定してください（例: `127.0.0.1`）。未設定の場合、`X-Forwarded-For` ヘッダーは無視され、IPスプーフィング攻撃を防止します。
- **本番環境では `.env` ファイルではなく、システムの環境変数を直接設定することを推奨します。**

### 6. テストの実行

```bash
# 全テストを実行
make test

# 詳細な出力付き
make test-v

# カバレッジを確認
make test-cover
```

## 本番環境へのデプロイ

本番環境ではsystemdでサービスを管理し、nginxでリバースプロキシを構成します。

### 1. サーバーの準備

```bash
# 必要なパッケージをインストール
sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx

# タイムゾーンを設定（日本の場合）
sudo timedatectl set-timezone Asia/Tokyo

# 設定を確認
timedatectl

# goblog用のユーザーとディレクトリを作成
sudo useradd -r -s /bin/false goblog
sudo mkdir -p /opt/goblog/bin
sudo mkdir -p /var/lib/goblog/uploads
sudo chown -R goblog:goblog /var/lib/goblog
```

**利用可能なタイムゾーン一覧を確認:**
```bash
timedatectl list-timezones | grep -i tokyo
```

### 2. バイナリのビルドとデプロイ

```bash
# Go、ビルドツールをインストール
sudo apt install -y golang-go build-essential git

# Node.js v24 をインストール（NodeSource経由）
curl -fsSL https://deb.nodesource.com/setup_24.x | sudo -E bash -
sudo apt install -y nodejs

# リポジトリをクローン
git clone https://github.com/capybara-translation/goblog.git
cd goblog

# すべてをビルド（npm依存インストール、フロントエンドビルド、バックエンドビルド）
make build

# バイナリを配置
sudo mv bin/goblog bin/adduser bin/seed /opt/goblog/bin/
sudo chown root:root /opt/goblog/bin/goblog /opt/goblog/bin/adduser /opt/goblog/bin/seed
```

**注**: マイグレーションファイル、テンプレート、静的ファイルはバイナリに埋め込まれているため、別途コピーする必要はありません。

### 3. systemdサービスの設定

```bash
# クローンしたリポジトリ内で作業（~/goblog）
cd ~/goblog

# サービスファイルをコピー
sudo cp deploy/goblog.service /etc/systemd/system/

# 環境変数を編集（ドメイン名やタイトルを変更）
sudo vim /etc/systemd/system/goblog.service

# サービスを有効化して起動
sudo systemctl daemon-reload
sudo systemctl enable goblog
sudo systemctl start goblog

# ステータス確認
sudo systemctl status goblog

# 環境変数を確認
sudo systemctl show goblog --property=Environment

# ログ確認
sudo journalctl -u goblog -f
```

**重要な環境変数:**

| 環境変数 | 本番環境での設定例 |
|---------|-------------------|
| `SECURE_COOKIE` | `true`（必須） |
| `PASSWORD_POLICY` | `STRONG`（推奨） |
| `TRUSTED_PROXIES` | `127.0.0.1`（nginx背後では必須） |
| `BASE_URL` | `https://your-domain.com` |
| `DATABASE_PATH` | `/var/lib/goblog/goblog.db` |
| `UPLOAD_DIR` | `/var/lib/goblog/uploads` |
| `BLOG_TITLE` | 任意のブログタイトル |

### 4. nginxの設定

```bash
# クローンしたリポジトリ内で作業（~/goblog）
cd ~/goblog

# 設定ファイルをコピーしてドメイン名を変更
sudo cp deploy/nginx.conf /etc/nginx/sites-available/goblog
sudo vim /etc/nginx/sites-available/goblog  # example.com を実際のドメインに変更

# サイトを有効化
sudo ln -s /etc/nginx/sites-available/goblog /etc/nginx/sites-enabled/

# デフォルトサイトを無効化（任意）
sudo rm /etc/nginx/sites-enabled/default

# 設定をテスト
sudo nginx -t

# nginxを再起動
sudo systemctl reload nginx
```

### 5. SSL証明書の取得（Let's Encrypt）

```bash
# certbotでSSL証明書を取得
sudo certbot --nginx -d your-domain.com -d www.your-domain.com

# 自動更新の確認
sudo certbot renew --dry-run
```

### 6. 管理者ユーザーの作成

```bash
# サーバーでユーザーを作成（adduserはステップ2で配置済み）
cd /opt/goblog
sudo -u goblog PASSWORD_POLICY=STRONG DATABASE_PATH=/var/lib/goblog/goblog.db ./bin/adduser
```

### 7. 動作確認

```bash
# ヘルスチェック
curl https://your-domain.com/api/v1/health

# サイトマップ確認
curl https://your-domain.com/sitemap.xml
```

### トラブルシューティング

```bash
# goblogのログを確認
sudo journalctl -u goblog -n 100

# goblogのログをリアルタイムで確認
sudo journalctl -u goblog -f

# nginxのエラーログを確認
sudo tail -f /var/log/nginx/goblog_error.log
```

### サービスの再起動

| 変更内容          | 必要なコマンド |
|---------------|---------------|
| Unitファイル      | `sudo systemctl daemon-reload && sudo systemctl restart goblog` |
| バイナリファイル      | `sudo systemctl restart goblog` |
| 環境変数（Unit内） | `sudo systemctl daemon-reload && sudo systemctl restart goblog` |
| nginx設定     | `sudo nginx -t && sudo systemctl reload nginx` |

```bash
# Unitファイルまたは環境変数を変更した場合
sudo systemctl daemon-reload
sudo systemctl restart goblog

# バイナリのみ更新した場合
sudo systemctl restart goblog

# nginx設定を変更した場合（設定テスト後にリロード）
sudo nginx -t && sudo systemctl reload nginx
```

### 既存画像のWebP派生バックフィル（初回のみ）

`make deploy` は `regenerate-variants` バイナリを配置するだけで、**自動実行はしません**。新規アップロードはアップロードハンドラが派生 WebP をその場で生成しますが、派生画像パイプラインが導入される前にアップロード済みの画像（および将来新しい幅を追加した時の既存画像）は、手動で 1 回だけバックフィルする必要があります。

```bash
# 実行中のサービスが使っている UPLOAD_DIR を確認する（goblog.service の
# Environment= 行で設定されている値）。CLI には同じディレクトリを渡す。
sudo systemctl show goblog -p Environment

# 確認した値と同じ UPLOAD_DIR を渡して実行。WalkDir でサブディレクトリ
# まで降りるため、/uploads/ogp-cache/ のリンクカード画像もバックフィル
# 対象に含まれる。
sudo UPLOAD_DIR=/opt/goblog/data/uploads /opt/goblog/bin/regenerate-variants
```

このコマンドは idempotent（`-480w.webp` / `-800w.webp` / `-1200w.webp` 派生がすでに揃っている画像はスキップ）なので、途中で失敗したり、後から再デプロイした後でも安全に再実行できます。終了コードが非ゼロのときは少なくとも 1 ファイル失敗しているので、stderr に出ている `FAIL` 行を確認してください。

バックフィル後の `goblog` 再起動は不要です。レンダラの派生キャッシュは画像ごとに初回リクエスト時に作られるため、バックフィル完了後の次のページ閲覧から新しい派生が自動的に使われます。

## 利用可能なMakeコマンド

開発でよく使うコマンドをMakefileにまとめています：

### 開発用コマンド

```bash
make help        # ヘルプを表示
make run         # サーバーを起動（開発用）
make stop        # 起動中のサーバーを停止
make test        # テストを実行
make test-v      # テストを詳細出力で実行
make test-cover  # テストカバレッジを表示
make clean       # データベースと管理者用SPAビルド成果物を削除
make seed        # テストデータを投入
make reset       # データベースをリセットしてテストデータ投入（管理者用SPAも再ビルド）
make deps        # Go依存関係をダウンロード
make adduser     # 管理者ユーザーを追加
```

### ビルドコマンド（本番用）

```bash
make build        # 本番用に全てビルド（依存インストール、フロントエンドビルド、バックエンドビルド）
make install      # npm依存関係をインストール（Tailwind CLI）
make install-admin # 管理者用SPAのnpm依存関係をインストール
make build-admin  # 管理者用SPAをビルド
make build-css    # 公開ページ用Tailwind CSSをビルド
```

### 管理者用SPA開発

```bash
make dev-admin    # 管理者用SPAの開発サーバーを起動
make clean-admin  # 管理者用SPAのビルド成果物を削除
```


## ディレクトリ構成

```
/cmd/
  /adduser/main.go     # 管理者ユーザー追加コマンド
  /goblog/main.go      # アプリケーション本体
  /seed/main.go        # テストデータ投入コマンド

/deploy/               # デプロイ設定
  goblog.service       # systemd Unitファイル
  nginx.conf           # nginx設定ファイル

/internal/
  /auth/               # 認証ユーティリティ
    session.go         # セッション管理
  /config/             # 設定管理
    config.go          # 環境変数からの設定読み込み
  /db/                 # データベース
    db.go              # DB接続・マイグレーション
  /domain/             # ドメインモデル
    post.go            # 記事モデル
    user.go            # ユーザーモデル
  /http/               # HTTPレイヤー
    router.go          # ルーティング設定
    middleware.go      # 認証・CSRFミドルウェア
    handlers_admin.go  # 管理者用SPA配信
    handlers_api.go    # API ハンドラー
    handlers_auth.go   # 認証ハンドラー
    handlers_image.go  # 画像アップロードハンドラー
    handlers_public.go # 公開ページハンドラー
    handlers_sitemap.go # サイトマップハンドラー
  /markdown/           # Markdown処理
    markdown.go        # Markdown→HTML変換
    dataline_extension.go # 行番号付与拡張
  /repo/               # データアクセス層
    post_repo.go       # 記事リポジトリ
    user_repo.go       # ユーザーリポジトリ
  /service/            # ビジネスロジック層
    auth_service.go    # 認証サービス
    post_service.go    # 記事サービス
  /view/               # ビュー関連
    /static/           # 静的ファイル
      markdown.css     # Markdownスタイル
    /templates/        # HTMLテンプレート
      layout.html      # 共通レイアウト
      home.html        # トップページ
      posts.html       # 記事一覧
      post.html        # 記事詳細
      tags.html        # タグ一覧
      tag_posts.html   # タグ別記事一覧
      notfound.html    # 404ページ

/migrations/           # SQLマイグレーション
  001_create_posts.sql # 記事テーブル
  002_create_users.sql # ユーザーテーブル
  003_add_is_pinned.sql # ピン留め機能

/web-admin/            # 管理者用SPA（React）
  /src/
    /api/              # APIクライアント
    /components/       # 共通コンポーネント
    /hooks/            # カスタムフック
    /pages/            # ページコンポーネント
    /mocks/            # テスト用モック（MSW）
    /utils/            # ユーティリティ
    App.tsx            # ルートコンポーネント
    main.tsx           # エントリポイント
```

## バックグラウンドプロセスとキャッシュのライフサイクル

goblogはサーバー起動時に複数のバックグラウンドgoroutineを起動し、セッションやキャッシュの期限切れデータを自動的にクリーンアップします。

### セッション管理

```
ログイン → セッション作成（インメモリ）→ 24時間後に期限切れ → 自動削除
```

| 項目 | 値 |
|------|------|
| 保存先 | インメモリ（`sync.RWMutex`で排他制御） |
| TTL | 24時間 |
| 自動クリーンアップ | 1時間ごとに期限切れセッションを削除 |
| 手動削除 | ログアウト時 |

**注意:** インメモリのため、サーバー再起動で全セッションが失われます（再ログインが必要）。

### ログイン試行の追跡（ブルートフォース対策）

```
ログイン失敗 → IP別にカウント → 段階的遅延 → ログイン成功でリセット
```

| 項目 | 値 |
|------|------|
| 保存先 | インメモリ（IP別） |
| 遅延 | 3回失敗→2秒、5回→5秒、10回→30秒 |
| 自動クリーンアップ | 10分ごとに、最終失敗から30分以上経過した記録を削除 |
| リセット | ログイン成功時 |

### OGPキャッシュ（リンクカード用）

```
URL → OGPメタデータ取得（5秒タイムアウト）→ 画像ダウンロード → SQLiteに保存
                                                ↓
                                     {UPLOAD_DIR}/ogp-cache/に保存
                                                ↓
                                          7日後に期限切れ
                                                ↓
                               再フェッチ → 古い画像を削除 → 新しいデータで更新
```

| 項目 | 値 |
|------|------|
| 保存先 | SQLite（`ogp_cache`テーブル） |
| 成功時TTL | 7日間 |
| エラー時TTL | 1時間 |
| 画像の保存先 | `{UPLOAD_DIR}/ogp-cache/`（UUID v4ファイル名） |
| 画像の最大サイズ | 2MB |
| 自動クリーンアップ | 1時間ごとに期限切れエントリとローカル画像を削除 |

**OGPキャッシュのポイント:**
- 外部サービスのレート制限（HTTP 429）を回避するため、OGP画像をローカルにキャッシュ
- キャッシュにヒットしてもローカル画像がない場合は、その場でダウンロード
- 画像ダウンロードに失敗しても、OGPデータ自体は正常に返却（外部URLにフォールバック）
- SSRF対策として、プライベートIPへのリクエストはブロック

### バックグラウンドgoroutineの一覧

| goroutine | 間隔 | 処理内容 | 起動元 |
|-----------|-------|---------|--------|
| セッションクリーンアップ | 1時間 | 期限切れセッションを削除 | `auth.NewInMemorySessionStore()` |
| ログイン試行クリーンアップ | 10分 | 古いログイン失敗記録を削除 | `service.NewAuthService()` |
| OGPキャッシュクリーンアップ | 1時間 | 期限切れOGPエントリとローカル画像を削除 | `service.NewOGPService()` |

## URL設計

### 公開ページ

- `GET /` - トップページ
- `GET /posts` - 記事一覧（ページネーション対応）
- `GET /posts/{slug}` - 記事詳細
- `GET /tags` - タグ一覧
- `GET /tags/{tag}` - タグ別記事一覧（ページネーション対応）
- `GET /sitemap.xml` - サイトマップ

### 静的ファイル 

- `GET /static/*` - CSS等の静的ファイル
- `GET /uploads/*` - アップロードされた画像

### 管理者用SPA

- `GET /admin` - SPA入口
- `GET /admin/*` - SPAフォールバック（クライアントサイドルーティング対応）

### API（/api/v1）

**公開エンドポイント:**
- `GET /api/v1/health` - ヘルスチェック
- `POST /api/v1/auth/login` - ログイン

**保護エンドポイント（認証 + CSRF 必須）:**
- `POST /api/v1/auth/logout` - ログアウト
- `GET /api/v1/auth/me` - ログイン状態確認
- `GET /api/v1/posts` - 記事一覧取得（`?status=draft|published&tag=タグ名&limit=N&offset=N`）
- `POST /api/v1/posts` - 記事作成
- `GET /api/v1/posts/{id}` - 記事取得
- `PUT /api/v1/posts/{id}` - 記事更新
- `DELETE /api/v1/posts/{id}` - 記事削除
- `POST /api/v1/posts/{id}/publish` - 記事公開
- `POST /api/v1/posts/{id}/unpublish` - 記事非公開化
- `POST /api/v1/posts/{id}/pin` - 記事をピン留め
- `POST /api/v1/posts/{id}/unpin` - 記事のピン留め解除
- `GET /api/v1/tags` - タグ一覧取得（`?status=draft|published`）
- `POST /api/v1/markdown/preview` - Markdownプレビュー
- `POST /api/v1/images` - 画像アップロード
