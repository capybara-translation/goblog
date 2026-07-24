# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Claude Code must follow the following rules:

- Always use Context7 MCP when I need library/API documentation, code generation, setup or configuration steps without me having to explicitly ask.

## プロジェクト概要

goblogはGoで書かれたシンプルなブログシステムです。公開ページはSSR（サーバーサイドレンダリング）、管理画面はReact SPAとして構成されています。

### 実装状況

✅ **実装済み:**
- 記事管理（作成、編集、削除、公開/非公開切替、ピン留め）
- タグ機能（タグ一覧、タグ別記事一覧、タグフィルタリング）
- 認証・認可（ユーザー名・パスワード認証、セッション管理）
- セキュリティ（CSRF対策、ブルートフォース対策、パスワードポリシー）
- タイムゾーン対応（ISO 8601形式 + タイムゾーン略称）
- 公開ページ（SSR）
- React SPA 管理画面（/admin）
- API エンドポイント（/api/v1）
- サイトマップ（/sitemap.xml）
- 画像アップロード（複数ファイル対応、メタデータ除去、マジックバイト検証）
- OGPリンクカード（外部URLのOGP情報取得・キャッシュ・ローカル画像保存）
- 閲覧回数トラッキング（ボットフィルタリング、IP+UA重複排除）
- Markdownプレビュー（サーバーサイドレンダリング、同期スクロール）
- Remember me（短命セッション+長命 remember token、SQLite 保存、SHA-256 ハッシュ、selector+raw 分離、記事を表示する公開ページ（トップ / 記事詳細 / タグ別一覧）と `/auth/me` で自動復元）
- 記事リアクション（匿名読者が複数絵文字でリアクション、1記事・1絵文字につき1回、Cookie 重複防止、件数は SSR + reacted 状態は JS で付与。記事詳細・トップ・タグ別一覧で表示し、その場でトグル可能。絵文字の解釈は読者に委ねるため label は UI 非表示）。管理画面 (`/admin/reactions`) で絵文字マスタを作成・編集・有効/無効・条件付き物理削除(件数0かつ非seedのみ)できる。
- ログイン中の端末管理（管理画面 `/admin/devices` で remember_token ベースの永続端末一覧＋保持OFFの一時セッションを併記。`mileusna/useragent` で UA から端末/ブラウザを導出、IP 表示。現在の端末に「この端末」バッジを付け個別失効は不可、非現端末の個別失効と「他の端末をすべてログアウト」に対応。remember 端末失効時は紐付くアクティブセッションも巻き添えで失効させ確実にログアウト）
- Health Planet 連携（タニタ Health Planet API から体重・体脂肪率・血圧・脈拍を毎日 0:00 JST に取得し `health_records` へ冪等 upsert。`HEALTHPLANET_ENABLED` でゲート（デフォルト無効・無効時は同期 no-op / 管理画面 UI 非表示 / API ルート未登録）。OAuth 認可は管理画面 `/admin/healthplanet` から（redirect → `/admin/healthplanet/success` → 明示ボタンで code 交換。自動交換しないのは攻撃者の code を踏ませる紐付け攻撃の防止）。日次同期は `cmd/hpsync run` + systemd timer 常設。CLI フォールバック `hpsync auth`（success.html + コピペ）あり。30日窓取得で手入力の遅延登録も吸収。トークンは DB 1行テーブルに平文保存・30日有効・リフレッシュでローテーションなし（実機検証済み）・失効7日前から exit≠0 で警告。手順書: docs/HEALTHPLANET.md。ブログ上での表示は未実装）

🚧 **計画中:**
- RSS フィード
- Tier 4: CSS 最適化（render-blocking 解消 + 未使用 Tailwind CSS の除去）。Tier 1-3 の画像最適化で LCP 94.2s → 10.6s まで改善したが、Lighthouse Performance score は依然 56。残る支配要因は HTML/CSS の初期配信時間 (FCP 9.8s) で、`render-blocking-insight` で 8.9s、`unused-css-rules` で 237 KiB の改善余地あり。Tailwind の `--minify` + `--content` による purge で Perf score 90+ 到達見込み

