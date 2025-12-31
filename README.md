# goblog
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

これにより以下のデータが投入されます：
- 公開記事: 19件
- 下書き記事: 5件

### 5. サーバーの起動

#### 開発環境

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

#### 本番環境（環境変数を使用）

```bash
# 環境変数で設定を指定
SECURE_COOKIE=true PASSWORD_POLICY=STRONG PORT=3000 BLOG_TITLE="My Awesome Blog" go run cmd/goblog/main.go

# または環境変数をエクスポート
export SECURE_COOKIE=true
export PASSWORD_POLICY=STRONG
export PORT=3000
export DATABASE_PATH=/var/lib/goblog/production.db
export BLOG_TITLE="My Awesome Blog"
go run cmd/goblog/main.go
```

**利用可能な環境変数:**

| 環境変数 | 説明 | デフォルト値 |
|---------|------|-------------|
| `PORT` | サーバーのポート番号 | `8080` |
| `SECURE_COOKIE` | Cookie の Secure フラグ（HTTPS環境では `true` に設定） | `false` |
| `PASSWORD_POLICY` | パスワードポリシー（`NONE` または `STRONG`） | `NONE` |
| `DATABASE_PATH` | データベースファイルのパス | `data/goblog.db` |
| `BLOG_TITLE` | ブログのタイトル（ヘッダーやページタイトルに表示） | `goblog` |

**パスワードポリシーについて:**

- `NONE`: 制限なし（開発/テスト環境向け）
- `STRONG`: 厳格なポリシー（本番環境向け）
  - 最小15文字
  - 大文字を1文字以上含む
  - 小文字を1文字以上含む
  - 数字を1文字以上含む
  - 記号を1文字以上含む

※ 大文字小文字を区別しません（`none`/`NONE`/`None`、`strong`/`STRONG`/`Strong` すべて有効）

**注意:**
- 本番環境（HTTPS対応サーバー）では必ず `SECURE_COOKIE=true` を設定してください。これにより Cookie が HTTPS 接続でのみ送信されるようになります。
- 本番環境では `PASSWORD_POLICY=STRONG` を設定することを強く推奨します。
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

## 利用可能なMakeコマンド

開発でよく使うコマンドをMakefileにまとめています：

```bash
make help        # ヘルプを表示
make run         # サーバーを起動
make stop        # 起動中のサーバーを停止
make test        # テストを実行
make test-v      # テストを詳細出力で実行
make test-cover  # テストカバレッジを表示
make clean       # データベースを削除
make seed        # テストデータを投入
make reset       # データベースをリセットしてテストデータ投入
make build       # バイナリをビルド
make install     # バイナリをインストール
make deps        # 依存関係をダウンロード
```

## ディレクトリ構成

```
/cmd/
  /goblog/main.go  (アプリケーション本体)
  /seed/main.go    (テストデータ投入コマンド)
/internal/
  /http/
    router.go
    middleware.go
    handlers_public.go
    handlers_api.go
  /domain/
    post.go
  /repo/
    post_repo.go
  /service/
    post_service.go
  /auth/
    session.go
  /view/
    templates/   (*.html)
    assets/      (admin distを置く or 埋め込む)
/migrations/
  001_init.sql
/web-admin/       (React)
  src/...
```

## 基本的な開発方針

- これは自分のGo言語勉強用プロジェクトなので、あまり凝った構成にはしない。
- ただしある程度のベストプラクティスには従って開発したい。例えば：
    - handlerは極力薄く、DB直叩きしない。
    - serviceに業務ロジックを寄せる。
    - インターフェースをうまく活用して疎結合にしテストしやすくする。
