.PHONY: help run stop test test-v test-cover clean seed reset build install deps

# デフォルトターゲット: ヘルプを表示
help:
	@echo "利用可能なコマンド:"
	@echo "  make run         - サーバーを起動"
	@echo "  make stop        - 起動中のサーバーを停止"
	@echo "  make test        - テストを実行"
	@echo "  make test-v      - テストを詳細出力で実行"
	@echo "  make test-cover  - テストカバレッジを表示"
	@echo "  make clean       - データベースを削除"
	@echo "  make seed        - テストデータを投入"
	@echo "  make reset       - データベースをリセットしてテストデータ投入"
	@echo "  make build       - バイナリをビルド"
	@echo "  make install     - バイナリをインストール"
	@echo "  make deps        - 依存関係をダウンロード"

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

# データベースを削除
clean:
	@echo "データベースを削除中..."
	@rm -f data/goblog.db
	@echo "データベースを削除しました"

# テストデータを投入
seed:
	@echo "テストデータを投入中..."
	go run cmd/seed/main.go

# データベースをリセットしてテストデータ投入
reset: clean seed

# バイナリをビルド
build:
	@echo "バイナリをビルド中..."
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