## アーキテクチャ

クリーンアーキテクチャを採用し、レイヤー間の責務を明確に分離しています：

```
HTTP Layer (handlers)
  ↓
Service Layer (business logic)
  ↓
Repository Layer (data access)
  ↓
Database (SQLite)
```

**重要な設計パターン：**
- **Repository Pattern**: データアクセスをインターフェースで抽象化
- **Service Pattern**: ビジネスロジックをサービス層に集約
- **Dependency Injection**: コンストラクタベースの依存注入でテスタビリティを確保
- **Middleware Pattern**: 認証・CSRF対策をミドルウェアで実装

## ディレクトリ構成

```
/cmd/
  /goblog/main.go      # アプリケーションエントリポイント
  /seed/main.go        # テストデータ投入コマンド
  /adduser/main.go     # 管理ユーザー追加コマンド
  /hpsync/main.go      # Health Planet 同期 CLI（run / auth サブコマンド）

/internal/
  /http/               # HTTPレイヤー
    router.go          # ルーティング設定
    handlers_public.go # 公開ページハンドラー
    handlers_api.go    # API ハンドラー（記事CRUD）
    handlers_auth.go   # 認証ハンドラー
    handlers_image.go  # 画像アップロードハンドラー
    handlers_sitemap.go # サイトマップハンドラー
    handlers_admin.go  # SPA配信ハンドラー
    middleware.go      # 認証・CSRFミドルウェア
    metadata_stripper.go # 画像メタデータ除去
    ogp_meta.go        # OGPメタタグ生成
    handlers_reaction.go  # リアクション公開APIハンドラー
    handlers_reaction_admin.go # リアクション種別管理APIハンドラー（認証+CSRF）
    handlers_device.go    # ログイン中の端末一覧・失効ハンドラー（認証+CSRF）
    handlers_healthplanet.go # Health Planet 管理APIハンドラー（status / auth-url / exchange）
    reaction_middleware.go # X-Requested-With 検証ミドルウェア
    client_ip.go       # 信頼プロキシ考慮の client IP 抽出（共有）
    ratelimiter.go     # IP 単位レートリミッタ
  /service/            # ビジネスロジック
    post_service.go
    post_view_service.go # 閲覧回数（ボットフィルタ・重複排除）
    auth_service.go
    ogp_service.go     # OGPリンクカード
    reaction_service.go  # リアクションのビジネスロジック
    reaction_type_service.go # リアクション種別のビジネスロジック（seed 保護・バリデーション）
    device_service.go    # ログイン中の端末のビジネスロジック（一覧マージ・失効・UAパース）
    health_sync_service.go   # 日次同期ロジック（トークン更新・取得・upsert・失効警告）
    health_planet_admin_service.go # 管理画面向け OAuth フロー（認可 URL 生成・code 交換・状態確認）
  /repo/               # データアクセス
    post_repo.go
    post_view_repo.go      # 閲覧記録
    user_repo.go
    ogp_repo.go            # OGPキャッシュ
    remember_token_repo.go # Remember me トークンの SQLite 実装
    reaction_repo.go       # リアクションの集計・記録
    reaction_type_repo.go  # リアクション種別 CRUD（FindAll/FindByID/Create/Update/DeleteIfUnused）
    reaction_seed.go       # DefaultReactionTypes / SeedReactionTypes / IsSeedEmoji（単一ソース）
    health_record_repo.go     # health_records upsert
    healthplanet_token_repo.go # healthplanet_tokens の load / save（1 行テーブル）
  /domain/             # ドメインモデル
    post.go
    user.go
  /auth/               # 認証ユーティリティ（セッション管理）
    session.go
    remember_token.go        # RememberTokenStore interface、暗号ユーティリティ、cookie コーデック、cleanup goroutine（SQLite 実装は /repo/remember_token_repo.go）
  /config/             # 設定管理
    config.go
  /markdown/           # Markdown変換
    converter.go
  /ogp/                # OGPフェッチャー
    fetcher.go
  /healthplanet/       # Health Planet API クライアント
    client.go          # OAuth・innerscan・sphygmomanometer フェッチャー
  /view/               # ビュー関連
    templates/         # HTMLテンプレート
      layout.html
      home.html
      post.html
      tags.html
      tag_posts.html
      notfound.html
    static/
      js/
        reactions.js   # リアクションボタンの JS（reacted 状態付与・トグル）

initschema.go          # InitSchema: マイグレーション + reaction seed を一括実行（cmd/* から呼び出し）

/migrations/           # SQLマイグレーションファイル
  001_create_posts.sql
  002_create_users.sql
  003_add_is_pinned.sql
  004_create_ogp_cache.sql
  005_add_ogp_local_image.sql
  006_add_post_views.sql
  008_create_reactions.sql
  011_create_health_records.sql # health_records + healthplanet_tokens テーブル

/web-admin/            # React SPA 管理画面
  /src/
    /pages/            # PostList, PostEdit, Login, ReactionTypeList
      ReactionTypeList.tsx # リアクション種別管理画面（テーブル + モーダル CRUD）
      HealthPlanet.tsx     # Health Planet 連携状態・認可フロー開始画面
      HealthPlanetSuccess.tsx # redirect 後の「連携を完了する」画面
    /components/       # MarkdownEditor, TagInput, StatusBadge, etc.
    /api/client.ts     # APIクライアント（CSRF対応）
```

