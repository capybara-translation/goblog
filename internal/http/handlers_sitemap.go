package http

import (
	"encoding/xml"
	"log"
	"net/http"
	"net/url"
)

// XML header
const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>`

// SitemapURLSet is the root element of sitemap.xml
type SitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []SitemapURL `xml:"url"`
}

// SitemapURL is an individual URL element within sitemap.xml
type SitemapURL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

// HandleSitemap generates and returns sitemap.xml
func (h *PublicHandlers) HandleSitemap(w http.ResponseWriter, r *http.Request) {
	// 1. Get published posts (max 10000)
	posts, err := h.postService.GetPublishedPosts(10000, 0)
	if err != nil {
		log.Printf("failed to get published posts for sitemap: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 2. Get tag list
	tagCounts, err := h.postService.GetPublishedTags()
	if err != nil {
		log.Printf("failed to get published tags for sitemap: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 3. Build URL list
	urls := []SitemapURL{
		{Loc: h.baseURL + "/", ChangeFreq: "weekly", Priority: 1.0},
		{Loc: h.baseURL + "/tags", ChangeFreq: "weekly", Priority: 0.6},
	}

	// Post URLs
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

	// Tag URLs
	for tagName := range tagCounts {
		urls = append(urls, SitemapURL{
			Loc:        h.baseURL + "/tags/" + url.PathEscape(tagName),
			ChangeFreq: "weekly",
			Priority:   0.5,
		})
	}

	// 4. Output XML response
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
