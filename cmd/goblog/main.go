package main

import (
	"fmt"
	"log"
	"net/http"

	gobloghttp "github.com/capybara-translation/goblog/internal/http"
)

func main() {
	// ルーターの初期化
	r := gobloghttp.NewRouter()

	// サーバー起動
	port := ":8080"
	fmt.Printf("Server starting on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}