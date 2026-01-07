package main

import (
	"fmt"
	"log"
	"time"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/db"
	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
)

// seedPost はシードデータの記事を表す構造体です
type seedPost struct {
	title   string
	slug    string
	content string
	tags    string
	publish bool
}

// getSeedPosts はシードデータの記事一覧を返します
// createdAtはmain関数で設定されます
func getSeedPosts() []seedPost {
	return []seedPost{
		{
			title:   "Goでブログシステムを作ってみた",
			slug:    "building-blog-with-go",
			content: "Goを使ってシンプルなブログシステムを構築しました。このプロジェクトでは、クリーンアーキテクチャの原則に基づいて、Repository、Service、Handlerの3層構造を採用しています。これにより、ビジネスロジックとインフラストラクチャ層を明確に分離し、テスタブルで保守性の高いコードベースを実現しました。\n\nこのブログシステムは以下の機能を持っています：記事の作成・編集・削除、公開・非公開の切り替え、タグによる分類、ページネーション対応、そして充実したセキュリティ機能です。特にセキュリティ面では、bcryptによるパスワードハッシュ化、HttpOnlyとSecure属性を持つセッションクッキー、CSRF保護（Double Submit Cookie方式）、IPアドレス別のブルートフォース対策など、本番環境でも使用できるレベルの実装を行いました。\n\nバックエンドはGo言語で開発し、フロントエンドにはGo標準のhtml/templateパッケージを使用しています。スタイリングにはTailwind CSSを採用し、シンプルで洗練されたデザインを実現しました。データベースにはSQLiteを採用し、軽量で扱いやすい構成にしました。本番環境ではPostgreSQLやMySQLへの移行も容易な設計になっています。\n\nテストについても力を入れており、モックを活用したユニットテストやテーブル駆動テストを実装し、高いテストカバレッジを維持しています。",
			tags:    "Go,ブログ,開発,クリーンアーキテクチャ",
			publish: true,
		},
		{
			title:   "Goのgoroutineとchannelについて",
			slug:    "go-goroutine-channel",
			content: "Goの並行処理について深く学びました。goroutineは軽量なスレッドのようなもので、OSスレッドと比較して非常に少ないメモリで動作します。通常のOSスレッドが数メガバイトのスタックを必要とするのに対し、goroutineは初期スタックサイズがわずか2KBから始まり、必要に応じて動的に拡張されます。このため、数千、数万のgoroutineを同時に実行することが可能です。\n\nchannelを使うことで、goroutine間で安全にデータをやり取りすることができます。これは「メモリを共有することで通信するのではなく、通信することでメモリを共有する」というGoの哲学を体現しています。channelには、バッファなしのチャネルとバッファ付きのチャネルがあり、用途に応じて使い分けることができます。バッファなしのチャネルは同期的に動作し、送信側と受信側が準備できるまで待機します。一方、バッファ付きのチャネルは非同期的に動作し、バッファがいっぱいになるまで送信側はブロックされません。\n\nまた、selectステートメントを使用することで、複数のチャネル操作を待機し、準備ができた最初の操作を実行することができます。これにより、タイムアウト処理やキャンセル処理などの複雑な制御フローを簡潔に記述できます。実際のコード例を交えて詳しく説明していきます。",
			tags:    "Go,並行処理,goroutine,channel",
			publish: true,
		},
		{
			title:   "SQLiteの使い方入門",
			slug:    "sqlite-basics",
			content: "SQLiteは軽量なリレーショナルデータベースで、ファイルベースで動作します。\n\nインストール不要で使えるため、小規模なアプリケーションや開発環境でのテストに最適です。このブログシステムでもSQLiteを採用しています。\n\nCRUD操作の基本から、インデックスの使い方まで解説します。",
			tags:    "SQLite,データベース,入門",
			publish: true,
		},
		{
			title:   "Goのテストの書き方",
			slug:    "go-testing-guide",
			content: "Goには標準でテストフレームワークが組み込まれています。testing パッケージを使うことで、簡単にユニットテストを書くことができます。\n\nテーブル駆動テストという手法を使うと、複数のテストケースを効率的に記述できます。また、モックを使った依存関係の切り離しも重要です。\n\n実際のテストコード例を見ていきましょう。",
			tags:    "Go,テスト,testing",
			publish: true,
		},
		{
			title:   "HTMLテンプレートエンジンの活用",
			slug:    "html-template-usage",
			content: "Goの標準パッケージ html/template を使ってWebページをレンダリングしています。\n\nテンプレート継承を使うことで、共通のレイアウトを定義し、各ページで個別のコンテンツを埋め込むことができます。XSS対策も自動的に行われるため、セキュアなWebアプリケーションを構築できます。",
			tags:    "Go,テンプレート,HTML",
			publish: true,
		},
		{
			title:   "RESTful APIの設計パターン",
			slug:    "restful-api-design",
			content: "RESTful APIを設計する際のベストプラクティスについてまとめました。\n\nHTTPメソッド（GET, POST, PUT, DELETE）を適切に使い分けることで、直感的で使いやすいAPIを構築できます。また、ステータスコードも適切に返すことが重要です。\n\nこのブログシステムでも管理画面用のAPIをRESTfulに設計しています。",
			tags:    "API,REST,設計",
			publish: true,
		},
		{
			title:   "Goでのエラーハンドリング",
			slug:    "go-error-handling",
			content: "Goのエラーハンドリングは他の言語と比べてユニークです。例外機構ではなく、error型を戻り値として返す方式を採用しています。\n\n明示的なエラー処理により、エラーが発生する可能性のある箇所が明確になります。errors.Is や errors.As を使った詳細なエラー判定も可能です。",
			tags:    "Go,エラー処理",
			publish: true,
		},
		{
			title:   "データベースマイグレーションの管理",
			slug:    "database-migration",
			content: "データベーススキーマの変更を管理するマイグレーション機能について解説します。\n\nバージョン管理されたSQLファイルを使うことで、開発環境から本番環境まで一貫したスキーマ管理ができます。このプロジェクトでもマイグレーション機能を実装しています。",
			tags:    "データベース,マイグレーション",
			publish: true,
		},
		{
			title:   "HTTPルーティングの実装",
			slug:    "http-routing",
			content: "gorilla/muxを使ったHTTPルーティングの実装方法を紹介します。\n\nパスパラメータやクエリパラメータの取得、ミドルウェアの適用など、実用的な機能が揃っています。標準のhttp.ServeMuxよりも柔軟なルーティングが可能です。",
			tags:    "Go,HTTP,ルーティング",
			publish: true,
		},
		{
			title:   "ページネーションの実装パターン",
			slug:    "pagination-implementation",
			content: "大量のデータを扱う際に必須となるページネーション機能の実装方法を解説します。\n\nオフセットベースとカーソルベースの2つの主要な方式があります。このブログではオフセットベースを採用し、limit+1のテクニックで次ページの有無を判定しています。\n\nパフォーマンスとUXのバランスを考えた実装が重要です。",
			tags:    "ページネーション,データベース,パフォーマンス",
			publish: true,
		},
		{
			title:   "Goのインターフェースを理解する",
			slug:    "go-interfaces",
			content: "Goのインターフェースは暗黙的に実装されるため、他の言語とは異なる特徴があります。\n\nダックタイピングのような柔軟性と、型安全性を両立しています。小さなインターフェースを組み合わせることで、テスタブルで保守性の高いコードが書けます。",
			tags:    "Go,インターフェース,設計",
			publish: true,
		},
		{
			title:   "依存性注入（DI）のパターン",
			slug:    "dependency-injection",
			content: "依存性注入（Dependency Injection）を使うことで、コンポーネント間の結合度を下げることができます。\n\nGoではコンストラクタ関数を通じて依存関係を注入するのが一般的です。このブログシステムでも、Repository → Service → Handler という階層で依存性注入を行っています。",
			tags:    "Go,DI,設計パターン",
			publish: true,
		},
		{
			title:   "JSONのエンコード・デコード",
			slug:    "json-encoding-decoding",
			content: "GoでJSONを扱う方法を解説します。encoding/json パッケージを使うことで、簡単にJSONとGoの構造体を相互変換できます。\n\nタグを使ってフィールド名のマッピングを制御したり、カスタムマーシャラーを実装することも可能です。API開発には欠かせない知識です。",
			tags:    "Go,JSON,API",
			publish: true,
		},
		{
			title:   "コンテキスト（context）の使い方",
			slug:    "context-usage",
			content: "context.Context はGoの並行処理において重要な役割を果たします。\n\nタイムアウトやキャンセル処理、リクエストスコープの値の受け渡しなどに使用します。HTTPハンドラーでは、リクエストごとにコンテキストが自動的に作成されます。",
			tags:    "Go,context,並行処理",
			publish: true,
		},
		{
			title:   "ログ出力のベストプラクティス",
			slug:    "logging-best-practices",
			content: "適切なログ出力は、運用時のトラブルシューティングに不可欠です。\n\nログレベル（DEBUG, INFO, WARN, ERROR）を適切に使い分け、構造化ログを採用することで、ログの検索や分析が容易になります。標準のlogパッケージや、サードパーティのライブラリを活用しましょう。",
			tags:    "ログ,運用,デバッグ",
			publish: true,
		},
		{
			title:   "Goのモジュールシステム",
			slug:    "go-modules",
			content: "Go Modulesは依存関係を管理するための公式ツールです。\n\ngo.modファイルで依存パッケージのバージョンを明示的に管理できます。go getコマンドで新しいパッケージを追加し、go mod tidyで不要な依存関係を削除できます。",
			tags:    "Go,モジュール,パッケージ管理",
			publish: true,
		},
		{
			title:   "セキュリティ対策の基本",
			slug:    "security-basics",
			content: "Webアプリケーションのセキュリティ対策について解説します。\n\nXSS、CSRF、SQLインジェクションなど、主要な脆弱性への対策方法を紹介します。Goの標準パッケージには、これらの対策が組み込まれているものも多いですが、正しく使うことが重要です。",
			tags:    "セキュリティ,Web,XSS",
			publish: true,
		},
		{
			title:   "パフォーマンスチューニングの手法",
			slug:    "performance-tuning",
			content: "Goアプリケーションのパフォーマンスを改善する手法を紹介します。\n\nプロファイリングツール（pprof）を使ってボトルネックを特定し、適切な最適化を行います。メモリ割り当ての削減、goroutineの適切な使用、データ構造の選択などが重要なポイントです。",
			tags:    "パフォーマンス,最適化,pprof",
			publish: true,
		},
		{
			title:   "テスト駆動開発（TDD）の実践",
			slug:    "tdd-practice",
			content: "テスト駆動開発（TDD）をGoで実践する方法を解説します。\n\nRed-Green-Refactorのサイクルを回すことで、テストカバレッジの高い、保守性の良いコードを書くことができます。このブログシステムの開発でもTDDを意識して実装しました。",
			tags:    "TDD,テスト,開発手法",
			publish: true,
		},
		{
			title:   "CI/CDパイプラインの構築",
			slug:    "cicd-pipeline",
			content: "GitHub Actionsを使ったCI/CDパイプラインの構築について解説します。\n\n自動テスト、ビルド、デプロイまでを自動化することで、開発効率を大幅に向上させることができます。ワークフローファイルの書き方から、実際のデプロイまでを詳しく説明します。",
			tags:    "CI/CD,GitHub Actions,自動化",
			publish: true,
		},
		{
			title:   "Dockerコンテナ化の手順",
			slug:    "docker-containerization",
			content: "GoアプリケーションをDockerコンテナ化する手順をまとめます。\n\nマルチステージビルドを使うことで、軽量なコンテナイメージを作成できます。本番環境での運用も考慮した実践的な内容です。",
			tags:    "Docker,コンテナ,デプロイ",
			publish: true,
		},
		{
			title:   "GraphQL APIの実装",
			slug:    "graphql-api",
			content: "RESTの代替として注目されているGraphQL APIの実装方法について解説します。\n\nスキーマ定義からリゾルバーの実装、クエリの最適化まで、実用的なGraphQL APIの構築方法を紹介します。",
			tags:    "GraphQL,API",
			publish: true,
		},
		{
			title:   "認証・認可の実装",
			slug:    "authentication-authorization",
			content: "JWT認証やセッション管理など、認証・認可の実装方法について解説します。\n\nセキュアな認証システムの構築方法、アクセストークンとリフレッシュトークンの使い分け、ロールベースアクセス制御（RBAC）の実装など、実践的な内容をカバーします。",
			tags:    "認証,JWT,セキュリティ",
			publish: true,
		},
		{
			title:   "WebSocketによるリアルタイム通信",
			slug:    "realtime-communication",
			content: "WebSocketを使ったリアルタイム通信の実装について解説します。\n\nチャットアプリケーションやリアルタイム通知など、双方向通信が必要な機能の実装方法を詳しく説明します。接続管理やエラーハンドリングのベストプラクティスも紹介します。",
			tags:    "WebSocket,リアルタイム",
			publish: true,
		},
		{
			title:   "Goのメモリ管理とガベージコレクション",
			slug:    "go-memory-management",
			content: "Goのメモリ管理の仕組みとガベージコレクションについて解説します。\n\nメモリリークの検出方法、プロファイリングツールの使い方、パフォーマンスチューニングのポイントなど、実践的な内容をカバーします。",
			tags:    "Go,メモリ管理,GC",
			publish: true,
		},
		{
			title:   "データベース接続プーリングの最適化",
			slug:    "database-connection-pooling",
			content: "データベース接続プーリングの仕組みと最適化について解説します。\n\nMaxOpenConns、MaxIdleConns、ConnMaxLifetimeなどのパラメータの適切な設定方法を、実際のパフォーマンス測定結果とともに紹介します。",
			tags:    "データベース,パフォーマンス,接続プール",
			publish: true,
		},
		{
			title:   "Goにおける並行処理のパターン",
			slug:    "go-concurrency-patterns",
			content: "Goの並行処理で使われる一般的なパターンについて解説します。\n\nワーカープール、パイプライン、ファンアウト・ファンイン、コンテキストによるキャンセル処理など、実用的なパターンをコード例とともに紹介します。",
			tags:    "Go,並行処理,デザインパターン",
			publish: true,
		},
		{
			title:   "RESTful APIのバージョニング戦略",
			slug:    "api-versioning-strategy",
			content: "APIのバージョニング戦略について解説します。\n\nURL、ヘッダー、クエリパラメータを使った方法の比較、後方互換性の維持、非推奨化の進め方など、長期的な運用を見据えた設計を紹介します。",
			tags:    "API,バージョニング,設計",
			publish: true,
		},
		{
			title:   "Goのジェネリクスを活用する",
			slug:    "go-generics-usage",
			content: "Go 1.18で導入されたジェネリクスの使い方を解説します。\n\n型パラメータの基本から、制約の定義、実用的なユースケースまで、ジェネリクスを効果的に活用する方法を紹介します。",
			tags:    "Go,ジェネリクス,型システム",
			publish: true,
		},
		{
			title:   "マイクロサービスアーキテクチャ入門",
			slug:    "microservices-architecture",
			content: "マイクロサービスアーキテクチャの基本と実装について解説します。\n\nサービス分割の考え方、サービス間通信、データ管理、トランザクション処理など、マイクロサービス化の実践的な内容をカバーします。",
			tags:    "マイクロサービス,アーキテクチャ,分散システム",
			publish: true,
		},
		{
			title:   "gRPCによるサービス間通信",
			slug:    "grpc-service-communication",
			content: "gRPCを使ったサービス間通信の実装について解説します。\n\nProtocol Buffersの定義、サーバー・クライアントの実装、ストリーミング、エラーハンドリング、インターセプターの使い方など、実践的な内容を紹介します。",
			tags:    "gRPC,マイクロサービス,RPC",
			publish: true,
		},
		{
			title:   "Goでのファイルアップロード処理",
			slug:    "file-upload-handling",
			content: "Goでファイルアップロードを処理する方法を解説します。\n\nマルチパートフォームデータの扱い、ファイルサイズ制限、画像のリサイズ、S3へのアップロード、セキュリティ対策など、実用的な実装方法を紹介します。",
			tags:    "Go,ファイルアップロード,画像処理",
			publish: true,
		},
		{
			title:   "レート制限の実装パターン",
			slug:    "rate-limiting-patterns",
			content: "APIのレート制限（Rate Limiting）の実装方法を解説します。\n\nトークンバケット、リーキーバケット、スライディングウィンドウなど、様々なアルゴリズムの特徴と実装例を紹介します。",
			tags:    "レート制限,API,セキュリティ",
			publish: true,
		},
		{
			title:   "Goのリフレクションを理解する",
			slug:    "go-reflection",
			content: "Goのリフレクション機能について解説します。\n\nreflectパッケージの基本、型情報の取得、値の操作、構造体タグの読み取りなど、リフレクションの実用的な使い方を紹介します。",
			tags:    "Go,リフレクション,メタプログラミング",
			publish: true,
		},
		{
			title:   "キャッシング戦略とRedisの活用",
			slug:    "caching-strategy-redis",
			content: "効果的なキャッシング戦略とRedisの活用方法を解説します。\n\nキャッシュの基本、有効期限の設定、キャッシュ無効化戦略、Redisのデータ構造の使い分けなど、パフォーマンス向上のためのテクニックを紹介します。",
			tags:    "キャッシュ,Redis,パフォーマンス",
			publish: true,
		},
		{
			title:   "Goにおけるクリーンアーキテクチャ",
			slug:    "clean-architecture-go",
			content: "Goでクリーンアーキテクチャを実装する方法を解説します。\n\nレイヤー分割、依存性の方向、インターフェースの活用、テスタビリティの向上など、保守性の高いアプリケーション設計を紹介します。",
			tags:    "クリーンアーキテクチャ,設計,Go",
			publish: true,
		},
		{
			title:   "OpenTelemetryによる分散トレーシング",
			slug:    "opentelemetry-distributed-tracing",
			content: "OpenTelemetryを使った分散トレーシングの実装について解説します。\n\nトレースの基本、スパンの作成、コンテキスト伝播、Jaegerとの連携など、マイクロサービス環境での可観測性を向上させる方法を紹介します。",
			tags:    "OpenTelemetry,分散トレーシング,可観測性",
			publish: true,
		},
		{
			title:   "Goのテストダブルとモックライブラリ",
			slug:    "test-doubles-mocking",
			content: "テストダブルの種類とモックライブラリの使い方を解説します。\n\nスタブ、モック、スパイ、フェイクの違い、gomockやtestifyの使い方、効果的なテストの書き方を紹介します。",
			tags:    "テスト,モック,Go",
			publish: true,
		},
		{
			title:   "環境変数と設定管理のベストプラクティス",
			slug:    "environment-config-management",
			content: "アプリケーションの設定管理のベストプラクティスを解説します。\n\n環境変数の使い方、.envファイル、設定ファイルのバリデーション、機密情報の管理など、安全で柔軟な設定管理の方法を紹介します。",
			tags:    "設定管理,環境変数,セキュリティ",
			publish: true,
		},
		{
			title:   "Goでのメール送信実装（下書き）",
			slug:    "email-sending-implementation",
			content: "Goでメール送信機能を実装する方法について書く予定です。\n\nSMTP、テンプレートエンジン、添付ファイルなどをカバーします。",
			tags:    "Go,メール,SMTP",
			publish: false,
		},
		{
			title:   "Kubernetes入門（下書き）",
			slug:    "kubernetes-introduction",
			content: "Kubernetesの基本と、Goアプリケーションのデプロイ方法について書く予定です。",
			tags:    "Kubernetes,コンテナ,デプロイ",
			publish: false,
		},
		{
			title:   "技術メモ：開発環境のセットアップ",
			slug:    "dev-environment-setup-memo",
			content: "開発環境のセットアップ手順についてのメモです。\n\n後で詳しく書く予定ですが、とりあえず備忘録として残しておきます。",
			tags:    "",
			publish: true,
		},
		{
			title:   "雑記：プログラミング学習の記録",
			slug:    "programming-learning-notes",
			content: "日々の学習記録です。\n\n特定のトピックではないため、タグは未設定です。",
			tags:    "",
			publish: true,
		},
		{
			title:   "未分類のアイデア（下書き）",
			slug:    "uncategorized-ideas",
			content: "まだカテゴリが決まっていないアイデアのメモです。\n\n後で整理する予定です。",
			tags:    "",
			publish: false,
		},
	}
}

