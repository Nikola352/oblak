package ratelimiter

import (
	"time"
)

type TokenBucket struct {
	maxTokenSize int64
	tokenCount   int64
	timeInterval int64
	timeUpdated  int64
}

func NewTokenBucket(maxSize int64, interval int64) *TokenBucket {
	tokenBucket := TokenBucket{
		maxTokenSize: maxSize,
		tokenCount:   maxSize,
		timeInterval: interval,
		timeUpdated:  Now(),
	}
	return &tokenBucket
}
func (TB *TokenBucket) CheckTokenCondition() bool {
	if isIntervalOver(TB.timeUpdated + TB.timeInterval) {
		TB.timeUpdated = Now()
		TB.tokenCount = TB.maxTokenSize
	}
	if TB.tokenCount <= 0 {
		return false
	}
	TB.tokenCount--
	return true
}

func Now() int64 {
	return time.Now().Unix()
}
func isIntervalOver(time int64) bool {
	return time <= Now()
}
