package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/amir20/dozzle/types"
)

const defaultTelegramTemplate = `<b>{{ .Subscription.Name }}</b>
<b>Container:</b> {{ .Container.Name }}
<b>Host:</b> {{ .Container.HostName }}
<b>Image:</b> {{ .Container.Image }}

{{ .Detail }}`

type TelegramDispatcher struct {
	Name            string
	BotToken        string
	ChatID          string
	MessageThreadID string
	ParseMode       string
	TemplateText    string
	apiURL          string
	client          *http.Client
}

func NewTelegramDispatcher(name, botToken, chatID, messageThreadID, parseMode, templateText string) (*TelegramDispatcher, error) {
	if strings.TrimSpace(botToken) == "" {
		return nil, fmt.Errorf("bot token is required for telegram dispatcher")
	}
	if strings.TrimSpace(chatID) == "" {
		return nil, fmt.Errorf("chat id is required for telegram dispatcher")
	}
	if strings.TrimSpace(parseMode) == "" {
		parseMode = "HTML"
	}
	if strings.TrimSpace(templateText) == "" {
		templateText = defaultTelegramTemplate
	}

	return &TelegramDispatcher{
		Name:            name,
		BotToken:        botToken,
		ChatID:          chatID,
		MessageThreadID: messageThreadID,
		ParseMode:       parseMode,
		TemplateText:    templateText,
		apiURL:          fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (t *TelegramDispatcher) WithTemplate(templateText string) (Dispatcher, error) {
	return NewTelegramDispatcher(t.Name, t.BotToken, t.ChatID, t.MessageThreadID, t.ParseMode, templateText)
}

func (t *TelegramDispatcher) Send(ctx context.Context, notification types.Notification) error {
	body, err := t.buildPayload(notification)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limitedReader := io.LimitReader(resp.Body, 1024*1024)
		responseBody, _ := io.ReadAll(limitedReader)
		return fmt.Errorf("telegram returned status code %d: %s", resp.StatusCode, string(responseBody))
	}

	return nil
}

func (t *TelegramDispatcher) buildPayload(notification types.Notification) ([]byte, error) {
	text, err := executeTextTemplate(t.TemplateText, notification)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"chat_id":                  t.ChatID,
		"text":                     text,
		"disable_web_page_preview": true,
	}
	if strings.TrimSpace(t.ParseMode) != "" && !strings.EqualFold(t.ParseMode, "none") {
		payload["parse_mode"] = t.ParseMode
	}
	if strings.TrimSpace(t.MessageThreadID) != "" {
		if value, err := strconv.Atoi(strings.TrimSpace(t.MessageThreadID)); err == nil {
			payload["message_thread_id"] = value
		} else {
			payload["message_thread_id"] = t.MessageThreadID
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal telegram payload: %w", err)
	}
	return body, nil
}