- Beyond the Twelve-Factor App (https://raw.githubusercontent.com/ffisk/books/master/beyond-the-twelve-factor-app.pdf) に従った開発 
- 当然最終的には公開するつもりなのでセキュリティにも気をつけたい。
- 公開ページはSSR（通常のページ遷移型のWebアプリケーション）として作成し、管理画面はVite + React + React RouterのSPAとする。
    - `/inernal/`: 公開ページのソースコード 
    - `/web-admin/`: 管理画面SPAのソースコード
- 公開ページはHTML（html/template）
- APIは必ずJSON（エラーもJSON）
- DBはSQLiteを使う。
- 認証はusername + passwordのシンプルな自前実装。CognitoやAuth0などの外部サービスは使わない。Cookieセッションで管理画面だけ守る。
- CSSフレームワークはTailwindを使う（これも勉強目的）。
- ホスト先はAWSのLightsailの予定。

### セキュリティ機能

このプロジェクトでは以下のセキュリティ対策を実装しています：

#### 1. パスワード管理 ✅ 実装済み

- **ハッシュ化**: bcryptによる安全なパスワードハッシュ化（平文保存なし）
- **パスワードポリシー**: 環境変数 `PASSWORD_POLICY` で制御
  - `NONE`: 制限なし（開発/テスト環境向け）
  - `STRONG`: 厳格なポリシー（本番環境推奨）
    - 最小15文字
    - 大文字・小文字・数字・記号をそれぞれ1文字以上含む

#### 2. セッション管理 ✅ 実装済み

- **HttpOnly Cookie**: JavaScriptからアクセス不可
- **Secure Cookie**: 本番環境では `SECURE_COOKIE=true` で有効化（HTTPS接続でのみ送信）
- **SameSite=Lax**: CSRF攻撃の基本的な防御
- **セッション有効期限**: 24時間
- **ランダムセッションID**: 暗号学的に安全な乱数生成

#### 3. CSRF対策 ✅ 実装済み

- **Double Submit Cookie方式**: SPAと相性の良い実装
- **対象メソッド**: POST、PUT、DELETE、PATCH（GETは除外）
- **仕組み**:
  - ログイン時にCSRFトークンをCookieに設定
  - クライアントはリクエストヘッダー `X-CSRF-Token` にトークンを含める
  - サーバーはCookieとヘッダーのトークンを照合

#### 4. ブルートフォース対策 ✅ 実装済み

- **IPアドレス別の失敗追跡**: 各IPアドレスごとに独立してログイン失敗回数を記録
- **段階的な遅延**:
  - 3回失敗後: 2秒の遅延
  - 5回失敗後: 5秒の遅延
  - 10回以上失敗: 30秒の遅延
- **自動リセット**: ログイン成功時にカウンターをリセット
- **自動クリーンアップ**: 30分以上前の失敗記録を定期的に削除（10分ごと）
- **プロキシ対応**: `X-Forwarded-For` および `X-Real-IP` ヘッダーに対応

#### 5. 認証ミドルウェア ✅ 実装済み

- すべての管理画面API（記事作成・編集・削除など）は認証を必須化
- 未認証の場合は 401 Unauthorized を返却
- セッションの検証とユーザー情報の取得を自動実行

**セキュリティのベストプラクティス:**
- 本番環境では必ず `SECURE_COOKIE=true` と `PASSWORD_POLICY=STRONG` を設定してください
- HTTPS環境で運用することを強く推奨します
- 管理画面へのアクセスをBasic認証で二重保護することも検討してください（Nginxなどで設定可能）

### URL設計

#### 公開ページ（SSR）

- GET /（トップ）
- GET /posts（一覧）
- GET /posts/{slug}（詳細）
- GET /tags/{tag}（タグ別）
- GET /rss.xml
- GET /sitemap.xml

#### 管理画面（SPA）

- GET /admin（SPAの入口）
- GET /admin/*（全部SPAにフォールバック）

#### API（管理画面が叩く）

- /api/v1/... に集約（将来破壊的変更しても共存できるのでおすすめ）
- POST /api/v1/auth/login
- POST /api/v1/auth/logout
- GET  /api/v1/auth/me（ログイン状態確認）
- GET  /api/v1/posts?status=draft|published&q=&page=&limit=
- POST /api/v1/posts
- GET  /api/v1/posts/{id}
- PUT  /api/v1/posts/{id}
- DELETE /api/v1/posts/{id}
- POST /api/v1/posts/{id}/publish（公開操作を分けたい場合）
- POST /api/v1/uploads（画像アップロード → URL返す）
- （任意）GET /api/v1/tags
