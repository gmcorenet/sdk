package gmcore_crawler

import (
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.UserAgent != "GmCore-Crawler/1.0" {
		t.Errorf("expected default user agent, got %s", opts.UserAgent)
	}

	if opts.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", opts.Timeout)
	}

	if opts.MaxDepth != 3 {
		t.Errorf("expected max depth 3, got %d", opts.MaxDepth)
	}

	if opts.MaxConcurrent != 5 {
		t.Errorf("expected max concurrent 5, got %d", opts.MaxConcurrent)
	}
}

func TestNewCrawler(t *testing.T) {
	c := New(nil)
	if c == nil {
		t.Fatal("expected non-nil crawler")
	}
	if c.client == nil {
		t.Error("expected non-nil http client")
	}
	if c.options == nil {
		t.Error("expected non-nil options")
	}
}

func TestBuilder(t *testing.T) {
	b := NewBuilder()
	b.UserAgent("TestBot/1.0").
		Timeout(10 * time.Second).
		MaxDepth(5).
		AllowedDomains("example.com", "test.com").
		MaxConcurrent(10).
		Delay(200 * time.Millisecond)

	c := b.Build()
	if c.options.UserAgent != "TestBot/1.0" {
		t.Errorf("expected TestBot/1.0, got %s", c.options.UserAgent)
	}
	if c.options.Timeout != 10*time.Second {
		t.Errorf("expected 10s timeout, got %v", c.options.Timeout)
	}
	if c.options.MaxDepth != 5 {
		t.Errorf("expected max depth 5, got %d", c.options.MaxDepth)
	}
	if len(c.options.AllowedDomains) != 2 {
		t.Errorf("expected 2 allowed domains, got %d", len(c.options.AllowedDomains))
	}
	if c.options.MaxConcurrent != 10 {
		t.Errorf("expected max concurrent 10, got %d", c.options.MaxConcurrent)
	}
}

func TestMakeAbsolute(t *testing.T) {
	c := New(nil)

	tests := []struct {
		link    string
		base    string
		want    string
	}{
		{"/path", "http://example.com/page", "http://example.com/path"},
		{"path", "http://example.com/dir/", "http://example.com/dir/path"},
		{"http://other.com/", "http://example.com/", "http://other.com/"},
		{"#anchor", "http://example.com/", ""},
		{"javascript:void(0)", "http://example.com/", ""},
		{"mailto:test@example.com", "http://example.com/", ""},
	}

	for _, tt := range tests {
		got := c.makeAbsolute(tt.link, tt.base)
		if got != tt.want {
			t.Errorf("makeAbsolute(%q, %q) = %q, want %q", tt.link, tt.base, got, tt.want)
		}
	}
}

func TestIsAllowedDomain(t *testing.T) {
	c := New(&Options{
		AllowedDomains: []string{"example.com"},
		BlockedDomains: []string{"blocked.com"},
	})

	if !c.isAllowedDomain("example.com") {
		t.Error("example.com should be allowed")
	}
	if !c.isAllowedDomain("sub.example.com") {
		t.Error("sub.example.com should be allowed")
	}
	if c.isAllowedDomain("other.com") {
		t.Error("other.com should not be allowed")
	}
	if c.isAllowedDomain("blocked.com") {
		t.Error("blocked.com should be blocked")
	}
	if c.isAllowedDomain("sub.blocked.com") {
		t.Error("sub.blocked.com should be blocked")
	}
}

func TestExtractLinks(t *testing.T) {
	c := New(nil)
	content := []byte(`<html>
		<body>
			<a href="/page1">Page 1</a>
			<a href="/page2">Page 2</a>
			<img src="/image.jpg">
			<script src="/app.js"></script>
		</body>
	</html>`)

	links := c.extractLinks(content, "http://example.com")
	if len(links) < 2 {
		t.Errorf("expected at least 2 links, got %d", len(links))
	}
}

func TestLinkExtractor(t *testing.T) {
	extractor := NewLinkExtractor()
	content := []byte(`<a href="/page1">Link 1</a>
		<a href="/page2">Link 2</a>
		<img src="/image.jpg">`)

	links := extractor.Extract(content)
	if len(links) < 3 {
		t.Errorf("expected at least 3 links, got %d", len(links))
	}
}

func TestLinkExtractorWithBaseURL(t *testing.T) {
	extractor := NewLinkExtractor()
	extractor.WithBaseURL("http://example.com")

	content := []byte(`<a href="/page1">Link 1</a>`)
	links := extractor.Extract(content)

	if len(links) != 1 {
		t.Errorf("expected 1 link, got %d", len(links))
	}
}

func TestParseRobotsTxt(t *testing.T) {
	content := []byte(`User-Agent: *
Disallow: /private/
Allow: /public/
Sitemap: http://example.com/sitemap.xml`)

	robots := ParseRobotsTxt(content)

	if len(robots.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(robots.Rules))
	}

	if !robots.IsAllowed("/public/") {
		t.Error("/public/ should be allowed")
	}

	if robots.IsAllowed("/private/") {
		t.Error("/private/ should not be allowed")
	}

	if len(robots.Sitemaps) != 1 {
		t.Errorf("expected 1 sitemap, got %d", len(robots.Sitemaps))
	}
}

func TestParseRobotsTxtEmptyDisallow(t *testing.T) {
	content := []byte(`User-Agent: *
Disallow:`)

	robots := ParseRobotsTxt(content)
	if !robots.IsAllowed("/anypath/") {
		t.Error("with empty disallow, /anypath/ should be allowed")
	}
}
