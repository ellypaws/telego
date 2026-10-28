package bot

import (
	"errors"
	"fmt"
	"os"
	"runtime/pprof"
	"time"
)

const (
	healthCheckInterval = 30 * time.Second
	// unhealthyLimit leaves discordgo time to notice a dead gateway and reconnect by itself,
	// which takes up to five heartbeat intervals plus its reconnect backoff.
	unhealthyLimit = 5 * time.Minute
)

// watchDiscord reports on b.unhealthy once every Discord health check has failed for
// unhealthyLimit, after writing a goroutine dump that shows where the process is stuck.
func (b *Bot) watchDiscord() {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	var unhealthySince time.Time
	for {
		select {
		case <-b.watchdog.Done():
			return
		case <-ticker.C:
		}

		err := b.Discord.Health()
		if err == nil {
			if !unhealthySince.IsZero() {
				b.Discord.Logger().Info(
					"Discord health check recovered",
					"after", time.Since(unhealthySince).Round(time.Second),
				)
				unhealthySince = time.Time{}
			}
			continue
		}

		if unhealthySince.IsZero() {
			unhealthySince = time.Now()
		}
		down := time.Since(unhealthySince).Round(time.Second)
		b.Discord.Logger().Warn(
			"Discord health check failed",
			"error", err,
			"for", down,
			"limit", unhealthyLimit,
		)
		if down < unhealthyLimit {
			continue
		}

		if file, err := dumpGoroutines(); err != nil {
			b.Discord.Logger().Error("Failed to write goroutine dump", "error", err)
		} else {
			b.Discord.Logger().Error("Discord is unresponsive, wrote goroutine dump", "file", file)
		}
		b.unhealthy <- fmt.Errorf("discord unhealthy for %s: %w", down, err)
		return
	}
}

func dumpGoroutines() (string, error) {
	name := fmt.Sprintf("goroutines-%s.txt", time.Now().Format("20060102-150405"))
	f, err := os.Create(name)
	if err != nil {
		return "", err
	}
	err = pprof.Lookup("goroutine").WriteTo(f, 2)
	return name, errors.Join(err, f.Close())
}