## 開発コマンド

```bash
# 開発サーバー起動（.envから自動読み込み）
make run

# テスト実行
make test              # 全テスト
make test-v            # 詳細出力
make test-cover        # カバレッジ表示
go test ./internal/service  # 特定パッケージのみ

# データベース管理
make clean             # DBファイル削除
make seed              # テストデータ投入（テストユーザー: admin/password）
make reset             # clean + seed

# ビルド・デプロイ
make build             # bin/goblog, bin/seed, bin/adduser を生成
make deploy            # ビルド→/opt/goblog/bin/に配置→systemctl restart（本番用）

# 管理画面（React SPA）
make install-admin     # npm依存インストール
make dev-admin         # 開発サーバー起動（Vite）
make build-admin       # プロダクションビルド
```

## 環境変数

`.env`ファイル（開発時のみ推奨）または環境変数で設定：

- `PORT`: サーバーポート（デフォルト: 8080）
- `SECURE_COOKIE`: Cookie Secure属性（本番: true）
- `PASSWORD_POLICY`: NONE（開発）/STRONG（本番、15文字以上+大小英数記号）
- `TRUSTED_PROXIES`: 信頼するプロキシのIP/CIDR（カンマ区切り、例: `127.0.0.1`）
  - 設定されたIPからのリクエストのみ `X-Forwarded-For` / `X-Real-IP` ヘッダーを信頼する
  - 未設定（デフォルト）: `RemoteAddr` のみ使用（X-Forwarded-For偽装を防止）
  - `X-Forwarded-For` は**右から**評価し、`TRUSTED_PROXIES` に該当するエントリ（自前のプロキシ連鎖）を末尾から剥がして、最初の非信頼エントリ＝実クライアントを採用する。クライアントが偽装できるのは左端（プリペンド）だけなので、偽装値は採用されない（左端を採ると IP 単位のレート制限/ブルートフォースを回避されるため）。各エントリは `net/netip` で正規化（port 除去・IPv6 正規化）し、パース不能（hostname・`unknown`・不正 port 等）はフェイルクローズで `RemoteAddr` にフォールバック。走査ホップ数には上限あり。CDN→nginx 等の多段構成では、各段のレンジを `TRUSTED_PROXIES` に列挙すれば右から順に剥がれる
  - nginx等リバースプロキシ背後で運用する場合は必ず設定すること（nginx 側は `X-Forwarded-For` を追記する標準設定 `$proxy_add_x_forwarded_for` でよい。アプリ側で右からの信頼プロキシ解決を行うため、上書き設定は不要）
