package template

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildCacheKey(t *testing.T) {
	repo := NewPostgresTemplateRepository(nil, time.Minute)
	key := repo.buildCacheKey(uuid.MustParse("11111111-1111-1111-1111-111111111111"), uuid.MustParse("22222222-2222-2222-2222-222222222222"), "welcome", "en")
	want := "11111111-1111-1111-1111-111111111111:22222222-2222-2222-2222-222222222222:welcome:en"
	if key != want {
		t.Fatalf("expected %q, got %q", want, key)
	}
}

func TestTemplateCacheGetSetAndExpiry(t *testing.T) {
	c := &templateCache{items: map[string]*cacheItem{}}
	tmpl := &EmailTemplate{TemplateName: "welcome"}
	c.set("k1", tmpl, time.Hour)
	if got := c.get("k1"); got == nil || got.TemplateName != "welcome" {
		t.Fatalf("expected cached template, got %+v", got)
	}

	c.items["k2"] = &cacheItem{template: tmpl, expiresAt: time.Now().Add(-time.Second)}
	if got := c.get("k2"); got != nil {
		t.Fatalf("expected expired cache miss, got %+v", got)
	}
}

func TestInvalidateCache(t *testing.T) {
	repo := NewPostgresTemplateRepository(nil, time.Minute)
	companyID := uuid.New()
	tenantID := uuid.New()
	match := repo.buildCacheKey(companyID, tenantID, "welcome", "en")
	other := repo.buildCacheKey(companyID, tenantID, "other", "en")
	repo.cache.items[match] = &cacheItem{template: &EmailTemplate{}, expiresAt: time.Now().Add(time.Hour)}
	repo.cache.items[other] = &cacheItem{template: &EmailTemplate{}, expiresAt: time.Now().Add(time.Hour)}

	repo.InvalidateCache(companyID, tenantID, "welcome")
	if _, ok := repo.cache.items[match]; ok {
		t.Fatal("expected matching cache item to be invalidated")
	}
	if _, ok := repo.cache.items[other]; !ok {
		t.Fatal("expected non-matching cache item to remain")
	}
}

func TestTemplateCacheCleanupExpired(t *testing.T) {
	c := &templateCache{items: map[string]*cacheItem{
		"expired": {template: &EmailTemplate{}, expiresAt: time.Now().Add(-time.Second)},
		"active":  {template: &EmailTemplate{}, expiresAt: time.Now().Add(time.Hour)},
	}}
	c.cleanupExpired()
	if _, ok := c.items["expired"]; ok {
		t.Fatal("expected expired item removed")
	}
	if _, ok := c.items["active"]; !ok {
		t.Fatal("expected active item to remain")
	}
}
