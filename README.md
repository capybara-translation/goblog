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

### 3. データベースのセットアップとテストデータ投入

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

### 4. サーバーの起動

#### 開発環境

```bash
make run
# または
go run cmd/goblog/main.go
```

ブラウザで http://localhost:8080 にアクセスして確認できます。

#### 本番環境（環境変数を使用）

```bash
# 環境変数で設定を指定
SECURE_COOKIE=true PORT=3000 BLOG_TITLE="My Awesome Blog" go run cmd/goblog/main.go

# または環境変数をエクスポート
export SECURE_COOKIE=true
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
| `DATABASE_PATH` | データベースファイルのパス | `data/goblog.db` |
| `BLOG_TITLE` | ブログのタイトル（ヘッダーやページタイトルに表示） | `goblog` |

**注意:** 本番環境（HTTPS対応サーバー）では必ず `SECURE_COOKIE=true` を設定してください。これにより Cookie が HTTPS 接続でのみ送信されるようになります。

### 5. テストの実行

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

### セキュリティ関連

1. パスワードは必ずハッシュ化（平文保存はNG）
    - DBには ハッシュのみ保存（例：bcrypt / argon2id）
    - ログイン時に照合してOKならセッション発行
2. セッションは HttpOnly + Secure cookie
    - HttpOnly（JSから読めない）
    - Secure（HTTPSのみ） - **本番環境では環境変数 `SECURE_COOKIE=true` で有効化**
    - SameSite=Lax（まずはこれでOK）
    - セッションIDは ランダムで十分長いもの
3. CSRF対策
    - 管理画面がCookie認証でAPI叩くなら、CSRFは基本やる。やり方は2択：
        - Double Submit Cookie（SPAと相性良い）
        - Origin/Refererチェック + CSRFトークン（実装は増えるが堅い）
4. ブルートフォース対策（最低限）
    - ログイン失敗回数で 遅延 or 一時ロック
    - 管理画面を Basic Authで二重ロックするのも安くて強い（Nginxで可能）

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
