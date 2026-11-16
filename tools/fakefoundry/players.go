package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// wsTarget turns a table's link into the WebSocket address a Foundry client
// opens. Empty for an empty link.
func wsTarget(link string) string {
	link = strings.TrimRight(strings.TrimSpace(link), "/")
	if link == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(link, "https://"):
		link = "wss://" + strings.TrimPrefix(link, "https://")
	case strings.HasPrefix(link, "http://"):
		link = "ws://" + strings.TrimPrefix(link, "http://")
	case !strings.HasPrefix(link, "ws"):
		link = "wss://" + link
	}
	return link + "/socket.io/?EIO=4&transport=websocket"
}

// players is the headless subcommand: N players join a link and stay until
// Ctrl+C, or leave one by one after -leave.
func players(args []string) {
	flags := flag.NewFlagSet("players", flag.ExitOnError)
	count := flags.Int("n", 3, "how many players join")
	gap := flags.Duration("every", 3*time.Second, "time between joins, and between leaves")
	leave := flags.Duration("leave", 0, "start leaving after this long (default: stay until Ctrl+C)")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/fakefoundry players <link> [-n 3] [-every 3s] [-leave 20s]")
		flags.PrintDefaults()
	}
	var link string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		link, args = args[0], args[1:]
	}
	_ = flags.Parse(args)
	if link == "" && flags.NArg() > 0 {
		link = flags.Arg(0)
	}
	target := wsTarget(link)
	if target == "" {
		flags.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	say := func(format string, args ...any) { fmt.Printf(format+"\n", args...) }

	var wg sync.WaitGroup
	cancels := []context.CancelFunc{}
	for i := 1; i <= *count; i++ {
		playerCtx, cancel := context.WithCancel(ctx)
		cancels = append(cancels, cancel)
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			join(playerCtx, n, target, say)
		}(i)
		if !sleep(ctx, *gap) {
			break
		}
	}

	if *leave > 0 && sleep(ctx, *leave) {
		for i := len(cancels) - 1; i >= 0; i-- {
			cancels[i]()
			if !sleep(ctx, *gap) {
				break
			}
		}
	} else if *leave == 0 {
		say("everyone is at the table; Ctrl+C to leave")
		<-ctx.Done()
	}
	for _, cancel := range cancels {
		cancel()
	}
	wg.Wait()
}

// join opens one player's WebSocket and keeps it until ctx ends.
func join(ctx context.Context, n int, target string, say func(string, ...any)) {
	dial, cancel := context.WithTimeout(ctx, 15*time.Second)
	conn, _, err := websocket.Dial(dial, target, nil)
	cancel()
	if err != nil {
		say("player %d could not join: %v", n, err)
		return
	}
	defer conn.CloseNow()
	// Pongs only arrive while something reads the connection; CloseRead reads
	// in the background and ends ctx when the server hangs up.
	ctx = conn.CloseRead(ctx)

	// A ping every 20 s keeps proxies (Cloudflare closes idle sockets) from
	// dropping the connection.
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = conn.Close(websocket.StatusNormalClosure, "left the table")
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil && ctx.Err() == nil {
				say("player %d was disconnected: %v", n, err)
				return
			}
		}
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
