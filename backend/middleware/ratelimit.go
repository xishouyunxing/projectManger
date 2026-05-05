package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type visitor struct {
	lastSeen time.Time
	tokens   float64
}

// RateLimiter 基于令牌桶算法的 IP 级限流中间件。
// rate: 每秒允许的请求数；burst: 突发容量。
func RateLimiter(rate float64, burst int) gin.HandlerFunc {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*visitor)
	)

	// 定期清理过期记录
	go func() {
		for range time.Tick(3 * time.Minute) {
			mu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 5*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		v, exists := visitors[ip]
		if !exists {
			v = &visitor{tokens: float64(burst)}
			visitors[ip] = v
		}

		elapsed := time.Since(v.lastSeen).Seconds()
		v.tokens += elapsed * rate
		if v.tokens > float64(burst) {
			v.tokens = float64(burst)
		}
		v.lastSeen = time.Now()

		if v.tokens < 1 {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
			})
			return
		}
		v.tokens--
		mu.Unlock()

		c.Next()
	}
}
