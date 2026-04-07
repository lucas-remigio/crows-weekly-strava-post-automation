package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	lastResumoTime = make(map[int64]time.Time)
	resumoMutex    sync.Mutex
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

func handleUnknownCommand(cfg Config, chatID int64, client *http.Client) {
	chatStr := strconv.FormatInt(chatID, 10)
	_ = sendToOne(client, cfg.TelegramBotToken, chatStr, "❓ Comando inválido.\n\nComandos disponíveis:\n/resumo - Gera um relatório de teste parcial da semana atual")
}

// tryConsumeRateLimit implements a thread-safe rate lock logic.
// Returns `allowed` boolean and the remaining minutes to wait if rejected.
func tryConsumeRateLimit(chatID int64) (allowed bool, waitMins int) {
	resumoMutex.Lock()
	defer resumoMutex.Unlock()

	if lastUsed, exists := lastResumoTime[chatID]; exists {
		if remaining := 30*time.Minute - time.Since(lastUsed); remaining > 0 {
			mins := int(remaining.Minutes())
			if mins < 1 {
				mins = 1
			}
			return false, mins
		}
	}

	lastResumoTime[chatID] = time.Now()
	return true, 0
}

func handlePreviewCommand(cfg Config, chatID int64, client *http.Client) {
	chatStr := strconv.FormatInt(chatID, 10)

	allowed, waitMins := tryConsumeRateLimit(chatID)
	if !allowed {
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, fmt.Sprintf("⏳ Por favor, aguarde %d minuto(s) antes de pedir um novo resumo para evitar spam nas APIs.", waitMins))
		return
	}

	_ = sendToOne(client, cfg.TelegramBotToken, chatStr, "⏳ A gerar o resumo da semana em tempo real (isto pode demorar uns segundos)...")

	postText, err := generateDryRunPost(cfg)
	if err != nil {
		slog.Error("Dry run failed for /resumo", "error", err)
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, "❌ Oops! Ocorreu um erro a gerar o relatório: "+err.Error())
	} else {
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, postText)
	}
}

func generateDryRunPost(cfg Config) (string, error) {
	if err := cfg.validate(); err != nil {
		return "", err
	}

	postText, _, _, _, _, err := buildWeeklyPost(cfg, 0, time.Now(), false)
	return postText, err
}