- `DATABASE_PATH`: SQLiteファイルパス（デフォルト: data/goblog.db）
- `BLOG_TITLE`: ブログタイトル
- `BASE_URL`: サイトのベースURL（例: https://example.com）
  - sitemap.xml生成に使用される
  - デフォルト: `http://localhost:{PORT}`
- `TZ`: タイムゾーン（例: Asia/Tokyo, UTC, America/New_York）
  - 日付表示に使用される
  - ISO 8601形式（YYYY-MM-DD）+ タイムゾーン略称で表示（例: `2024-12-26 (JST)`）
  - デフォルト: システムのタイムゾーン

- `UPLOAD_DIR`: 画像アップロード先ディレクトリ（デフォルト: data/uploads）
- `MAX_UPLOAD_SIZE`: 最大アップロードサイズ（デフォルト: 5242880 = 5MB）
- `POSTS_PER_PAGE`: トップページ (`/`) とタグ別記事一覧 (`/tags/{tag}`) の 1 ページあたり件数（デフォルト: 20、有効範囲: 1-100、範囲外/パース不能な値はデフォルトに silent fallback）
- `SESSION_TTL`: 管理者セッションの有効期限（`time.ParseDuration` 形式: `24h`、`30m`、`168h` 等）。デフォルト: `24h`、最小: `1m`、不正値/未満は silent fallback。サーバ側セッション TTL とログインクッキーの `MaxAge` の両方をこの値から導出するため、片方だけがズレることはない。**注意**: 変更は次回ログイン以降に発行されるセッションにのみ適用される。既存セッションは発行時の TTL を保持したまま残るため、即座に全員ログアウトさせたい場合はサーバを再起動する（インメモリストアなのでセッションは消える）
- `REMEMBER_TTL`: Remember me クッキーの有効期限（Go duration 形式）。デフォルト: `720h` (30 日)、最小: `1h`、不正値は silent fallback。ログイン時にチェックボックスを ON にすると `remember_tokens` テーブルにこの TTL のレコードが作られ、ブラウザにも同じ MaxAge の `remember_token` クッキーが設定される。session_id が切れていても remember_token が有効なら、公開ページ / `/auth/me` 経由で自動的に新しい session_id が払い出される
- `HEALTHPLANET_ENABLED`: Health Planet 連携の有効化フラグ（`true` のみ有効。デフォルト: 無効）。goblog 本体（管理画面 OAuth フロー）と `hpsync` CLI（日次同期）の両方に必要。本番では `/etc/goblog/healthplanet.env` が単一ソース（goblog.service は `EnvironmentFile=-` で同ファイルを読み、重複キーは unit 内の inline `Environment=` が後勝ちで優先）
- `HEALTHPLANET_CLIENT_ID`: Health Planet OAuth クライアント ID（goblog 本体と hpsync の両方に必要）
- `HEALTHPLANET_CLIENT_SECRET`: Health Planet OAuth クライアントシークレット（goblog 本体と hpsync の両方に必要）

**重要**: 本番環境では`SECURE_COOKIE=true`と`PASSWORD_POLICY=STRONG`と`BASE_URL`と`TRUSTED_PROXIES`を必ず設定すること。

## テスト戦略

**モックパターン:**
```go
type mockPostRepository struct {
    findAllFunc func(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error)
    // 他のメソッド...
}

func (m *mockPostRepository) FindAll(status *domain.PostStatus, limit, offset int) ([]*domain.Post, error) {
    if m.findAllFunc != nil {
        return m.findAllFunc(status, limit, offset)
    }
    return []*domain.Post{}, nil
}
```

