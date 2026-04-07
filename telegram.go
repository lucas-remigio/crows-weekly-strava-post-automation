package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

const telegramAPIBase = "https://api.telegram.org/bot%s/sendMessage"

func sendTelegramMessage(cfg Config, message string) {
	if len(cfg.TelegramChatIDs) == 0 {
		slog.Warn("No TELEGRAM_CHAT_IDS configured — skipping Telegram.")
		return
	}

	client := httpClient(cfg.HTTPTimeoutSeconds)

	for _, chatID := range cfg.TelegramChatIDs {
		if err := sendToOne(client, cfg.TelegramBotToken, chatID, message); err != nil {
			slog.Error("Failed to send Telegram message", "chat_id", chatID, "error", err)
		}
	}
}

func sendToOne(client *http.Client, botToken, chatID, message string) error {
	apiURL := fmt.Sprintf(telegramAPIBase, botToken)

	payload, err := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    message,
	})
	if err != nil {
		return err
	}

	slog.Info("Sending Telegram message", "chat_id", chatID)
	resp, err := client.Post(apiURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	slog.Info("Telegram response", "chat_id", chatID, "status", resp.Status)
	return checkStatus(resp)
}
