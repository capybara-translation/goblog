# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Claude Code must follow the following rules:

- Always use Context7 MCP when I need library/API documentation, code generation, setup or configuration steps without me having to explicitly ask.

## プロジェクト概要

goblogはGoで書かれたシンプルなブログシステムです。公開ページはSSR（サーバーサイドレンダリング）、管理画面はReact SPAとして構成されています（SPA は計画中、現在未実装）。

### 実装状況

✅ **実装済み:**
- 記事管理（作成、編集、削除、公開/非公開切替）
- タグ機能（タグ一覧、タグ別記事一覧、タグフィルタリング）
- 認証・認可（ユーザー名・パスワード認証、セッション管理）
- セキュリティ（CSRF対策、ブルートフォース対策、パスワードポリシー）
- タイムゾーン対応（ISO 8601形式 + タイムゾーン略称）
- 公開ページ（SSR）
- API エンドポイント（/api/v1）

🚧 **計画中:**
- React SPA 管理画面（/admin）
- RSS フィード
- サイトマップ
- 画像アップロード

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

/internal/
  /http/               # HTTPレイヤー
    router.go          # ルーティング設定
    handlers_public.go # 公開ページハンドラー
    handlers_api.go    # API ハンドラー
    handlers_auth.go   # 認証ハンドラー
    middleware.go      # 認証・CSRFミドルウェア
    static.go          # SPA 配信（計画中）
  /service/            # ビジネスロジック
    post_service.go
    user_service.go
  /repo/               # データアクセス
    post_repo.go
    user_repo.go
  /domain/             # ドメインモデル
    post.go
    user.go
  /auth/               # 認証ユーティリティ（セッション管理）
    session.go
  /config/             # 設定管理
    config.go
  /view/               # ビュー関連
    templates/         # HTMLテンプレート
      layout.html
      home.html
      posts.html
      post.html
      tags.html
      tag_posts.html

/migrations/           # SQLマイグレーションファイル
  001_init.sql
  002_add_tags.sql

/web-admin/            # React SPA 管理画面（計画中、未実装）
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

# ビルド
make build             # bin/goblog, bin/seed を生成
```

## 環境変数

`.env`ファイル（開発時のみ推奨）または環境変数で設定：

- `PORT`: サーバーポート（デフォルト: 8080）
- `SECURE_COOKIE`: Cookie Secure属性（本番: true）
- `PASSWORD_POLICY`: NONE（開発）/STRONG（本番、15文字以上+大小英数記号）
- `DATABASE_PATH`: SQLiteファイルパス（デフォルト: data/goblog.db）
- `BLOG_TITLE`: ブログタイトル
- `BASE_URL`: サイトのベースURL（例: https://example.com）
  - sitemap.xml生成に使用される
  - デフォルト: `http://localhost:{PORT}`
- `TZ`: タイムゾーン（例: Asia/Tokyo, UTC, America/New_York）
  - 日付表示に使用される
  - ISO 8601形式（YYYY-MM-DD）+ タイムゾーン略称で表示（例: `2024-12-26 (JST)`）
  - デフォルト: システムのタイムゾーン

**重要**: 本番環境では`SECURE_COOKIE=true`と`PASSWORD_POLICY=STRONG`と`BASE_URL`を必ず設定すること。

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
- リポジトリテスト: データベースをモック
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

## ルーティング設計

### 公開ページ（SSR）
- `GET /` - トップページ
- `GET /sitemap.xml` - サイトマップ（XML形式）
- `GET /posts` - 記事一覧
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

**保護エンドポイント（認証+CSRF必須）:**
- `POST /api/v1/auth/logout` - ログアウト
- `GET /api/v1/auth/me` - ログイン状態確認
- `GET /api/v1/posts` - 記事一覧取得（`?status=draft|published&tag=タグ名&limit=N&offset=N`）
- `GET /api/v1/tags` - タグ一覧取得（`?status=draft|published`）
- `POST /api/v1/posts` - 記事作成
- `GET /api/v1/posts/{id}` - 記事取得
- `PUT /api/v1/posts/{id}` - 記事更新
- `DELETE /api/v1/posts/{id}` - 記事削除
- `POST /api/v1/posts/{id}/publish` - 記事公開
- `POST /api/v1/posts/{id}/unpublish` - 記事非公開化

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
- マイグレーションは起動時に自動実行
- インデックス: `slug`, `status`, `published_at`, `username`, `tags`
- トランザクションはリポジトリ層で管理
- **スキーマ**:
  - `posts`: 記事データ（id, title, slug, content, status, tags, created_at, updated_at, published_at）
  - `users`: ユーザーデータ（id, username, password_hash, created_at, updated_at）

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
