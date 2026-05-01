package gmcoreview

import (
	"context"
	"html/template"
	"strings"
)

type assetRegistryKey struct{}

type assetEntry struct {
	Key  string
	HTML template.HTML
}

type assetRegistry struct {
	head     []assetEntry
	headSeen map[string]bool
}

const headAssetsPlaceholder = "<!-- GMCORE_HEAD_ASSETS -->"

func withAssetRegistry(ctx context.Context) context.Context {
	if registryFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, assetRegistryKey{}, &assetRegistry{
		head:     []assetEntry{},
		headSeen: map[string]bool{},
	})
}

func registryFromContext(ctx context.Context) *assetRegistry {
	if ctx == nil {
		return nil
	}
	value, _ := ctx.Value(assetRegistryKey{}).(*assetRegistry)
	return value
}

func RegisterHeadAsset(ctx context.Context, key string, html template.HTML) {
	registry := registryFromContext(ctx)
	if registry == nil {
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		key = strings.TrimSpace(string(html))
	}
	if key == "" || registry.headSeen[key] {
		return
	}
	registry.headSeen[key] = true
	registry.head = append(registry.head, assetEntry{Key: key, HTML: html})
}

func renderHeadAssets(ctx context.Context) template.HTML {
	registry := registryFromContext(ctx)
	if registry == nil || len(registry.head) == 0 {
		return template.HTML("")
	}
	var builder strings.Builder
	for _, entry := range registry.head {
		builder.WriteString(string(entry.HTML))
		if !strings.HasSuffix(string(entry.HTML), "\n") {
			builder.WriteString("\n")
		}
	}
	return template.HTML(builder.String())
}
