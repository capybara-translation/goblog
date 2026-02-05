package goblog

import (
	"embed"
)

//go:embed web-admin/dist
var AdminSPA embed.FS

//go:embed migrations/*.sql
var Migrations embed.FS

//go:embed internal/view/templates/*.html
var Templates embed.FS

//go:embed internal/view/static/*
var StaticFiles embed.FS
