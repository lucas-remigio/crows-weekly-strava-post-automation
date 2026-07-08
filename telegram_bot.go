package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type telegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

type getUpdatesResponse struct {
	Ok     bool             `json:"ok"`
	Result []telegramUpdate `json:"result"`
}

func pollTelegramCommands(cfg Config) {
	if cfg.TelegramBotToken == "" {
		slog.Warn("No TELEGRAM_BOT_TOKEN configured, ignoring incoming commands.")
		return
	}

	client := &http.Client{Timeout: 65 * time.Second} // Slightly longer than long-polling timeout
	offset := 0

	slog.Info("Listening for Telegram commands (/resumo)...")

	for {
		updates, err := fetchUpdates(client, cfg.TelegramBotToken, offset)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			offset = update.UpdateID + 1
			text := strings.TrimSpace(update.Message.Text)

			if !strings.HasPrefix(text, "/") {
				continue
			}

			if strings.HasPrefix(text, "/resumo") {
				slog.Info("Received /resumo command", "chat_id", update.Message.Chat.ID)
				go handlePreviewCommand(cfg, update.Message.Chat.ID, client)
			} else if strings.HasPrefix(text, "/wook") {
				slog.Info("Received /wook command", "chat_id", update.Message.Chat.ID)
				go handleWookCommand(cfg, update.Message.Chat.ID, client)
			} else if strings.HasPrefix(text, "/fnac") {
				slog.Info("Received /fnac command", "chat_id", update.Message.Chat.ID)
				go handleFnacCommand(cfg, update.Message.Chat.ID, client)
			} else if strings.HasPrefix(text, "/livrarias") {
				slog.Info("Received /livrarias command", "chat_id", update.Message.Chat.ID)
				go handleLibrariesCommand(cfg, update.Message.Chat.ID, client)
			} else {
				slog.Info("Received unknown command", "chat_id", update.Message.Chat.ID, "command", text)
				go handleUnknownCommand(cfg, update.Message.Chat.ID, client)
			}
		}
	}
}

func fetchUpdates(client *http.Client, token string, offset int) ([]telegramUpdate, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=60", token, offset)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updateResp getUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil || !updateResp.Ok {
		return nil, fmt.Errorf("failed to decode updates")
	}
	return updateResp.Result, nil
}
