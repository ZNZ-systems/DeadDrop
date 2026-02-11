package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter is a per-IP token bucket rate limiter. It tracks each visitor by IP
// address and automatically cleans up stale entries.
type Limiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rps      rate.Limit
	burst    int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewLimiter creates a new per-IP rate limiter that allows rps requests per
// second with the given burst size. It starts a background goroutine that
// removes visitors not seen for 5 or more minutes, running every 3 minutes.
func NewLimiter(rps float64, burst int) *Limiter {
	l := &Limiter{
		visitors: make(map[string]*visitor),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
	go l.cleanup()
	return l
}

// Allow reports whether a request from the given IP address should be
// permitted. It creates a new token bucket for the IP if one does not already
// exist.
func (l *Limiter) Allow(ip string) bool {
	l.mu.Lock()
	v, exists := l.visitors[ip]
	if !exists {
		v = &visitor{
			limiter: rate.NewLimiter(l.rps, l.burst),
		}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	l.mu.Unlock()

	return v.limiter.Allow()
}

// cleanup periodically removes visitors that have not been seen for 5 or more
// minutes. It runs in a loop every 3 minutes.
func (l *Limiter) cleanup() {
	for {
		time.Sleep(3 * time.Minute)

		l.mu.Lock()
		for ip, v := range l.visitors {
			if time.Since(v.lastSeen) >= 5*time.Minute {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}
