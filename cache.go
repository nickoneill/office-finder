package main

import (
	"encoding/json"
	"os"
	"time"
)

const CacheFile = "cache/contact-urls.json"

// CacheEntry represents a cached contact URL for a legislator
type CacheEntry struct {
	ContactURL  string    `json:"contact_url"`
	LastChecked time.Time `json:"last_checked"`
}

// URLCache is a map of bioguide IDs to their cached contact URLs
type URLCache map[string]CacheEntry

// LoadCache loads the URL cache from disk
func LoadCache() (URLCache, error) {
	cache := make(URLCache)

	data, err := os.ReadFile(CacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil // Return empty cache if file doesn't exist
		}
		return cache, err
	}

	if err := json.Unmarshal(data, &cache); err != nil {
		return cache, err
	}

	return cache, nil
}

// SaveCache saves the URL cache to disk
func SaveCache(cache URLCache) error {
	// Ensure cache directory exists
	if err := os.MkdirAll("cache", 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(CacheFile, data, 0644)
}

// GetCachedURL returns the cached URL for a bioguide ID if it exists
func (c URLCache) GetCachedURL(bioguide string) (string, bool) {
	entry, ok := c[bioguide]
	if !ok {
		return "", false
	}
	return entry.ContactURL, true
}

// SetCachedURL sets or updates the cached URL for a bioguide ID
func (c URLCache) SetCachedURL(bioguide, url string) {
	c[bioguide] = CacheEntry{
		ContactURL:  url,
		LastChecked: time.Now(),
	}
}
