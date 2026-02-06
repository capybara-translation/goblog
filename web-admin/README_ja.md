# goblog 管理画面 (web-admin)

goblogの管理画面用シングルページアプリケーション（SPA）です。

## 技術スタック

| カテゴリ | 技術 |
|---------|------|
| フレームワーク | React 19 |
| 言語 | TypeScript |
| ビルドツール | Vite |
| スタイリング | Tailwind CSS |
| ルーティング | React Router |
| テスト | Vitest + Testing Library + MSW |
| Markdownエディタ | @uiw/react-md-editor |
| 日付処理 | date-fns |

## 開発コマンド

```bash
# 依存関係のインストール
npm install

# 開発サーバーの起動（http://localhost:5173）
npm run dev

# ビルド（本番用）
npm run build

# ビルドプレビュー
npm run preview

# Lint
npm run lint

# テスト実行
npm test

# テスト（UIモード）
npm run test:ui

# テスト（カバレッジ）
npm run test:coverage
```

**注意:** 開発サーバーを使用する場合、APIリクエストはバックエンド（デフォルト: `http://localhost:8080`）にプロキシされます。`vite.config.ts`の`proxy`設定を参照してください。

## ディレクトリ構成

```
/src
  /api
    client.ts          # APIクライアント（fetch wrapper）
  /components
    Header.tsx         # ヘッダーコンポーネント
    Layout.tsx         # レイアウトコンポーネント
    MarkdownEditor.tsx # Markdownエディタ
    Modal.tsx          # モーダルダイアログ
    PrivateRoute.tsx   # 認証必須ルート
    StatusBadge.tsx    # 公開/下書きバッジ
    TagInput.tsx       # タグ入力コンポーネント
    TagList.tsx        # タグ一覧コンポーネント
  /hooks
    useAuth.tsx        # 認証フック
    useModal.tsx       # モーダル管理フック
  /mocks
    handlers.ts        # MSWモックハンドラー
    server.ts          # MSWサーバー設定
  /pages
    Login.tsx          # ログインページ
    PostEdit.tsx       # 記事編集ページ
    PostList.tsx       # 記事一覧ページ
  /utils
    date.ts            # 日付フォーマット関数
  App.tsx              # ルートコンポーネント（ルーティング定義）
  main.tsx             # エントリポイント
  index.css            # グローバルスタイル
```

## 機能

### 認証
- ログイン/ログアウト
- セッションベース認証（HttpOnly Cookie）
- CSRF対策（X-CSRF-Token ヘッダー）

### 記事管理
- 記事一覧（ステータス・タグでフィルタリング）
- 記事作成・編集（Markdownエディタ）
- 記事の公開/非公開切り替え
- 記事のピン留め/解除
- 記事の削除
- 画像アップロード（ドラッグ&ドロップ対応）

### キーボードショートカット
- `Ctrl/Cmd + S` - 記事を保存

## テスト

### テストツール
- **Vitest** - テストランナー
- **Testing Library** - UIテスト
- **MSW (Mock Service Worker)** - APIモック

### テスト実行

```bash
# 全テスト実行
npm test

# 監視モード
npm test -- --watch

# 特定ファイルのテスト
npm test -- src/components/Header.test.tsx

# UIモード（ブラウザでテスト結果確認）
npm run test:ui

# カバレッジレポート
npm run test:coverage
```

### テストファイル配置
テストファイルはテスト対象と同じディレクトリに配置しています：
- `Header.tsx` → `Header.test.tsx`
- `useAuth.tsx` → `useAuth.test.tsx`

## 環境変数

`.env`ファイルまたは環境変数で設定可能：

| 変数 | 説明 | デフォルト |
|-----|------|-----------|
| `VITE_BLOG_TITLE` | 管理画面に表示するブログタイトル | `goblog` |

## 本番ビルド

```bash
npm run build
```

ビルド成果物は `dist/` ディレクトリに出力されます。Goバイナリに埋め込まれ、`/admin` パスで配信されます。

## 開発時の注意

1. **APIプロキシ**: 開発時は`vite.config.ts`でAPIリクエストをバックエンドにプロキシしています。バックエンド（`make run`）を先に起動してください。

2. **認証**: 開発時は`make seed`でテストユーザーを作成し、`admin` / `password`でログインできます。

3. **ホットリロード**: Viteのホットリロードが有効です。ファイルを保存すると自動的に反映されます。