**テスト構造:**
- すべての依存関係はコンストラクタでインジェクト
- インターフェースを使用して疎結合を実現
- HTTPハンドラーテスト: `httptest.NewRequest/ResponseRecorder`を使用
- サービステスト: リポジトリをモック
- リポジトリテスト: インメモリSQLiteを使用した統合テスト
- テンプレートパス: `NewRouterWithTemplates()`でテスト用パスを指定可能

## セキュリティ実装

このプロジェクトでは以下のセキュリティ対策が実装済み：

### 1. パスワード管理
- bcryptによるハッシュ化（平文保存なし）
- パスワードポリシー（STRONG: 15文字以上、大小英数記号必須）

### 2. セッション管理
- HttpOnly Cookie（JSからアクセス不可）
- Secure Cookie（HTTPS環境では必須）
- SameSite=Lax（CSRF基本防御）
- 24時間TTL、自動期限切れクリーンアップ

### 3. CSRF対策
- Double Submit Cookie方式
- ログイン時にCSRFトークンをCookieに設定
- クライアントは`X-CSRF-Token`ヘッダーにトークンを含める
- POST/PUT/DELETE/PATCHメソッドで検証（GETは除外）

### 4. ブルートフォース対策
- IPアドレス別のログイン失敗追跡
- 段階的な遅延: 3回失敗→2秒、5回→5秒、10回→30秒
- ログイン成功時に自動リセット
- 30分以上前の記録は自動削除（10分ごとにクリーンアップ）
- プロキシ対応（X-Forwarded-For、X-Real-IPヘッダー）

### 5. 認証ミドルウェア
- すべての管理画面API（記事CRUD）は認証必須
- 未認証は401 Unauthorizedを返却
- セッション検証とユーザー情報取得を自動実行
- ユーザーIDはコンテキストに格納（`GetUserIDFromContext()`で取得）

### 6. Remember me
- selector (16B random) + raw_token (32B random) の 2 段構造
- DB には raw_token を保存せず SHA-256 ハッシュのみ
- 検証は `crypto/subtle.ConstantTimeCompare` で timing 攻撃を防止
- ログアウト時に DB レコードと cookie の両方を失効
- バックグラウンドで 1 時間ごとに期限切れトークンを sweep
- ログイン中の端末管理のため、端末の last-seen `user_agent` / `ip_address` を `remember_tokens` に平文保存する（発行時＋復元時に更新。`TRUSTED_PROXIES` を考慮して IP を解決。運用者自身の端末情報のため読者データのようなセンシティビティは低い）。保持期間はトークン寿命と同じで、期限切れ sweep・ログアウト失効・ユーザー削除（ON DELETE CASCADE）で削除される。別途のデータ保持ポリシーは設けていない
- CDN 導入時の注意: 公開ページの GET で Set-Cookie 副作用が発生する。CDN が Set-Cookie を含むレスポンスをキャッシュ対象外にする設定であれば実害なし。盲目的にキャッシュする CDN を使う場合は `Cache-Control: private, no-store` を当該レスポンスに付ける設計が必要

### 7. 記事リアクション
- **匿名識別**: `reaction_visitor` Cookie（32 バイト乱数、HttpOnly、SameSite=Lax、400 日）。DB には raw 値を保存せず SHA-256 ハッシュ（visitor_key）のみ。
- **CSRF**: 公開状態変更 API（POST/DELETE）は `X-Requested-With` ヘッダ必須。cross-site form / simple request を遮断する。Double-Submit Cookie はこの機能には過剰として不採用。CORS を導入する際は、リアクション API を `Access-Control-Allow-Headers` の許可対象から除外するか、Origin チェックを追加すること（X-Requested-With 防御はブラウザの preflight 強制に依存しているため、CORS を緩めると突破されうる）。
- **レート制限**: IP 単位 30 回/分（インメモリ・単一インスタンス前提）。複数インスタンス化時は Redis 等の共有ストアへの移行が必要。
- **CDN 注意**: リアクション GET エンドポイントは `reaction_visitor` の Set-Cookie 副作用を持つためキャッシュ対象外にすべき。件数の SSR は visitor 非依存だが reacted 状態は JS で付与するためページ自体のキャッシュ性は変えない（Remember me の CDN 注意と同様）。

