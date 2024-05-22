package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var ctx = context.Background()

type RedisLock struct {
	clients     []*redis.Client
	lockKey     string
	uniqueValue string
	ttl         time.Duration
}

func NewRedisLock(clients []*redis.Client, lockKey string, ttl time.Duration) *RedisLock {
	return &RedisLock{
		clients:     clients,
		lockKey:     lockKey,
		uniqueValue: uuid.NewString(),
		ttl:         ttl,
	}
}

func (lock *RedisLock) AcquireLock() (bool, error) {
	startTime := time.Now()
	totalLocksAcquired := 0
	MAX_ATTEMPTS := 100
	counter := 0
	for totalLocksAcquired != len(lock.clients) && counter < MAX_ATTEMPTS {
		log.Printf("Attempt [%d] to acquire lock", counter)
		totalLocksAcquired = lock.AcquireLockOnAllRedis(totalLocksAcquired)
		// Check for quorum
		quorum := len(lock.clients)/2 + 1
		if totalLocksAcquired >= quorum && time.Since(startTime) < lock.ttl {
			return true, nil
		}
		// If we didn't get the quorum, release all locks and retry
		// Else just release the lock we couldn't get it faster
		lock.ReleaseLock()
		counter++
	}
	return false, fmt.Errorf("failed to aquire lock : %s", lock.lockKey)
}

func (lock *RedisLock) AcquireLockOnAllRedis(totalLocksAcquired int) int {
	for _, client := range lock.clients {
		result, err := client.SetNX(ctx, lock.lockKey, lock.uniqueValue, lock.ttl).Result()
		if err == nil && result {
			totalLocksAcquired++
		}
	}
	return totalLocksAcquired
}

func (lock *RedisLock) ReleaseLock() {
	// EVAL Atomic Script
	script := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0	
	end	
	`

	// Release everywhere
	for _, client := range lock.clients {
		client.Eval(ctx, script, []string{lock.lockKey}, lock.uniqueValue).Result()
	}
}

func main() {
	clients := []*redis.Client{
		redis.NewClient(&redis.Options{Addr: "localhost:6379"}),
		redis.NewClient(&redis.Options{Addr: "localhost:6380"}),
		redis.NewClient(&redis.Options{Addr: "localhost:6381"}),
		redis.NewClient(&redis.Options{Addr: "localhost:6382"}),
		redis.NewClient(&redis.Options{Addr: "localhost:6383"}),
	}

	instanceID := os.Args[1]

	lock := NewRedisLock(clients, "my_lock_key", 10*time.Second)

	for {
		acquired, err := lock.AcquireLock()
		if err != nil {
			fmt.Printf("Instance %s: Error acquiring lock: %v\n", instanceID, err)
		}

		if acquired {
			fmt.Printf("Instance %s: Lock acquired!\n", instanceID)

			// Simulating production work
			time.Sleep(time.Duration(rand.Intn(5)+1) * time.Second)

			lock.ReleaseLock()
			fmt.Printf("Instance %s: Lock released!\n", instanceID)
		} else {
			startTime := time.Now()
			baseSleep := time.Duration(rand.Intn(10)+1) * time.Second
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond // up to 1 second of jitter
			// Add jitter to the base sleep duration
			sleepDuration := baseSleep + jitter
			time.Sleep(sleepDuration)

			log.Println("Retrying to acquire lock after %d seconds", time.Since(startTime)*time.Second)

		}
	}
}
