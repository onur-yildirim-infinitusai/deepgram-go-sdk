package commonv1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dvonthenen/websocket"

	commonv1interfaces "github.com/deepgram/deepgram-go-sdk/pkg/client/common/v1/interfaces"
	clientinterfaces "github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces"
)

func TestWriteDeadline(t *testing.T) {
    // Create a test server that sleeps longer than the deadline
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        upgrader := websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool {
                return true
            },
        }
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            t.Fatalf("Failed to upgrade connection: %v", err)
            return
        }
        defer conn.Close()

        // Read the message but delay the response
        _, _, err = conn.ReadMessage()
        if err != nil {
            t.Logf("Read error (expected due to timeout): %v", err)
            return
        }
        time.Sleep(2 * time.Second)
    }))
    defer server.Close()

	// Convert http://... to ws://...
	url := strings.Replace(server.URL, "http://", "ws://", 1)

	// Create client with a very short write deadline
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	options := &clientinterfaces.ClientOptions{
		Host:          url,
		WriteDeadline: 100 * time.Millisecond, // Very short deadline
	}

	var handler commonv1interfaces.WebSocketHandler = &MockWebSocketHandler{}
    var router commonv1interfaces.Router = &MockRouter{}

	client := NewWS(ctx, cancel, "api-key", options, &handler, &router)
	if !client.Connect() {
		t.Fatal("Failed to connect")
	}

	// Attempt to write a large payload
	largePayload := make([]byte, 1024*1024) // 1MB of data
	err := client.WriteBinary(largePayload)

	// Verify that we got a timeout error
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "deadline exceeded") {
		t.Errorf("Expected deadline exceeded error, got: %v", err)
	}
}

type MockWebSocketHandler struct{}

func (m *MockWebSocketHandler) GetURL(host string) (string, error)                { return host, nil }
func (m *MockWebSocketHandler) Start()                                           {}
func (m *MockWebSocketHandler) Finish()                                          {}
func (m *MockWebSocketHandler) ProcessMessage(msgType int, msg []byte) error     { return nil }
func (m *MockWebSocketHandler) ProcessError(err error) error                     { return nil }
func (m *MockWebSocketHandler) GetCloseMsg() []byte                             { return nil }

type MockRouter struct{}

func (m *MockRouter) Open(resp *commonv1interfaces.OpenResponse) error   { return nil }
func (m *MockRouter) Close(resp *commonv1interfaces.CloseResponse) error { return nil }
func (m *MockRouter) Binary(msg []byte) error                              { return nil }
func (m *MockRouter) Error(err *clientinterfaces.DeepgramError) error       { return nil }
func (m *MockRouter) Message(msg []byte) error                  { return nil }