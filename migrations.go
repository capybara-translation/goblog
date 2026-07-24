package goblog

// MigrationFiles is the ordered list of embedded migration files, applied at
// startup. Shared by the server and the seed/adduser CLIs so every entrypoint
// brings a fresh DB to the same schema.
var MigrationFiles = []string{
	"migrations/001_create_posts.sql",
	"migrations/002_create_users.sql",
	"migrations/003_add_is_pinned.sql",
	"migrations/004_create_ogp_cache.sql",
	"migrations/005_add_ogp_local_image.sql",
	"migrations/006_add_post_views.sql",
	"migrations/007_create_remember_tokens.sql",
	"migrations/008_create_reactions.sql",
	"migrations/009_add_remember_token_device.sql",
	"migrations/010_add_remember_token_ip.sql",
	"migrations/011_create_health_records.sql",
	"migrations/012_add_post_health_date.sql",
}
