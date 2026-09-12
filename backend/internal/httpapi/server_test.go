package httpapi

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServeStopsAfterContextCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	server := NewServer(listener.Addr().String(), http.NotFoundHandler())
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, server, listener, time.Second)
	}()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not stop after context cancellation")
	}
}
