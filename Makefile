.PHONY: help run stop test test-v test-cover clean seed reset build install deps install-admin build-admin dev-admin clean-admin

# デフォルトターゲット: ヘルプを表示
help:
	@echo "利用可能なコマンド:"
	@echo "  make run          - サーバーを起動"
	@echo "  make stop         - 起動中のサーバーを停止"
	@echo "  make test         - テストを実行"
	@echo "  make test-v       - テストを詳細出力で実行"
	@echo "  make test-cover   - テストカバレッジを表示"
	@echo "  make clean        - データベースとフロントエンドビルド成果物を削除"
	@echo "  make seed         - テストデータを投入"
	@echo "  make reset        - データベースをリセットしてテストデータ投入"
	@echo "  make build        - フロントエンドとバックエンドをビルド"
	@echo "  make install      - バイナリをインストール"
	@echo "  make deps         - 依存関係をダウンロード"
	@echo "  make install-admin - 管理画面のnpm依存関係をインストール"
	@echo "  make build-admin   - 管理画面をビルド"
	@echo "  make dev-admin     - 管理画面の開発サーバーを起動"
	@echo "  make clean-admin   - 管理画面のビルド成果物を削除"

# サーバーを起動
run:
	@echo "サーバーを起動中..."
	go run cmd/goblog/main.go

# サーバーを停止
stop:
	@echo "goblogプロセスを停止中..."
	@lsof -ti :8080 | xargs kill -9 2>/dev/null || true
	@pkill -f "go run cmd/goblog/main.go" 2>/dev/null || true
	@pkill -f "goblog" 2>/dev/null || true
	@echo "停止しました"

# テストを実行
test:
	@echo "テストを実行中..."
	go test ./...

# テストを詳細出力で実行
test-v:
	@echo "テストを詳細出力で実行中..."
	go test ./... -v

# テストカバレッジを表示
test-cover:
	@echo "テストカバレッジを計算中..."
	go test ./... -cover

# データベースとフロントエンドビルド成果物を削除
clean: clean-admin
	@echo "データベースを削除中..."
	@rm -f data/goblog.db
	@echo "データベースを削除しました"

# テストデータを投入
seed:
	@echo "テストデータを投入中..."
	go run cmd/seed/main.go

# データベースをリセットしてテストデータ投入
reset: clean seed

# フロントエンドとバックエンドをビルド
build: build-admin
	@echo "バックエンドをビルド中..."
	@mkdir -p bin
	go build -o bin/goblog cmd/goblog/main.go
	go build -o bin/seed cmd/seed/main.go
	@echo "ビルド完了: bin/goblog, bin/seed"

# バイナリをインストール
install:
	@echo "バイナリをインストール中..."
	go install ./cmd/goblog
	go install ./cmd/seed
	@echo "インストール完了"

# 依存関係をダウンロード
deps:
	@echo "依存関係をダウンロード中..."
	go mod download
	@echo "ダウンロード完了"

# 管理画面のnpm依存関係をインストール
install-admin:
	@echo "管理画面の依存関係をインストール中..."
	cd web-admin && npm install
	@echo "インストール完了"

# 管理画面をビルド
build-admin:
	@echo "管理画面をビルド中..."
	cd web-admin && npm run build
	@echo "管理画面のビルド完了: web-admin/dist/"

# 管理画面の開発サーバーを起動
dev-admin:
	@echo "管理画面の開発サーバーを起動中..."
	cd web-admin && npm run dev

# 管理画面のビルド成果物を削除
clean-admin:
	@echo "管理画面のビルド成果物を削除中..."
	@rm -rf web-admin/dist
	@echo "削除完了"
