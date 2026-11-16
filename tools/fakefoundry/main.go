// Command fakefoundry stands in for Foundry VTT while you work on ftm, so you
// can try detection, the world name and player counting without installing
// Foundry.
//
//	go run ./tools/fakefoundry                      # Foundry on :30000 with a world launched
//	go run ./tools/fakefoundry -port 30004 -world tomb-of-annihilation
//	go run ./tools/fakefoundry -setup               # open, but no world launched
//	go run ./tools/fakefoundry players <link> -n 4  # four players join a table, no view
//
// The server answers what ftm looks at: a page titled "Foundry Virtual
// Tabletop" (a real page you can open), /api/status with the same fields as
// Foundry 14, and WebSocket upgrades it keeps open. Every open WebSocket is a
// player, which is how a browser in a Foundry world behaves.
//
// In the terminal view, paste the table's public link (l) and add players (+):
// they join through the tunnel, so ftm counts them, the way real players do.
package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	tea "charm.land/bubbletea/v2"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "players" {
		players(os.Args[2:])
		return
	}
	serve(os.Args[1:])
}

func serve(args []string) {
	flags := flag.NewFlagSet("fakefoundry", flag.ExitOnError)
	port := flags.Int("port", 30000, "port to listen on, like Foundry's own")
	world := flags.String("world", worlds[0].id, "world id to report")
	version := flags.String("version", "14.368", "Foundry version to report")
	setup := flags.Bool("setup", false, "start on the setup screen, with no world launched")
	link := flags.String("link", "", "the table's public link, so + joins players through the tunnel")
	plain := flags.Bool("plain", false, "print events instead of the terminal view")
	_ = flags.Parse(args)

	t := newTable(*version, *world, *setup, *port)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "fakefoundry: port %d: %v\n", *port, err)
		os.Exit(1)
	}
	server := &http.Server{Handler: t.handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()
	defer t.dropBots()

	if *plain {
		t.notify = func() {
			t.mu.Lock()
			if len(t.log) > 0 {
				fmt.Println(t.log[len(t.log)-1])
			}
			t.mu.Unlock()
		}
		fmt.Printf("fake Foundry v%s on http://localhost:%d\n", t.version, *port)
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)
		<-stop
		return
	}

	program := tea.NewProgram(newView(t, *link))
	t.notify = func() { program.Send(refresh{}) }
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
