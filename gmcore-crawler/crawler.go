package gmcore_crawler

// Package gmcore_crawler provides web crawling and scraping utilities with
// support for robots.txt, rate limiting, and concurrent fetching.
//
// Examples:
//
//	// Simple crawl
//	crawler := NewBuilder().
//	    AllowedDomains("example.com").
//	    MaxDepth(3).
//	    Build()
//
//	results, _ := crawler.Crawl("https://example.com")
//	for page := range results {
//	    fmt.Printf("URL: %s, Status: %d\n", page.URL, page.StatusCode)
//	}
//
//	// Extract links from HTML
//	extractor := NewLinkExtractor()
//	links := extractor.Extract(htmlContent)

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Options struct {
	UserAgent       string
	Timeout         time.Duration
	MaxDepth        int
	AllowedDomains  []string
	BlockedDomains  []string
	RespectRobotsTxt bool
	MaxConcurrent   int
	Delay           time.Duration
	StartDelay      time.Duration
}

type Page struct {
	URL         string
	StatusCode  int
	Headers     http.Header
	Content     []byte
	Links       []string
	ExternalLinks []string
	Depth       int
	FetchTime   time.Duration
	Error       error
}

type Crawler struct {
	client   *http.Client
	options  *Options
	visited  map[string]bool
	visitedMu sync.RWMutex
	results  chan *Page
	wg       sync.WaitGroup
	rateLimiter *time.Ticker
	stopChan chan struct{}
}

func DefaultOptions() *Options {
	return &Options{
		UserAgent:    "GmCore-Crawler/1.0",
		Timeout:      30 * time.Second,
		MaxDepth:     3,
		MaxConcurrent: 5,
		Delay:        100 * time.Millisecond,
	}
}

func New(opts *Options) *Crawler {
	if opts == nil {
		opts = DefaultOptions()
	}

	client := &http.Client{
		Timeout: opts.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	c := &Crawler{
		client:   client,
		options:  opts,
		visited:  make(map[string]bool),
		results:  make(chan *Page, 100),
		stopChan: make(chan struct{}),
	}

	if opts.Delay > 0 {
		c.rateLimiter = time.NewTicker(opts.Delay)
	}

	return c
}

func (c *Crawler) Crawl(startURL string) (<-chan *Page, error) {
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start URL: %w", err)
	}

	if !c.isAllowedDomain(parsedURL.Host) {
		return nil, fmt.Errorf("start URL domain not allowed: %s", parsedURL.Host)
	}

	c.visited[parsedURL.String()] = true

	sem := make(chan struct{}, c.options.MaxConcurrent)
	c.wg.Add(1)
	go c.crawlURL(parsedURL.String(), 0, sem)

	go func() {
		c.wg.Wait()
		close(c.results)
	}()

	return c.results, nil
}

func (c *Crawler) crawlURL(targetURL string, depth int, sem chan struct{}) {
	defer c.wg.Done()
	defer func() { <-sem }()

	select {
	case <-c.stopChan:
		return
	default:
	}

	if c.rateLimiter != nil {
		<-c.rateLimiter.C
	}

	sem <- struct{}{}

	page := &Page{
		URL:   targetURL,
		Depth: depth,
	}

	startTime := time.Now()

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		page.Error = fmt.Errorf("failed to create request: %w", err)
		page.FetchTime = time.Since(startTime)
		c.results <- page
		return
	}

	req.Header.Set("User-Agent", c.options.UserAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		page.Error = fmt.Errorf("failed to fetch: %w", err)
		page.FetchTime = time.Since(startTime)
		c.results <- page
		return
	}
	defer resp.Body.Close()

	page.StatusCode = resp.StatusCode
	page.Headers = resp.Header

	content, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		page.Error = fmt.Errorf("failed to read body: %w", err)
		page.FetchTime = time.Since(startTime)
		c.results <- page
		return
	}

	page.Content = content
	page.FetchTime = time.Since(startTime)

	links := c.extractLinks(content, targetURL)
	page.Links = links

	var externalLinks []string
	for _, link := range links {
		parsed, err := url.Parse(link)
		if err != nil {
			continue
		}
		if parsed.Host != "" && parsed.Host != urlParse(targetURL).Host {
			externalLinks = append(externalLinks, link)
		}
	}
	page.ExternalLinks = externalLinks

	c.results <- page

	if depth < c.options.MaxDepth {
		for _, link := range links {
			absoluteURL := c.makeAbsolute(link, targetURL)
			if absoluteURL == "" {
				continue
			}

			c.visitedMu.Lock()
			if c.visited[absoluteURL] {
				c.visitedMu.Unlock()
				continue
			}
			c.visited[absoluteURL] = true
			c.visitedMu.Unlock()

			parsed, _ := url.Parse(absoluteURL)
			if parsed == nil || !c.isAllowedDomain(parsed.Host) {
				continue
			}

			c.wg.Add(1)
			go c.crawlURL(absoluteURL, depth+1, sem)
		}
	}
}

func (c *Crawler) extractLinks(content []byte, baseURL string) []string {
	linkPattern := regexp.MustCompile(`(?i)href=["']?([^"'\s>]+)["']?\s*`)
	imgPattern := regexp.MustCompile(`(?i)src=["']?([^"'\s>]+)["']?\s*`)

	var links []string

	for _, match := range linkPattern.FindAllSubmatch(content, -1) {
		if len(match) > 1 {
			links = append(links, string(match[1]))
		}
	}

	for _, match := range imgPattern.FindAllSubmatch(content, -1) {
		if len(match) > 1 {
			links = append(links, string(match[1]))
		}
	}

	seen := make(map[string]bool)
	var unique []string
	for _, link := range links {
		if !seen[link] {
			seen[link] = true
			unique = append(unique, link)
		}
	}

	return unique
}

