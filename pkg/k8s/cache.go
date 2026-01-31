// Package k8s provides Kubernetes cluster interaction capabilities.
// This file contains caching functionality for cluster status.
package k8s

import (
	"time"
)

// getCachedStatus retrieves a cached cluster status if it exists and is not expired
func (p *Provider) getCachedStatus(contextName string) *ClusterStatus {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	cached, exists := p.cache[contextName]
	if !exists {
		return nil
	}

	if time.Now().After(cached.ExpiresAt) {
		// Cache expired
		return nil
	}

	return cached.Status
}

// cacheStatus stores a cluster status in the cache
func (p *Provider) cacheStatus(contextName string, status *ClusterStatus) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	p.cache[contextName] = &CachedClusterStatus{
		Status:    status,
		ExpiresAt: time.Now().Add(p.cacheTTL),
	}
}

// ClearCache clears all cached cluster statuses
func (p *Provider) ClearCache() {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	p.cache = make(map[string]*CachedClusterStatus)
}

// SetCacheTTL sets the cache time-to-live duration
func (p *Provider) SetCacheTTL(ttl time.Duration) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	p.cacheTTL = ttl
}
