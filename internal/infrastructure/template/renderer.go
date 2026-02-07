package template

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/redis"
)

// TemplateRenderer renders templates from files, caching raw template strings in Redis for performance.
type TemplateRenderer struct {
	basePath  string
	cache     ports.Cache
	redisCtx  context.Context
}

// cleanAndJoinPath ensures the path is joined safely and cleaned using the basePath and templateName.
func cleanAndJoinPath(basePath, templateName string) string {
	// Ensure basePath and templateName are joined with the correct path separator, and cleaned.
	basePath = strings.TrimRight(basePath, string(os.PathSeparator)+"/")
	templateName = strings.TrimLeft(templateName, string(os.PathSeparator)+"/")
	p := filepath.Join(basePath, templateName)
	return filepath.Clean(p)
}

// NewTemplateRenderer creates a new TemplateRenderer with a base path (e.g., "templates/emails" or "templates\emails").
func NewTemplateRenderer(basePath string, redisCache ports.Cache) (ports.TemplateRenderer, error) {
	return &TemplateRenderer{
		basePath: basePath,
		cache:    redisCache,
		redisCtx: context.Background(),
	}, nil
}

// Render renders a template with the given name and data.
// It tries to get the raw template from Redis; if cache miss, reads from disk and stores in Redis.
func (r *TemplateRenderer) Render(templateName string, data map[string]string) (string, error) {
	cacheKey := "tmpl:" + templateName

	var tmplSource string

	// 1. Try to retrieve template content from Redis cache.
	err := r.cache.Get(r.redisCtx, cacheKey, &tmplSource)
	if err != nil {
		// If the error indicates the template was not found in cache, let it fall through to disk loading.
		// The correct comparison for ErrorCacheMiss must use the exported package path for redis.ErrorCacheMiss.
		if errors.Is(err, redis.ErrorCacheMiss) {
			tmplSource = ""
		} else {
			return "", fmt.Errorf("failed to get template from redis: %w", err)
		}
	}

	if tmplSource == "" {
		// 2. Cache miss: Read template file from the disk.
		path := cleanAndJoinPath(r.basePath, templateName)
		fileBytes, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read template file: %w", err)
		}
		tmplSource = string(fileBytes)
		// Store raw template string for next time.
		const templateCacheExpiration = 10 * 24 * time.Hour
		if err := r.cache.Set(r.redisCtx, cacheKey, tmplSource, templateCacheExpiration); err != nil {
			return "", fmt.Errorf("failed to cache template in redis: %w", err)
		}
	}

	// 3. Parse the template safely.
	parsedTmpl, err := template.New(filepath.Base(templateName)).Parse(tmplSource)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	mapData := make(map[string]interface{}, len(data))
	for k, v := range data {
		mapData[k] = v
	}

	var buf bytes.Buffer
	if err := parsedTmpl.Execute(&buf, mapData); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

/*
How this works:
- The renderer first tries to get the raw template content from the Redis cache for performance.
- If not present in Redis, it loads from disk and stores in Redis for future renderings.
- The actual Go compiled template is generated and used for each render since compiled templates cannot be marshalled directly to Redis.
- This pattern reduces disk I/O and serves templates rapidly, especially in distributed systems or multi-instance deployments.
*/
