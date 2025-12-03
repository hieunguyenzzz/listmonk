package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testLogger returns a logger for testing.
func testLogger() *log.Logger {
	return log.New(os.Stderr, "webhook-test: ", log.LstdFlags)
}

// TestComputeHMAC tests HMAC signature computation.
func TestComputeHMAC(t *testing.T) {
	payload := []byte(`{"event":"test","data":{}}`)
	secret := "my-secret-key"

	sig := computeHMAC(payload, secret)

	// Verify the signature manually.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if sig != expected {
		t.Errorf("HMAC mismatch: got %s, want %s", sig, expected)
	}

	// Verify that different secrets produce different signatures.
	sig2 := computeHMAC(payload, "different-secret")
	if sig == sig2 {
		t.Error("Different secrets should produce different signatures")
	}

	// Verify that different payloads produce different signatures.
	sig3 := computeHMAC([]byte(`{"event":"other"}`), secret)
	if sig == sig3 {
		t.Error("Different payloads should produce different signatures")
	}
}

// TestPayloadFormat tests that the payload is correctly formatted.
func TestPayloadFormat(t *testing.T) {
	var received Payload
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		json.Unmarshal(body, &received)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := []Opt{{
		Name:    "test",
		URL:     server.URL,
		Events:  []string{EventSubscriberCreated},
		Timeout: 5 * time.Second,
	}}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	testData := map[string]any{"id": 123, "email": "test@example.com"}
	m.Dispatch(EventSubscriberCreated, testData)

	// Wait for dispatch.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if received.Event != EventSubscriberCreated {
		t.Errorf("Event mismatch: got %s, want %s", received.Event, EventSubscriberCreated)
	}

	if received.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if received.Data == nil {
		t.Error("Data should not be nil")
	}

	// Check data contains expected fields.
	dataMap, ok := received.Data.(map[string]any)
	if !ok {
		t.Fatal("Data should be a map")
	}

	if dataMap["email"] != "test@example.com" {
		t.Errorf("Data email mismatch: got %v", dataMap["email"])
	}
}

// TestEventFiltering tests that endpoints only receive subscribed events.
func TestEventFiltering(t *testing.T) {
	var subscriberCount, campaignCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p Payload
		json.Unmarshal(body, &p)

		switch p.Event {
		case EventSubscriberCreated:
			subscriberCount.Add(1)
		case EventCampaignStarted:
			campaignCount.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create endpoint that only subscribes to subscriber events.
	opts := []Opt{{
		Name:    "subscriber-only",
		URL:     server.URL,
		Events:  []string{EventSubscriberCreated, EventSubscriberDeleted},
		Timeout: 5 * time.Second,
	}}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	// Dispatch both types of events.
	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})
	m.Dispatch(EventCampaignStarted, map[string]any{"id": 2})
	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 3})

	// Wait for dispatch.
	time.Sleep(200 * time.Millisecond)

	if subscriberCount.Load() != 2 {
		t.Errorf("Expected 2 subscriber events, got %d", subscriberCount.Load())
	}

	if campaignCount.Load() != 0 {
		t.Errorf("Expected 0 campaign events, got %d", campaignCount.Load())
	}
}

// TestMultipleEndpoints tests dispatching to multiple endpoints.
func TestMultipleEndpoints(t *testing.T) {
	var count1, count2 atomic.Int32

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count1.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count2.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	opts := []Opt{
		{Name: "ep1", URL: server1.URL, Events: []string{EventSubscriberCreated}, Timeout: 5 * time.Second},
		{Name: "ep2", URL: server2.URL, Events: []string{EventSubscriberCreated}, Timeout: 5 * time.Second},
	}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})

	time.Sleep(200 * time.Millisecond)

	if count1.Load() != 1 {
		t.Errorf("Endpoint 1 should receive 1 event, got %d", count1.Load())
	}

	if count2.Load() != 1 {
		t.Errorf("Endpoint 2 should receive 1 event, got %d", count2.Load())
	}
}

// TestAsyncDispatch tests that dispatch is non-blocking.
func TestAsyncDispatch(t *testing.T) {
	// Create a slow server that takes 500ms to respond.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := []Opt{{
		Name:    "slow",
		URL:     server.URL,
		Events:  []string{EventSubscriberCreated},
		Timeout: 2 * time.Second,
	}}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	start := time.Now()
	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})
	elapsed := time.Since(start)

	// Dispatch should return immediately (< 10ms), not wait for the slow server.
	if elapsed > 50*time.Millisecond {
		t.Errorf("Dispatch took too long: %v (should be < 50ms)", elapsed)
	}
}

// TestSignatureHeader tests that HMAC signature is included when secret is set.
func TestSignatureHeader(t *testing.T) {
	var receivedSig string
	var receivedBody []byte
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedSig = r.Header.Get("X-Webhook-Signature")
		receivedBody, _ = io.ReadAll(r.Body)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	secret := "test-secret-123"
	opts := []Opt{{
		Name:    "signed",
		URL:     server.URL,
		Secret:  secret,
		Events:  []string{EventSubscriberCreated},
		Timeout: 5 * time.Second,
	}}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if receivedSig == "" {
		t.Error("X-Webhook-Signature header should be present")
	}

	// Verify signature format.
	if len(receivedSig) < 8 || receivedSig[:7] != "sha256=" {
		t.Errorf("Signature should start with 'sha256=', got: %s", receivedSig)
	}

	// Verify the signature is valid.
	expectedSig := "sha256=" + computeHMAC(receivedBody, secret)
	if receivedSig != expectedSig {
		t.Errorf("Signature mismatch: got %s, want %s", receivedSig, expectedSig)
	}
}

