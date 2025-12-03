// Package main provides a mock webhook receiver for testing listmonk webhooks.
// It logs received webhooks and verifies HMAC signatures.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// WebhookEvent represents a received webhook event.
type WebhookEvent struct {
	Event         string    `json:"event"`
	Timestamp     time.Time `json:"timestamp"`
	Data          any       `json:"data"`
	ReceivedAt    time.Time `json:"received_at"`
	SignatureValid bool     `json:"signature_valid"`
}

// Server holds the webhook receiver state.
type Server struct {
	secret string
	events []WebhookEvent
	mu     sync.RWMutex
}

func main() {
	secret := os.Getenv("WEBHOOK_SECRET")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	srv := &Server{
		secret: secret,
		events: make([]WebhookEvent, 0),
	}

	http.HandleFunc("/webhook", srv.handleWebhook)
	http.HandleFunc("/events", srv.handleEvents)
	http.HandleFunc("/clear", srv.handleClear)
	http.HandleFunc("/health", srv.handleHealth)

	log.Printf("Webhook receiver starting on port %s", port)
	if secret != "" {
		log.Printf("HMAC signature verification enabled")
	} else {
		log.Printf("HMAC signature verification disabled (no WEBHOOK_SECRET set)")
	}

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// handleWebhook receives and logs webhook events.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Error reading body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify signature if secret is configured.
	signatureValid := true
	if s.secret != "" {
		sig := r.Header.Get("X-Webhook-Signature")
		signatureValid = s.verifySignature(body, sig)
		if signatureValid {
			log.Printf("signature: valid")
		} else {
			log.Printf("signature: INVALID (expected sha256=%s, got %s)",
				computeHMAC(body, s.secret), sig)
		}
	}

	// Parse the payload.
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	event.ReceivedAt = time.Now()
	event.SignatureValid = signatureValid

	// Store the event.
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()

	// Log the event.
	log.Printf("Received event: %s", event.Event)
	log.Printf("  Timestamp: %s", event.Timestamp.Format(time.RFC3339))
	log.Printf("  Data: %v", event.Data)

	// Pretty print the full payload.
	prettyJSON, _ := json.MarshalIndent(event, "  ", "  ")
	log.Printf("  Full payload:\n  %s", string(prettyJSON))

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "received", "event": "%s"}`, event.Event)
}

// handleEvents returns all received events as JSON.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter by event type if specified.
	eventType := r.URL.Query().Get("event")

	var filtered []WebhookEvent
	if eventType != "" {
		for _, e := range s.events {
			if e.Event == eventType {
				filtered = append(filtered, e)
			}
		}
	} else {
		filtered = s.events
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"count":  len(filtered),
		"events": filtered,
	})
}

// handleClear clears all stored events.
func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	s.events = make([]WebhookEvent, 0)
	s.mu.Unlock()

	log.Printf("Events cleared")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "cleared"}`)
}

// handleHealth returns health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	count := len(s.events)
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "healthy", "events_received": %d}`, count)
}

// verifySignature verifies the HMAC-SHA256 signature.
func (s *Server) verifySignature(payload []byte, signature string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	sig := strings.TrimPrefix(signature, "sha256=")
	expected := computeHMAC(payload, s.secret)

	return hmac.Equal([]byte(sig), []byte(expected))
}

// computeHMAC generates the HMAC-SHA256 signature.
func computeHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
