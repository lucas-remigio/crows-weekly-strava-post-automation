package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	history map[int64][]time.Time
	limit   int
	window  time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		history: make(map[int64][]time.Time),
		limit:   limit,
		window:  window,
	}
}

// consume attempts to take n tokens. Returns allowed=true if successful.
func (rl *rateLimiter) consume(chatID int64, tokens int) (allowed bool, waitMins int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	var valid []time.Time

	for _, t := range rl.history[chatID] {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}
	rl.history[chatID] = valid

	if len(valid)+tokens > rl.limit {
		idx := len(valid) + tokens - rl.limit - 1
		if idx < 0 || idx >= len(valid) {
			return false, int(rl.window.Minutes())
		}
		
		neededToExpire := valid[idx]
		waitDur := rl.window - now.Sub(neededToExpire)
		mins := int(waitDur.Minutes())
		if mins < 1 {
			mins = 1
		}
		return false, mins
	}

	for i := 0; i < tokens; i++ {
		rl.history[chatID] = append(rl.history[chatID], now)
	}
	return true, 0
}

var (
	resumoLimiter  = newRateLimiter(3, time.Hour)
	libraryLimiter = newRateLimiter(3, time.Hour)
)

func handleUnknownCommand(cfg Config, chatID int64, client *http.Client) {
	chatStr := strconv.FormatInt(chatID, 10)
	_ = sendToOne(client, cfg.TelegramBotToken, chatStr, "❓ Comando inválido.\n\nComandos disponíveis:\n/resumo - Gera um relatório de teste parcial da semana atual\n/livrarias - Verifica promoções na Wook e Fnac\n/wook - Verifica a promoção atual na Wook\n/fnac - Verifica a promoção atual na Fnac")
}

func handlePreviewCommand(cfg Config, chatID int64, client *http.Client) {
	chatStr := strconv.FormatInt(chatID, 10)

	allowed, waitMins := resumoLimiter.consume(chatID, 1)
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

func handleWookCommand(cfg Config, chatID int64, client *http.Client) {
	chatStr := strconv.FormatInt(chatID, 10)

	// A bit of spam protection reusing the same mutex but maybe it's fine
	allowed, waitMins := libraryLimiter.consume(chatID, 1)
	if !allowed {
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, fmt.Sprintf("⏳ Por favor, aguarde %d minuto(s) antes de pedir uma nova verificação.", waitMins))
		return
	}

	doWookCheck(cfg, chatStr, client)
}

func doWookCheck(cfg Config, chatStr string, client *http.Client) {
	_ = sendToOne(client, cfg.TelegramBotToken, chatStr, "⏳ A verificar a Wook em tempo real (isto pode demorar uns segundos)...")

	msg, err := getWookPromoMessage(cfg)
	if err != nil {
		slog.Error("Failed to fetch wook promo for /wook", "error", err)
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, "❌ Oops! Ocorreu um erro ao ler a página: "+err.Error())
		return
	}

	if msg == "" {
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, "Nenhuma promoção clara encontrada neste momento.")
	} else {
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, msg)
	}
}

func handleFnacCommand(cfg Config, chatID int64, client *http.Client) {
	chatStr := strconv.FormatInt(chatID, 10)

	allowed, waitMins := libraryLimiter.consume(chatID, 1)
	if !allowed {
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, fmt.Sprintf("⚠️ Demasiados pedidos. Por favor, aguarda %d minuto(s).", waitMins))
		return
	}

	doFnacCheck(cfg, chatStr, client)
}

func doFnacCheck(cfg Config, chatStr string, client *http.Client) {
	_ = sendToOne(client, cfg.TelegramBotToken, chatStr, "⏳ A verificar a Fnac em tempo real (isto pode demorar uns segundos)...")

	msg, err := getFnacPromoMessage(cfg)
	if err != nil {
		slog.Error("Failed to fetch fnac promo for /fnac", "error", err)
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, "❌ Oops! Ocorreu um erro ao ler a página: "+err.Error())
		return
	}

	if msg == "" {
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, "Nenhuma promoção clara encontrada neste momento.")
	} else {
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, msg)
	}
}

func handleLibrariesCommand(cfg Config, chatID int64, client *http.Client) {
	chatStr := strconv.FormatInt(chatID, 10)

	allowed, waitMins := libraryLimiter.consume(chatID, 1)
	if !allowed {
		_ = sendToOne(client, cfg.TelegramBotToken, chatStr, fmt.Sprintf("⏳ Por favor, aguarde %d minuto(s) antes de pedir uma nova verificação.", waitMins))
		return
	}

	// Let Wook and Fnac handles run concurrently
	go doWookCheck(cfg, chatStr, client)
	go doFnacCheck(cfg, chatStr, client)
}
