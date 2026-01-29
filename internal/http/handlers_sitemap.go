package http

import (
	"encoding/xml"
	"log"
	"net/http"
	"net/url"
)

// XMLヘッダー
const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`

// SitemapURLSet はsitemap.xmlのルート要素
type SitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []SitemapURL `xml:"url"`
}

// SitemapURL はsitemap.xml内の個別URL要素
type SitemapURL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

// HandleSitemap はsitemap.xmlを生成して返します
func (h *PublicHandlers) HandleSitemap(w http.ResponseWriter, r *http.Request) {
	// 1. 公開済み記事を取得（最大10000件）
	posts, err := h.postService.GetPublishedPosts(10000, 0)
	if err != nil {
		log.Printf("failed to get published posts for sitemap: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 2. タグ一覧を取得
	tagCounts, err := h.postService.GetPublishedTags()
	if err != nil {
		log.Printf("failed to get published tags for sitemap: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 3. URL一覧を構築
	urls := []SitemapURL{
		{Loc: h.baseURL + "/", ChangeFreq: "weekly", Priority: 1.0},
		{Loc: h.baseURL + "/posts", ChangeFreq: "daily", Priority: 0.8},
		{Loc: h.baseURL + "/tags", ChangeFreq: "weekly", Priority: 0.6},
	}

	// 記事URL
	for _, post := range posts {
		lastMod := ""
		if !post.UpdatedAt.IsZero() {
			lastMod = post.UpdatedAt.UTC().Format("2006-01-02")
		}
		urls = append(urls, SitemapURL{
			Loc:        h.baseURL + "/posts/" + post.Slug,
			LastMod:    lastMod,
			ChangeFreq: "monthly",
			Priority:   0.7,
		})
	}

	// タグURL
	for tagName := range tagCounts {
		urls = append(urls, SitemapURL{
			Loc:        h.baseURL + "/tags/" + url.PathEscape(tagName),
			ChangeFreq: "weekly",
			Priority:   0.5,
		})
	}

	// 4. XMLレスポンスを出力
	urlset := SitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(xmlHeader + "\n"))
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(urlset); err != nil {
		log.Printf("failed to encode sitemap XML: %v", err)
	}
}
