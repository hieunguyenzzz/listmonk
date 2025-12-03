package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// Event type constants.
const (
	EventSubscriberCreated      = "subscriber.created"
	EventSubscriberUpdated      = "subscriber.updated"
	EventSubscriberUnsubscribed = "subscriber.unsubscribed"
	EventSubscriberConfirmed    = "subscriber.confirmed"
	EventSubscriberBlocklisted  = "subscriber.blocklisted"
	EventSubscriberDeleted      = "subscriber.deleted"

	EventCampaignStarted  = "campaign.started"
	EventCampaignFinished = "campaign.finished"

	EventLinkClick = "tracking.link_click"
	EventEmailOpen = "tracking.email_open"
	EventBounce    = "tracking.bounce"
)

// AllEvents is a list of all supported event types.
var AllEvents = []string{
	EventSubscriberCreated,
	EventSubscriberUpdated,
	EventSubscriberUnsubscribed,
	EventSubscriberConfirmed,
	EventSubscriberBlocklisted,
	EventSubscriberDeleted,
	EventCampaignStarted,
	EventCampaignFinished,
	EventLinkClick,
	EventEmailOpen,
	EventBounce,
}

// Payload represents a webhook payload envelope.
type Payload struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// Opt represents configuration for a single webhook endpoint.
type Opt struct {
	UUID     string
	Name     string
	URL      string
	Secret   string
	Events   []string
	MaxConns int
	Timeout  time.Duration
}

// endpoint represents a configured webhook endpoint.
type endpoint struct {
	uuid   string
	name   string
	url    string
	secret string
	events map[string]bool
	client *http.Client
}

// dispatchJob represents a job to dispatch a webhook.
type dispatchJob struct {
	ep      *endpoint
	payload []byte
}

// Manager handles webhook dispatch.
type Manager struct {
	endpoints []*endpoint
	log       *log.Logger
	ch        chan dispatchJob
	closeCh   chan struct{}
}

// New creates a new webhook Manager.
func New(opts []Opt, lo *log.Logger) (*Manager, error) {
	m := &Manager{
		endpoints: make([]*endpoint, 0, len(opts)),
		log:       lo,
		ch:        make(chan dispatchJob, 1000),
		closeCh:   make(chan struct{}),
	}

	for _, o := range opts {
		events := make(map[string]bool, len(o.Events))
		for _, e := range o.Events {
			events[e] = true
		}

		timeout := o.Timeout
		if timeout == 0 {
			timeout = 5 * time.Second
		}

		maxConns := o.MaxConns
		if maxConns == 0 {
			maxConns = 5
		}

		ep := &endpoint{
			uuid:   o.UUID,
			name:   o.Name,
			url:    o.URL,
			secret: o.Secret,
			events: events,
			client: &http.Client{
				Timeout: timeout,
				Transport: &http.Transport{
					MaxIdleConnsPerHost:   maxConns,
					MaxConnsPerHost:       maxConns,
					ResponseHeaderTimeout: timeout,
					IdleConnTimeout:       timeout,
				},
			},
		}
		m.endpoints = append(m.endpoints, ep)
	}

	// Start worker goroutine.
	go m.worker()

	return m, nil
}

// worker processes dispatch jobs from the channel.
func (m *Manager) worker() {
	for {
		select {
		case job := <-m.ch:
			m.send(job.ep, job.payload)
		case <-m.closeCh:
			return
		}
	}
}

// Dispatch sends an event to all subscribed endpoints.
func (m *Manager) Dispatch(event string, data any) {
	if len(m.endpoints) == 0 {
		return
	}

	payload := Payload{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		m.log.Printf("webhook: error marshalling payload: %v", err)
		return
	}

	for _, ep := range m.endpoints {
		if ep.events[event] {
			select {
			case m.ch <- dispatchJob{ep: ep, payload: b}:
			default:
				m.log.Printf("webhook: dispatch queue full, dropping event %s for %s", event, ep.name)
			}
		}
	}
}

// send performs the actual HTTP POST to the endpoint.
func (m *Manager) send(ep *endpoint, payload []byte) {
	req, err := http.NewRequest(http.MethodPost, ep.url, bytes.NewReader(payload))
	if err != nil {
		m.log.Printf("webhook: error creating request for %s: %v", ep.name, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "listmonk")

	// Add HMAC signature if secret is configured.
	if ep.secret != "" {
		sig := computeHMAC(payload, ep.secret)
		req.Header.Set("X-Webhook-Signature", "sha256="+sig)
	}

	resp, err := ep.client.Do(req)
	if err != nil {
		m.log.Printf("webhook: error sending to %s: %v", ep.name, err)
		return
	}
	defer func() {
		// Drain and close the body to let the Transport reuse the connection.
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		m.log.Printf("webhook: non-OK response from %s: %d", ep.name, resp.StatusCode)
	}
}

// computeHMAC generates the HMAC-SHA256 signature.
func computeHMAC(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// Close shuts down the webhook manager.
func (m *Manager) Close() {
	close(m.closeCh)
}

// HasEndpoints returns true if there are any configured endpoints.
func (m *Manager) HasEndpoints() bool {
	return len(m.endpoints) > 0
}
