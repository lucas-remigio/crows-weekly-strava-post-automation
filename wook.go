package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func runWookDaemon(cfg Config, loc *time.Location) {
	slog.Info("Starting Wook Promo checker daemon mode...")

	for {
		now := time.Now().In(loc)
		target := calculateNextWookRunTime(now, loc)

		slog.Info("Wook Daemon sleeping until next run...", "target", target.Format(time.RFC3339))
		time.Sleep(target.Sub(now))

		slog.Info("Wook Daemon wake up! Checking promotions.")
		checkWookPromo(cfg)

		time.Sleep(1 * time.Minute) // Prevent double-triggering in the same minute
	}
}

func calculateNextWookRunTime(now time.Time, loc *time.Location) time.Time {
	// We want every day at 09:00:00 AM
	target := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)

	if now.After(target) || now.Equal(target) {
		target = target.AddDate(0, 0, 1)
	}
	return target
}

func checkWookPromo(cfg Config) {
	if cfg.WookTelegramChatID == "" {
		slog.Warn("No WOOK_TELEGRAM_CHAT_ID configured, skipping wook check.")
		return
	}

	msg, err := getWookPromoMessage(cfg)
	if err != nil {
		slog.Error("Failed to get wook promo message", "error", err)
		return
	}
	if msg == "" {
		slog.Info("No active wook promo found.")
		return
	}

	// Send to Telegram
	client := httpClient(cfg.HTTPTimeoutSeconds)
	if err := sendToOne(client, cfg.TelegramBotToken, cfg.WookTelegramChatID, msg); err != nil {
		slog.Error("Failed to send Wook Telegram message", "error", err)
	}
}

func getWookPromoMessage(cfg Config) (string, error) {
	html, err := fetchWookHTML(cfg.HTTPTimeoutSeconds)
	if err != nil {
		return "", fmt.Errorf("fetch Wook HTML: %w", err)
	}

	_ = os.WriteFile("wook_debug.html", []byte(html), 0644)
	slog.Info("Fetched Wook HTML", "length", len(html), "file", "wook_debug.html")

	if cfg.OpenAIAPIKey == "" {
		return "", fmt.Errorf("No OPENAI_API_KEY configured")
	}

	// Strip scripts and styles to save tokens and avoid truncation of real content
	reScript := regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	reStyle := regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	cleanHTML := reScript.ReplaceAllString(html, "")
	cleanHTML = reStyle.ReplaceAllString(cleanHTML, "")

	slog.Info("Cleaned HTML (removed scripts/styles)", "old_length", len(html), "new_length", len(cleanHTML))

	// OpenAI has a high token limit for gpt-4o-mini, let's allow up to 150k chars
	limit := 50000
	if len(cleanHTML) > limit {
		slog.Info("Trimmed the content of the html to fit OpenAI limit", "limit", limit)
		cleanHTML = cleanHTML[:limit]
	}

	_ = os.WriteFile("wook_debug_clean.html", []byte(cleanHTML), 0644)
	slog.Info("Ready to call OpenAI", "payload_length", len(cleanHTML), "file", "wook_debug_clean.html")

	promo, err := extractWookPromoWithOpenAI(cfg, cleanHTML)
	if err != nil {
		return "", fmt.Errorf("extract with OpenAI: %w", err)
	}

	if promo == "" || strings.Contains(strings.ToLower(promo), "nenhuma promoção") {
		return "", nil
	}

	slog.Info("Current promo found", "promo", promo)

	return fmt.Sprintf("📚 *WOOK Promoção do Dia*\n\n%s\n\n🔗 https://www.wook.pt", promo), nil
}

func fetchWookHTML(timeoutSeconds int) (string, error) {
	req, err := http.NewRequest("GET", "https://www.wook.pt", nil)
	if err != nil {
		return "", err
	}
	// Use a standard browser user agent and accept headers to bypass basic Cloudflare checks
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-PT,pt;q=0.9,en-US;q=0.8,en;q=0.7")

	client := httpClient(timeoutSeconds)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func extractWookPromoWithOpenAI(cfg Config, html string) (string, error) {
	systemPrompt := "És um assistente útil e focado na extração de dados."
	userPrompt := fmt.Sprintf(
		"Aqui está o código HTML limpo da página inicial da livraria Wook (wook.pt). "+
			"A tua tarefa é extrair TODAS as promoções, campanhas e ofertas especiais em destaque (por exemplo, no carrossel principal, banners, etc).\n"+
			"Apresenta as campanhas em formato de lista (bullet points), de forma limpa, direta e atraente.\n"+
			"Ignora livros individuais que só têm o desconto normal, foca-te nas campanhas agregadoras (ex: 'Livros Escolares 5%% imediato', '3 Livros de bolso = 1 Toalha', 'Portes Grátis', etc).\n"+
			"Se não encontrares nenhuma promoção clara no HTML fornecido, responde apenas 'Nenhuma promoção encontrada'.\n\n"+
			"HTML:\n%s", html)

	return callOpenAI(cfg, systemPrompt, userPrompt, 400, 0.3)
}
