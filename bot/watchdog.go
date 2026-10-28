package bot

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime/pprof"
	"time"

	"telegram-discord/lib"

	"github.com/bwmarrin/discordgo"
)

const (
	healthCheckInterval = 30 * time.Second
	// unhealthyLimit leaves discordgo time to notice a dead gateway and reconnect by itself,
	// which takes up to five heartbeat intervals plus its reconnect backoff.
	unhealthyLimit = 5 * time.Minute

	// restartFile records why the watchdog gave up, so the next start can report it.
	restartFile          = "watchdog-restart.txt"
	restartNotifyTimeout = 15 * time.Second
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

		reason := fmt.Errorf("discord unhealthy for %s: %w", down, err)
		if file, err := dumpGoroutines(); err != nil {
			b.Discord.Logger().Error("Failed to write goroutine dump", "error", err)
		} else {
			b.Discord.Logger().Error("Discord is unresponsive, wrote goroutine dump", "file", file)
			reason = fmt.Errorf("%w, goroutine dump in %s", reason, file)
		}
		if err := os.WriteFile(restartFile, []byte(reason.Error()), 0o644); err != nil {
			b.Discord.Logger().Error("Failed to record watchdog restart", "error", err)
		}
		b.unhealthy <- reason
		return
	}
}

// notifyRestart reports a restart the watchdog caused in the previous run, then removes the
// record so it is reported once.
func (b *Bot) notifyRestart() {
	reason, err := os.ReadFile(restartFile)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		b.Discord.Logger().Error("Failed to read watchdog restart record", "error", err)
		return
	}
	if err := os.Remove(restartFile); err != nil {
		b.Discord.Logger().Warn("Failed to remove watchdog restart record", "error", err)
	}
	b.Discord.Logger().Warn("Restarted by the watchdog", "reason", string(reason))

	if len(b.notifyOnRestart.Users) == 0 && len(b.notifyOnRestart.Channels) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), restartNotifyTimeout)
	defer cancel()
	embeds := lib.ErrorEmbed("Watchdog restart", string(reason))
	if err := b.notifyOnRestart.Notify(b.Discord.Session, embeds, discordgo.WithContext(ctx)); err != nil {
		b.Discord.Logger().Error("Failed to send restart notification", "error", err)
		return
	}
	b.Discord.Logger().Info("Sent restart notification")
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
