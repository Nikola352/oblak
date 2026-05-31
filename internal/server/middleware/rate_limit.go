package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	ginlimiter "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func RateLimitByIP(requestLimit, timeLimit string) gin.HandlerFunc {
	rate, err := limiter.NewRateFromFormatted(fmt.Sprintf("%s-%s", requestLimit, timeLimit))
	if err != nil {
		panic(err)
	}

	store := memory.NewStore()

	instance := limiter.New(store, rate)

	return ginlimiter.NewMiddleware(instance,
		ginlimiter.WithKeyGetter(func(c *gin.Context) string {
			return c.ClientIP()
		}),
	)
}