## ルーティング設計

### 公開ページ（SSR）
- `GET /` - トップページ（記事一覧ページ1。`?page=N` / `?q=` 対応）
- `GET /sitemap.xml` - サイトマップ（XML形式）
- `GET /posts` - 301リダイレクト → `/`（`?page=` / `?q=` を保持）
- `GET /posts/{slug}` - 記事詳細
- `GET /tags` - タグ一覧
- `GET /tags/{tag}` - タグ別記事一覧

### 管理画面（SPA）
- `GET /admin` - SPA入口
- `GET /admin/*` - SPAフォールバック

### API（/api/v1）
**公開エンドポイント:**
- `POST /api/v1/auth/login` - ログイン
- `GET /api/v1/health` - ヘルスチェック
- `GET /api/v1/reactions?slugs=a,b,c` - 複数記事のリアクションをバッチ取得（一覧ページの N+1 回避用、最大100スラグ。件数 + reacted）
- `GET /api/v1/posts/{slug}/reactions` - リアクション一覧取得（件数 + reacted）
- `POST /api/v1/posts/{slug}/reactions/{reactionTypeID}` - リアクション追加（X-Requested-With 必須）
- `DELETE /api/v1/posts/{slug}/reactions/{reactionTypeID}` - リアクション解除（X-Requested-With 必須）

状態変更エンドポイント（POST/DELETE）は認証不要だが X-Requested-With ヘッダ必須 + IP レート制限（30回/分）で保護。

**保護エンドポイント（認証+CSRF必須）:**
- `POST /api/v1/auth/logout` - ログアウト
- `GET /api/v1/auth/me` - ログイン状態確認
- `GET /api/v1/posts` - 記事一覧取得（`?status=draft|published&tag=タグ名&q=検索&limit=N&offset=N`）
- `GET /api/v1/tags` - タグ一覧取得（`?status=draft|published`）
- `POST /api/v1/posts` - 記事作成
- `GET /api/v1/posts/{id}` - 記事取得
- `PUT /api/v1/posts/{id}` - 記事更新
- `DELETE /api/v1/posts/{id}` - 記事削除
- `POST /api/v1/posts/{id}/publish` - 記事公開
- `POST /api/v1/posts/{id}/unpublish` - 記事非公開化
- `POST /api/v1/posts/{id}/pin` - 記事ピン留め
- `POST /api/v1/posts/{id}/unpin` - 記事ピン留め解除
- `POST /api/v1/markdown/preview` - Markdownプレビュー
- `POST /api/v1/images` - 画像アップロード（複数ファイル対応）
- `GET /api/v1/reaction-types` - リアクション種別一覧（無効含む全件、`is_seed` 付き）
- `POST /api/v1/reaction-types` - リアクション種別作成
- `PUT /api/v1/reaction-types/{id}` - リアクション種別更新
- `DELETE /api/v1/reaction-types/{id}` - リアクション種別削除（件数0かつ非seedのみ）
- `GET /api/v1/devices` - ログイン中の端末一覧（remember 端末＋一時セッション、`is_current`/`is_ephemeral` 付き、最終利用の降順）
- `DELETE /api/v1/devices/{kind}/{id}` - 端末の個別失効（kind: `remember`|`session`。現在の端末は 403、他ユーザー/不明な id は 404、不正な kind は 400）
- `POST /api/v1/devices/logout-others` - 現在の端末以外を一括失効
- `GET /api/v1/healthplanet/status` - Health Planet 連携状態確認（常設。`enabled: false` で機能無効を表現）
- `GET /api/v1/healthplanet/auth-url` - 認可 URL 取得（`HEALTHPLANET_ENABLED=true` のときのみ登録）
- `POST /api/v1/healthplanet/exchange` - 認可コードをトークンに交換して保存（`HEALTHPLANET_ENABLED=true` のときのみ登録）

