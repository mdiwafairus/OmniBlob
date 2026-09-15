package api

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type failedAttempt struct {
	count     int
	firstFail time.Time
	lastFail  time.Time
}

var (
	banList       = sync.Map{}
	failedTracker = sync.Map{}
)

const (
	MaxFailures   = 5
	BanDuration   = 30 * time.Minute
	TrackDuration = 5 * time.Minute
)

// IsIPBlocked returns true if IP is currently banned
func IsIPBlocked(ip string) bool {
	if val, ok := banList.Load(ip); ok {
		banExpiry := val.(time.Time)
		if time.Now().Before(banExpiry) {
			return true
		}
		// Ban expired, remove it
		banList.Delete(ip)
	}
	return false
}

// RecordFailedAttempt records a failure and returns true if the IP just got banned
func RecordFailedAttempt(ip, reason string, dbPool *pgxpool.Pool) bool {
	now := time.Now()
	
	val, ok := failedTracker.Load(ip)
	if !ok {
		failedTracker.Store(ip, failedAttempt{count: 1, firstFail: now, lastFail: now})
		return false
	}

	attempt := val.(failedAttempt)
	
	// Reset tracker if its been too long since first failure
	if now.Sub(attempt.firstFail) > TrackDuration {
		failedTracker.Store(ip, failedAttempt{count: 1, firstFail: now, lastFail: now})
		return false
	}

	attempt.count++
	attempt.lastFail = now

	if attempt.count >= MaxFailures {
		// Ban the IP
		expiresAt := now.Add(BanDuration)
		banList.Store(ip, expiresAt)
		failedTracker.Delete(ip)

		// Insert into database if available
		if dbPool != nil {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_, _ = dbPool.Exec(ctx, 
					"INSERT INTO blocked_ip (ip_address, reason, blocked_at, expires_at) VALUES ($1, $2, $3, $4)",
					ip, reason, now, expiresAt)
			}()
		}

		return true // Indicates that it just got banned
	}

	failedTracker.Store(ip, attempt)
	return false
}

