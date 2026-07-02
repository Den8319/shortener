package audit 

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	 
)

type MockObserver struct {
	mu     sync.Mutex
	events []AuditEvent
	err    error
}

func (m *MockObserver) Notify(event AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, event)
	return nil
}

func (m *MockObserver) GetEvents() []AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.events
}

// Аудитор отправляет события всем наблюдателям
func TestAuditor_Notify_DispatchesToAllObservers(t *testing.T) {
	auditor := NewAuditor()
	mock1 := &MockObserver{}
	mock2 := &MockObserver{}

	auditor.Register(mock1)
	auditor.Register(mock2)

	event := AuditEvent{
		Ts:     time.Now().Unix(),
		Action: "shorten",
		UserID: "user_123",
		URL:    "https://example.com",
	}

	auditor.Notify(event)

	// Проверяем, что ВСЕ зарегистрированные наблюдатели получили событие
	assert.Len(t, mock1.GetEvents(), 1)
	assert.Len(t, mock2.GetEvents(), 1)
	assert.Equal(t, event, mock1.GetEvents()[0])
}

 
// Тест для HTTPObserver
func TestHTTPObserver_Notify_TableDriven(t *testing.T) {
	sampleEvent := AuditEvent{
		Ts:     1700000000,
		Action: "follow",
		UserID: "user_789",
		URL:    "https://github.com",
	}

	tests := []struct {
		name           string
		statusCode     int
		handlerAsserts func(t *testing.T, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:       "Успешная отправка 200 OK",
			statusCode: http.StatusOK,
			handlerAsserts: func(t *testing.T, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				
				var received AuditEvent
				err = json.Unmarshal(body, &received)
				require.NoError(t, err)
				assert.Equal(t, sampleEvent, received)
			},
			wantErr: false,
		},
		{
			name:        "Ошибка клиента 400 Bad Request",
			statusCode:  http.StatusBadRequest,
			wantErr:     true,
			errContains: "http observer got unexpected status: 400",
		},
		{
			name:        "Ошибка сервера 500 Internal Error",
			statusCode:  http.StatusInternalServerError,
			wantErr:     true,
			errContains: "http observer got unexpected status: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.handlerAsserts != nil {
					tt.handlerAsserts(t, r)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			ho := NewHTTPObserver(server.URL)
			err := ho.Notify(sampleEvent)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

 
func TestFileObserver_Notify_WritesValidJSONLines(t *testing.T) {
 
	tmpFile, err := os.CreateTemp("", "audit_test_*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())  
	tmpFile.Close()

	fo, err := NewFileObserver(tmpFile.Name())
	require.NoError(t, err)
	defer fo.Close()

	event1 := AuditEvent{Ts: 111, Action: "GetShort", UserID: "u1"}
	event2 := AuditEvent{Ts: 222, Action: "GetLong", UserID: "u2"}

	 
	require.NoError(t, fo.Notify(event1))
	require.NoError(t, fo.Notify(event2))

	 
	fileBytes, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
 
	
	var decoded1, decoded2 AuditEvent
	
	 
	err = json.Unmarshal(fileBytes, &decoded1)  
	f, err := os.Open(tmpFile.Name())
	require.NoError(t, err)
	defer f.Close()

	dec := json.NewDecoder(f)
	
	err = dec.Decode(&decoded1)
	require.NoError(t, err)
	assert.Equal(t, event1, decoded1)

	err = dec.Decode(&decoded2)
	require.NoError(t, err)
	assert.Equal(t, event2, decoded2)
}
 