func main() {
	fmt.Println("=== シードデータ投入開始 ===")

	// 設定の読み込み
	cfg := config.Load()

	// データベースの初期化
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// マイグレーションの実行
	if err := db.RunMigrations(database, "migrations/001_create_posts.sql", "migrations/002_create_users.sql"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Println("Database initialized successfully")

	// Repository層の初期化
	postRepo := repo.NewPostRepository(database)
	userRepo := repo.NewUserRepository(database)

	// SessionStoreとAuthServiceの初期化
	sessionStore := auth.NewInMemorySessionStore()
	authService := service.NewAuthService(userRepo, sessionStore, config.PasswordPolicyNone)

	// 既存のデータを削除（開発環境用）
	fmt.Println("Clearing existing data...")
	_, err = database.Exec("DELETE FROM posts")
	if err != nil {
		log.Fatalf("Failed to clear existing posts data: %v", err)
	}
	_, err = database.Exec("DELETE FROM users")
	if err != nil {
		log.Fatalf("Failed to clear existing users data: %v", err)
	}

	// シードデータの投入
	seedPosts := getSeedPosts()
	publishedCount := 0
	draftCount := 0

	fmt.Printf("Inserting %d posts...\n", len(seedPosts))

	// 基準日時（最新記事の日時）
	baseTime := time.Date(2024, 12, 28, 12, 0, 0, 0, time.UTC)

	for i, seed := range seedPosts {
		// 配列の最初（index=0）が最新記事となるよう、日付を遡って設定
		createdAt := baseTime.AddDate(0, 0, -i)
		updatedAt := createdAt

		// 公開日時の設定
		var publishedAt *time.Time
		if seed.publish {
			publishedAt = &createdAt
		}

		// Postオブジェクトを作成
		post := &domain.Post{
			Title:       seed.title,
			Slug:        seed.slug,
			Content:     seed.content,
			Status:      domain.PostStatusDraft,
			Tags:        seed.tags,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
			PublishedAt: publishedAt,
		}

		// 公開フラグによってステータスを設定
		if seed.publish {
			post.Status = domain.PostStatusPublished
			publishedCount++
		} else {
			draftCount++
		}

		// Repositoryで直接作成
		if err := postRepo.Create(post); err != nil {
			log.Printf("Warning: Failed to create post %d (%s): %v", i+1, seed.title, err)
			continue
		}
	}

	fmt.Printf("\n公開記事: %d件\n", publishedCount)
	fmt.Printf("下書き: %d件\n", draftCount)
	fmt.Printf("合計: %d件\n", publishedCount+draftCount)

	// テストユーザーの作成
	fmt.Println("\nCreating test user...")
	testUser, err := authService.CreateUser("admin", "password")
	if err != nil {
		log.Printf("Warning: Failed to create test user: %v", err)
	} else {
		fmt.Printf("Test user created: username=%s, id=%d\n", testUser.Username, testUser.ID)
	}

	fmt.Printf("\n=== シードデータ投入完了 ===\n")
	fmt.Printf("\nブログを起動して http://localhost:8080 で確認できます\n")
	fmt.Printf("管理画面ログイン情報: username=admin, password=password\n")
}
