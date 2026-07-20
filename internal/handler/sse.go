package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// keepaliveInterval controls how often a comment line is sent to keep the
// connection (and any intermediary proxies) alive during quiet periods.
const keepaliveInterval = 30 * time.Second

// Stream serves the SSE endpoint for a project's live log tail. It subscribes to
// the broker, streams each new log entry as a rendered `log` event, and cleans up
// its subscription when the client disconnects.
func (a *App) Stream(w http.ResponseWriter, r *http.Request) {
	project := a.loadProject(w, r)
	if project == nil {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering
	flusher.Flush()

	events := a.Broker.Subscribe(project.ID)
	defer a.Broker.Unsubscribe(project.ID, events)

	keepalive := time.NewTicker(keepaliveInterval)
	defer keepalive.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-keepalive.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case logEntry, ok := <-events:
			if !ok {
				return
			}
			html, err := a.renderPartial("log-row", logEntry)
			if err != nil {
				log.Printf("sse: render row failed: %v", err)
				continue
			}
			writeSSEEvent(w, "log", html)
			flusher.Flush()
		}
	}
}

// writeSSEEvent writes a named SSE event whose data payload may span multiple
// lines. Each line of the payload is emitted as its own `data:` field per the SSE
// spec, and the event is terminated by a blank line.
func writeSSEEvent(w http.ResponseWriter, event, payload string) {
	fmt.Fprintf(w, "event: %s\n", event)
	for _, line := range strings.Split(payload, "\n") {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	fmt.Fprint(w, "\n")
}