func (c *Crawler) makeAbsolute(link, baseURL string) string {
	if link == "" || strings.HasPrefix(link, "#") || strings.HasPrefix(link, "javascript:") || strings.HasPrefix(link, "mailto:") {
		return ""
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	absolute, err := base.Parse(link)
	if err != nil {
		return ""
	}

	if absolute.Scheme != "http" && absolute.Scheme != "https" {
		return ""
	}

	return absolute.String()
}

func (c *Crawler) isAllowedDomain(domain string) bool {
	if c.options.AllowedDomains != nil && len(c.options.AllowedDomains) > 0 {
		allowed := false
		for _, d := range c.options.AllowedDomains {
			if d == domain || strings.HasSuffix(domain, "."+d) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	for _, d := range c.options.BlockedDomains {
		if d == domain || strings.HasSuffix(domain, "."+d) {
			return false
		}
	}

	return true
}

func (c *Crawler) Stop() {
	close(c.stopChan)
}

func (c *Crawler) Reset() {
	c.visitedMu.Lock()
	c.visited = make(map[string]bool)
	c.visitedMu.Unlock()
}

func urlParse(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

type Builder struct {
	crawler *Crawler
}

func NewBuilder() *Builder {
	return &Builder{
		crawler: New(DefaultOptions()),
	}
}

func (b *Builder) UserAgent(ua string) *Builder {
	b.crawler.options.UserAgent = ua
	return b
}

func (b *Builder) Timeout(t time.Duration) *Builder {
	b.crawler.options.Timeout = t
	return b
}

func (b *Builder) MaxDepth(d int) *Builder {
	b.crawler.options.MaxDepth = d
	return b
}

func (b *Builder) AllowedDomains(domains ...string) *Builder {
	b.crawler.options.AllowedDomains = domains
	return b
}

func (b *Builder) MaxConcurrent(n int) *Builder {
	b.crawler.options.MaxConcurrent = n
	return b
}

func (b *Builder) Delay(d time.Duration) *Builder {
	b.crawler.options.Delay = d
	return b
}

func (b *Builder) Build() *Crawler {
	return b.crawler
}

type LinkExtractor struct {
	includeImages bool
	includeScripts bool
	onlyExternal  bool
	baseURL       string
}

func NewLinkExtractor() *LinkExtractor {
	return &LinkExtractor{
		includeImages: true,
	}
}

func (e *LinkExtractor) WithBaseURL(baseURL string) *LinkExtractor {
	e.baseURL = baseURL
	return e
}

func (e *LinkExtractor) Extract(content []byte) []string {
	var links []string

	hrefPattern := regexp.MustCompile(`(?i)href=["']?([^"'\s>]+)["']?`)
	for _, match := range hrefPattern.FindAllSubmatch(content, -1) {
		if len(match) > 1 {
			links = append(links, string(match[1]))
		}
	}

	if e.includeImages {
		srcPattern := regexp.MustCompile(`(?i)src=["']?([^"'\s>]+)["']?`)
		for _, match := range srcPattern.FindAllSubmatch(content, -1) {
			if len(match) > 1 {
				links = append(links, string(match[1]))
			}
		}
	}

	return links
}

func (e *LinkExtractor) ExtractWithText(content []byte) map[string]string {
	result := make(map[string]string)

	pattern := regexp.MustCompile(`(?i)<a[^>]+href=["']?([^"'\s>]+)["']?[^>]*>([^<]*)</a>`)
	for _, match := range pattern.FindAllSubmatch(content, -1) {
		if len(match) > 2 {
			href := string(match[1])
			text := string(match[2])
			text = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(text, " "))
			result[href] = text
		}
	}

	return result
}

type RobotsTxt struct {
	Rules    map[string]bool
	Sitemaps []string
}

func ParseRobotsTxt(content []byte) *RobotsTxt {
	robots := &RobotsTxt{
		Rules: make(map[string]bool),
	}

	var currentUserAgent string

	for _, line := range bytes.Split(content, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || bytes.HasPrefix(line, []byte("#")) {
			continue
		}

		parts := bytes.SplitN(line, []byte(":"), 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(string(parts[0]))
		value := strings.TrimSpace(string(parts[1]))

		switch strings.ToLower(key) {
		case "user-agent":
			currentUserAgent = strings.ToLower(value)
		case "disallow":
			if currentUserAgent == "*" || currentUserAgent == "gmcore-crawler" {
				if value != "" {
					robots.Rules[value] = false
				}
			}
		case "allow":
			if currentUserAgent == "*" || currentUserAgent == "gmcore-crawler" {
				if value != "" {
					robots.Rules[value] = true
				}
			}
		case "sitemap":
			robots.Sitemaps = append(robots.Sitemaps, value)
		}
	}

	return robots
}

func (r *RobotsTxt) IsAllowed(path string) bool {
	if allowed, ok := r.Rules[path]; ok {
		return allowed
	}

	longestMatch := ""
	for rule := range r.Rules {
		if strings.HasPrefix(path, rule) && len(rule) > len(longestMatch) {
			longestMatch = rule
		}
	}

	if longestMatch != "" {
		return r.Rules[longestMatch]
	}

	return true
}
