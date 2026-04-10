package dispatcher

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/amir20/dozzle/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramDispatcherSend(t *testing.T) {
	dispatcher, err := NewTelegramDispatcher("telegram", "token", "-1001", "42", "HTML", "", "", "", "", "", "<b>{{ .Container.Name }}</b>\n{{ .Detail }}")
	require.NoError(t, err)

	notification := types.Notification{
		ID:           "n1",
		Type:         types.LogNotification,
		Detail:       "hello",
		Timestamp:    time.Now(),
		Container:    types.NotificationContainer{Name: "api"},
		Subscription: types.SubscriptionConfig{Name: "Alert"},
	}

	body, err := dispatcher.buildPayload(notification)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, "-1001", payload["chat_id"])
	assert.Equal(t, float64(42), payload["message_thread_id"])
	assert.Equal(t, "HTML", payload["parse_mode"])
	assert.Equal(t, "<b>api</b>\nhello", payload["text"])
}

func TestTelegramDispatcherRejectsMTProtoProxy(t *testing.T) {
	_, err := NewTelegramDispatcher("telegram", "token", "-1001", "", "HTML", "mtproto", "127.0.0.1:443", "", "", "secret", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mtproto proxy is not supported")
}
