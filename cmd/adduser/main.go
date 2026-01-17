package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/capybara-translation/goblog/internal/auth"
	"github.com/capybara-translation/goblog/internal/config"
	"github.com/capybara-translation/goblog/internal/db"
	"github.com/capybara-translation/goblog/internal/repo"
	"github.com/capybara-translation/goblog/internal/service"
	"github.com/joho/godotenv"
	"golang.org/x/term"
)

func main() {
	// .envファイルから環境変数を読み込む（存在しない場合はスキップ）
	if err := godotenv.Load(); err != nil {
		// .envファイルが存在しない場合はログに記録するが、エラーにはしない
		log.Printf(".env file not found or could not be loaded: %v", err)
	}

	// フラグの定義
	username := flag.String("username", "", "ユーザー名（必須）")
	password := flag.String("password", "", "パスワード（省略時はプロンプトで入力）")
	help := flag.Bool("help", false, "ヘルプを表示")
	flag.BoolVar(help, "h", false, "ヘルプを表示（短縮形）")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "使い方: adduser [オプション]\n\n")
		fmt.Fprintf(os.Stderr, "goblogの管理者ユーザーを作成します。\n\n")
		fmt.Fprintf(os.Stderr, "オプション:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n例:\n")
		fmt.Fprintf(os.Stderr, "  adduser -username admin                    # パスワードをプロンプトで入力\n")
		fmt.Fprintf(os.Stderr, "  adduser -username admin -password secret   # パスワードを指定（スクリプト用）\n")
		fmt.Fprintf(os.Stderr, "\n環境変数（.envファイルからも読み込み可能）:\n")
		fmt.Fprintf(os.Stderr, "  DATABASE_PATH    - データベースファイルのパス（デフォルト: data/goblog.db）\n")
		fmt.Fprintf(os.Stderr, "  PASSWORD_POLICY  - パスワードポリシー（NONE または STRONG）\n")
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	// ユーザー名の取得（フラグが空ならプロンプト）
	if *username == "" {
		fmt.Print("ユーザー名: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: ユーザー名の読み取りに失敗しました: %v\n", err)
			os.Exit(1)
		}
		*username = strings.TrimSpace(input)
	}

	if *username == "" {
		fmt.Fprintf(os.Stderr, "エラー: ユーザー名は必須です\n")
		os.Exit(1)
	}

	// パスワードの取得（フラグが空ならプロンプト）
	if *password == "" {
		fmt.Print("パスワード: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // 改行
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: パスワードの読み取りに失敗しました: %v\n", err)
			os.Exit(1)
		}
		*password = string(bytePassword)

		// パスワード確認
		fmt.Print("パスワード（確認）: ")
		bytePasswordConfirm, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // 改行
		if err != nil {
			fmt.Fprintf(os.Stderr, "エラー: パスワードの読み取りに失敗しました: %v\n", err)
			os.Exit(1)
		}

		if *password != string(bytePasswordConfirm) {
			fmt.Fprintf(os.Stderr, "エラー: パスワードが一致しません\n")
			os.Exit(1)
		}
	}

	if *password == "" {
		fmt.Fprintf(os.Stderr, "エラー: パスワードは必須です\n")
		os.Exit(1)
	}

	// 設定の読み込み
	cfg := config.Load()

	// データベースの初期化
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: データベースの接続に失敗しました: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// マイグレーションの実行
	if err := db.RunMigrations(database, "migrations/001_create_posts.sql", "migrations/002_create_users.sql"); err != nil {
		fmt.Fprintf(os.Stderr, "エラー: マイグレーションに失敗しました: %v\n", err)
		os.Exit(1)
	}

	// Repository と Service の初期化
	userRepo := repo.NewUserRepository(database)
	sessionStore := auth.NewInMemorySessionStore()
	authService := service.NewAuthService(userRepo, sessionStore, cfg.PasswordPolicy)

	// ユーザーの作成
	user, err := authService.CreateUser(*username, *password)
	if err != nil {
		if errors.Is(err, service.ErrUsernameAlreadyExists) {
			fmt.Fprintf(os.Stderr, "エラー: ユーザー名 '%s' は既に使用されています\n", *username)
		} else if errors.Is(err, service.ErrWeakPassword) {
			fmt.Fprintf(os.Stderr, "エラー: パスワードが弱すぎます\n")
			fmt.Fprintf(os.Stderr, "パスワードポリシー（STRONG）の要件:\n")
			fmt.Fprintf(os.Stderr, "  - 15文字以上\n")
			fmt.Fprintf(os.Stderr, "  - 大文字を含む\n")
			fmt.Fprintf(os.Stderr, "  - 小文字を含む\n")
			fmt.Fprintf(os.Stderr, "  - 数字を含む\n")
			fmt.Fprintf(os.Stderr, "  - 記号を含む\n")
		} else {
			fmt.Fprintf(os.Stderr, "エラー: ユーザーの作成に失敗しました: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("ユーザーを作成しました: username=%s, id=%d\n", user.Username, user.ID)
}
