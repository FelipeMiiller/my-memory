// fakeworker is a test helper binary for the supervisor lifecycle
// tests. It mimics a long-running worker: prints READY on startup
// and stays alive until SIGTERM/SIGINT (graceful shutdown) or
// until FAKEWORKER_EXIT_AFTER_MS elapses (used by heartbeat_lost
// tests where the supervisor must observe an unexpected exit).
//
// Args:
//   - argv[0]: this binary's path (filled in by go build)
//   - FAKEWORKER_EXIT_AFTER_MS (env): integer milliseconds; when set,
//     the process exits 0 after this delay, simulating a crash that
//     the supervisor should detect via worker.heartbeat_lost.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	fmt.Println("READY")

	if raw := os.Getenv("FAKEWORKER_EXIT_AFTER_MS"); raw != "" {
		ms, err := strconv.Atoi(raw)
		if err == nil && ms > 0 {
			time.AfterFunc(time.Duration(ms)*time.Millisecond, func() { os.Exit(0) })
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
}
