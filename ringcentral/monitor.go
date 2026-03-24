package ringcentral

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	maxConsecutiveFailures = 5
	initialBackoff         = 3 * time.Second
	maxBackoff             = 60 * time.Second
	wsPingInterval         = 30 * time.Second
)

// MessageHandler is called for each received post.
type MessageHandler func(ctx context.Context, client *Client, post Post)

// Monitor manages the WebSocket connection for receiving messages.
type Monitor struct {
	client   *Client
	handler  MessageHandler
	failures int
}

// NewMonitor creates a new WebSocket monitor.
func NewMonitor(client *Client, handler MessageHandler) *Monitor {
	return &Monitor{
		client:  client,
		handler: handler,
	}
}

// Run starts the WebSocket event loop. Blocks until ctx is cancelled.
func (m *Monitor) Run(ctx context.Context) error {
	log.Println("[monitor] starting WebSocket event loop")

	for {
		select {
		case <-ctx.Done():
			log.Println("[monitor] shutting down")
			return ctx.Err()
		default:
		}

		err := m.connectAndListen(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		m.failures++
		backoff := m.calcBackoff()
		log.Printf("[monitor] WebSocket disconnected (%d/%d, backoff=%s): %v",
			m.failures, maxConsecutiveFailures, backoff, err)

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *Monitor) connectAndListen(ctx context.Context) error {
	// Get WebSocket token
	wsToken, err := m.client.Auth().GetWSToken()
	if err != nil {
		return fmt.Errorf("get WS token: %w", err)
	}

	// Connect to WebSocket server
	wsURL := wsToken.URI + "?access_token=" + url.QueryEscape(wsToken.WSAccessToken)
	log.Printf("[monitor] connecting to WebSocket: %s", wsToken.URI)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("WebSocket dial: %w", err)
	}
	defer conn.Close()

	// Read ConnectionDetails
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read connection details: %w", err)
	}
	log.Printf("[monitor] connected: %s", string(msg))

	// Subscribe to team messaging post events
	if err := m.subscribe(conn); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	// Reset failure counter on successful connection
	m.failures = 0
	log.Println("[monitor] subscribed to post events, listening...")

	// Start ping ticker
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	// Read events
	for {
		select {
		case <-ctx.Done():
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return ctx.Err()
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return fmt.Errorf("ping: %w", err)
			}
		default:
		}

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return nil
			}
			return fmt.Errorf("read message: %w", err)
		}

		m.handleWSMessage(ctx, msg)
	}
}

func (m *Monitor) subscribe(conn *websocket.Conn) error {
	subReq := []interface{}{
		WSClientRequest{
			Type:      "ClientRequest",
			MessageID: uuid.New().String(),
			Method:    "POST",
			Path:      "/restapi/v1.0/subscription/",
		},
		WSSubscriptionBody{
			EventFilters: []string{
				"/team-messaging/v1/posts",
			},
			DeliveryMode: WSDeliveryMode{
				TransportType: "WebSocket",
			},
		},
	}

	data, err := json.Marshal(subReq)
	if err != nil {
		return fmt.Errorf("marshal subscription: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("send subscription: %w", err)
	}

	// Read subscription response
	_, resp, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read subscription response: %w", err)
	}
	log.Printf("[monitor] subscription response: %s", string(resp))
	return nil
}

func (m *Monitor) handleWSMessage(ctx context.Context, msg []byte) {
	var event WSEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		// Not all messages are events (heartbeats, etc.)
		return
	}

	// Only process PostAdded events
	if event.Body.EventType != "PostAdded" {
		return
	}

	// Skip messages from self
	if event.Body.CreatorID == m.client.OwnerID() {
		return
	}

	// Only process text messages
	if event.Body.Type != "TextMessage" {
		return
	}

	// Filter by chat ID if configured
	chatID := m.client.ChatID()
	if chatID != "" && event.Body.GroupID != chatID {
		return
	}

	log.Printf("[monitor] received post from %s in chat %s: %q",
		event.Body.CreatorID, event.Body.GroupID, truncate(event.Body.Text, 50))

	go m.handler(ctx, m.client, event.Body)
}

func (m *Monitor) calcBackoff() time.Duration {
	d := initialBackoff
	for i := 1; i < m.failures; i++ {
		d *= 2
		if d > maxBackoff {
			return maxBackoff
		}
	}
	return d
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