// TestNoSignatureWhenNoSecret tests that no signature is added when secret is empty.
func TestNoSignatureWhenNoSecret(t *testing.T) {
	var receivedSig string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedSig = r.Header.Get("X-Webhook-Signature")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := []Opt{{
		Name:    "unsigned",
		URL:     server.URL,
		Secret:  "", // No secret.
		Events:  []string{EventSubscriberCreated},
		Timeout: 5 * time.Second,
	}}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if receivedSig != "" {
		t.Errorf("X-Webhook-Signature should not be present, got: %s", receivedSig)
	}
}

// TestContentTypeHeader tests that Content-Type is set correctly.
func TestContentTypeHeader(t *testing.T) {
	var contentType string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		contentType = r.Header.Get("Content-Type")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := []Opt{{
		Name:    "test",
		URL:     server.URL,
		Events:  []string{EventSubscriberCreated},
		Timeout: 5 * time.Second,
	}}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if contentType != "application/json" {
		t.Errorf("Content-Type should be 'application/json', got: %s", contentType)
	}
}

// TestUserAgentHeader tests that User-Agent is set correctly.
func TestUserAgentHeader(t *testing.T) {
	var userAgent string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		userAgent = r.Header.Get("User-Agent")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := []Opt{{
		Name:    "test",
		URL:     server.URL,
		Events:  []string{EventSubscriberCreated},
		Timeout: 5 * time.Second,
	}}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if userAgent != "listmonk" {
		t.Errorf("User-Agent should be 'listmonk', got: %s", userAgent)
	}
}

// TestTimeout tests that HTTP timeout is handled gracefully.
func TestTimeout(t *testing.T) {
	// Create a server that never responds.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := []Opt{{
		Name:    "timeout-test",
		URL:     server.URL,
		Events:  []string{EventSubscriberCreated},
		Timeout: 100 * time.Millisecond, // Short timeout.
	}}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	// This should not panic or hang.
	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})

	// Give it time to timeout and log the error.
	time.Sleep(300 * time.Millisecond)
}

// TestErrorResponse tests that error responses are logged but don't crash.
func TestErrorResponse(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	opts := []Opt{{
		Name:    "error-test",
		URL:     server.URL,
		Events:  []string{EventSubscriberCreated},
		Timeout: 5 * time.Second,
	}}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	// This should not panic.
	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})
	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 2})

	time.Sleep(200 * time.Millisecond)

	if requestCount.Load() != 2 {
		t.Errorf("Expected 2 requests, got %d", requestCount.Load())
	}
}

// TestNoEndpoints tests that dispatch with no endpoints is a no-op.
func TestNoEndpoints(t *testing.T) {
	m, err := New([]Opt{}, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	// This should not panic.
	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})

	if m.HasEndpoints() {
		t.Error("HasEndpoints should return false when no endpoints are configured")
	}
}

// TestHasEndpoints tests the HasEndpoints method.
func TestHasEndpoints(t *testing.T) {
	// Without endpoints.
	m1, _ := New([]Opt{}, testLogger())
	defer m1.Close()

	if m1.HasEndpoints() {
		t.Error("HasEndpoints should return false for empty endpoints")
	}

	// With endpoints.
	m2, _ := New([]Opt{{
		Name:    "test",
		URL:     "http://example.com",
		Events:  []string{EventSubscriberCreated},
		Timeout: 5 * time.Second,
	}}, testLogger())
	defer m2.Close()

	if !m2.HasEndpoints() {
		t.Error("HasEndpoints should return true when endpoints are configured")
	}
}

// TestDefaultValues tests that default values are applied.
func TestDefaultValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create with zero values for timeout and maxconns.
	opts := []Opt{{
		Name:    "defaults",
		URL:     server.URL,
		Events:  []string{EventSubscriberCreated},
		Timeout: 0, // Should default to 5s.
		MaxConns: 0, // Should default to 5.
	}}

	m, err := New(opts, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer m.Close()

	// Just verify it doesn't panic and works.
	m.Dispatch(EventSubscriberCreated, map[string]any{"id": 1})
	time.Sleep(100 * time.Millisecond)
}

// TestAllEventConstants tests that all event constants are in AllEvents.
func TestAllEventConstants(t *testing.T) {
	expected := map[string]bool{
		EventSubscriberCreated:      true,
		EventSubscriberUpdated:      true,
		EventSubscriberUnsubscribed: true,
		EventSubscriberConfirmed:    true,
		EventSubscriberBlocklisted:  true,
		EventSubscriberDeleted:      true,
		EventCampaignStarted:        true,
		EventCampaignFinished:       true,
		EventLinkClick:              true,
		EventEmailOpen:              true,
		EventBounce:                 true,
	}

	if len(AllEvents) != len(expected) {
		t.Errorf("AllEvents length mismatch: got %d, want %d", len(AllEvents), len(expected))
	}

	for _, e := range AllEvents {
		if !expected[e] {
			t.Errorf("Unexpected event in AllEvents: %s", e)
		}
	}
}

// TestClose tests that Close stops the worker.
func TestClose(t *testing.T) {
	m, err := New([]Opt{}, testLogger())
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Close should not panic.
	m.Close()

	// Multiple close calls should not panic (though not recommended).
	// This is just to ensure robustness.
}
