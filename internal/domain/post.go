package domain

import (
	"time"
)

// PostStatus は記事のステータスを表します
type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
)

// Post はブログ記事を表すドメインモデルです
type Post struct {
	ID          int64      `json:"id" db:"id"`
	Title       string     `json:"title" db:"title"`
	Slug        string     `json:"slug" db:"slug"`
	Content     string     `json:"content" db:"content"`
	Status      PostStatus `json:"status" db:"status"`
	Tags        string     `json:"tags" db:"tags"` // カンマ区切りで保存（シンプルのため）
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	PublishedAt *time.Time `json:"published_at,omitempty" db:"published_at"` // 公開日時（nilの場合は未公開）
}