## データフロー例

### 記事作成（認証済みユーザー）
```
POST /api/v1/posts (JSON)
  → AuthMiddleware: セッション検証
  → CSRFMiddleware: トークン検証
  → handlers_api.HandleCreatePost
  → service.CreatePost: スラグ重複チェック、バリデーション
  → repo.Create: DB挿入
  → 作成された記事を返却
```

### 公開記事閲覧（認証不要）
```
GET /posts/{slug}
  → handlers_public.HandlePostDetail
  → service.GetPostBySlug: 公開記事のみ取得
  → goroutine: postViewService.RecordView（ボットフィルタ・IP+UA重複排除30分）
  → HTMLテンプレートレンダリング
```

### ログイン
```
POST /api/v1/auth/login (JSON)
  → handlers_auth.HandleLogin
  → getClientIP: IPアドレス取得（プロキシヘッダー対応）
  → service.Login: パスワード検証、ブルートフォースチェック
  → セッション作成、CSRFトークン生成
  → HttpOnly + Secure Cookie設定
  → ユーザー情報返却（パスワードハッシュは除外）
```

## 重要な実装詳細

### ドメインモデル
- `Post`: 記事（Draft/Published状態）
  - `PublishedAt`: `*time.Time`（下書きの場合はnil）
  - `Tags`: カンマ区切り文字列（非正規化）
  - `IsPinned`: ヘッダーにピン留め表示
  - `ViewCount`: 閲覧回数（`db:"-"`、post_viewsテーブルからサービス層で付与）
- `User`: 認証ユーザー
  - `PasswordHash`: bcryptハッシュ（JSON出力時は`json:"-"`で除外）

### セッション管理
- インメモリストア（`auth.SessionStore`）
- スレッドセーフ（`sync.RWMutex`使用）
- バックグラウンドゴルーチンで期限切れセッションを自動削除

### テンプレート
- ページごとに独立したテンプレートセット
- `NewPublicHandlers()`でテンプレートパスを指定
- テスト時は`NewRouterWithTemplates()`でパスをオーバーライド
- **テンプレート関数**:
  - `truncate`: 文字列をルーン単位で切り詰め（日本語対応）
  - `splitTags`: カンマ区切りタグをスライスに変換
  - `formatDateWithTZ`: ISO 8601形式 + タイムゾーン略称で日付表示（TZ環境変数を使用）

### エラーハンドリング
- APIは常にJSON形式でエラーを返却
- HTTPステータスコードを適切に使用（401/403/404/500など）
- ログは`log.Printf()`で出力（本番環境では構造化ログ推奨）

## 新機能追加時のチェックリスト

1. **ドメインモデル定義** (`internal/domain/`)
2. **リポジトリインターフェース追加** (`internal/repo/`)
3. **サービスロジック実装** (`internal/service/`)
4. **HTTPハンドラー作成** (`internal/http/`)
5. **ルーターに登録** (`router.go`)
6. **ユニットテスト作成** (`*_test.go`)
   - モックを使用したサービステスト
   - `httptest`を使用したハンドラーテスト
7. **マイグレーション追加** （スキーマ変更時、`/migrations/`）

## データベース

- SQLite3を使用
- マイグレーションは起動時に自動実行（`IF NOT EXISTS`で冪等性を確保）
- トランザクションはリポジトリ層で管理
- **スキーマ**:
  - `posts`: 記事データ（id, title, slug, content, status, tags, is_pinned, created_at, updated_at, published_at）
  - `users`: ユーザーデータ（id, username, password_hash, created_at, updated_at）
  - `ogp_cache`: OGPメタ情報キャッシュ（url, title, description, image, local_image, expires_at）
  - `post_views`: 閲覧記録（post_id, viewed_at, ip_address, user_agent）※ON DELETE CASCADE。`ip_address` は信頼プロキシ解決済みの client IP（`clientIP(r, trustedProxies)`、ログイン/リアクションと同じ）で、IP+UA の 30 分重複排除に使う。nginx 背後でも実クライアント単位で dedup される。プライバシー対策として、dedup ウィンドウ経過後（既定 1 時間 = `IPRetentionWindow`、`StartPostViewCleanupLoop` が定期実行）に `ip_address` / `user_agent` を空文字へスクラブする。行は削除しないので累計PV（`COUNT(*)`）は不変、閲覧者の生IP/UAは必要な間だけ保持される。なお起動時スクラブは無くループ初回発火も1間隔後（remember-token sweep と同じ）なので、実効保持 ≈ `IPRetentionWindow` + ループ間隔
  - `remember_tokens`: Remember me トークン（selector / token_hash / expires_at / user_id ON DELETE CASCADE / user_agent / ip_address）。`user_agent`（migration 009）と `ip_address`（migration 010）は端末の **last-seen** 情報で、ログイン中の端末一覧に使用。remember 復元時（`RestoreFromRememberToken`）に現リクエストの UA/IP と差があれば書き戻す（差分が無ければ書き込みなし、UA が空のリクエストでは上書きしない）。これにより、カラム追加前に発行された空文字の既存行（一時的に「不明な端末」表示）も、信頼プロキシ未解決で記録されてしまった IP（nginx 背後の `127.0.0.1` 等）も、次回復元時に自動修復される。IP は公開ページ／管理 API いずれの復元パスでも `TRUSTED_PROXIES` を考慮して解決する
  - `reaction_types`: リアクション絵文字マスタ（id, emoji, label, sort_order, is_active, created_at）。seed は `repo.DefaultReactionTypes` (Go) を単一ソースに `goblog.InitSchema` で `INSERT OR IGNORE` 投入する。migration 008 は CREATE のみ（seed 行は持たない）。seed 絵文字（👍❤️🎉👀🤔）は管理画面から emoji 変更・物理削除不可（無効化のみ）。
  - `post_reactions`: リアクション記録（post_id, reaction_type_id, visitor_key, created_at, UNIQUE(post_id, reaction_type_id, visitor_key)）※ON DELETE CASCADE
  - `health_records`: 健康測定データ（id, measured_at, metric, value, created_at, UNIQUE(measured_at, metric)）。`measured_at` は Health Planet の分単位ローカル時刻。UNIQUE 制約により同一測定点への upsert が冪等。metrics: weight / body_fat / systolic / diastolic / pulse（tags 6021/6022/622E/622F/6230）
  - `healthplanet_tokens`: Health Planet OAuth トークン（id=1 の 1 行テーブル。access_token / refresh_token / expires_at / updated_at）。トークンは API 呼び出しに平文が必要なため平文保存（DB は S3 バックアップに含まれるためアクセス制御に注意）。`updated_at` は毎回リフレッシュ時に更新されるため管理画面の「トークン最終リフレッシュ」タイムスタンプとして表示する（リフレッシュ成功時に更新、後続の同期失敗は hpsync exit コードで検知）

## 依存関係

主要なライブラリ：
- `gorilla/mux`: HTTPルーター
- `jmoiron/sqlx`: データベース抽象化
- `mattn/go-sqlite3`: SQLiteドライバー
- `golang.org/x/crypto/bcrypt`: パスワードハッシュ化
- `joho/godotenv`: .envファイル読み込み
- `html/template`: サーバーサイドレンダリング

## Beyond the Twelve-Factor App

このプロジェクトは[Beyond the Twelve-Factor App](https://www.oreilly.com/library/view/beyond-the-twelve-factor/9781492042631/)の原則に従っています：
- 環境変数による設定管理
- `.env`ファイル（開発環境のみ、本番は環境変数推奨）
- ステートレス（セッションはインメモリだが将来的にRedis移行可能）
- ログは標準出力
