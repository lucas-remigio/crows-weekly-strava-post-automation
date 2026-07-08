package main

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

func checkFnacPromo(cfg Config) {
	if cfg.WookTelegramChatID == "" {
		slog.Warn("No WOOK_TELEGRAM_CHAT_ID configured, skipping fnac check.")
		return
	}

	msg, err := getFnacPromoMessage(cfg)
	if err != nil {
		slog.Error("Failed to get fnac promo message", "error", err)
		return
	}
	if msg == "" {
		slog.Info("No active fnac promo found.")
		return
	}

	// Send to Telegram (reusing the wook telegram chat ID for now since it's the personal one)
	client := httpClient(cfg.HTTPTimeoutSeconds)
	if err := sendToOne(client, cfg.TelegramBotToken, cfg.WookTelegramChatID, msg); err != nil {
		slog.Error("Failed to send Fnac Telegram message", "error", err)
	}
}

func getFnacPromoMessage(cfg Config) (string, error) {
	html, err := fetchFnacHTML(cfg.HTTPTimeoutSeconds)
	if err != nil {
		return "", fmt.Errorf("fetch Fnac HTML: %w", err)
	}

	_ = os.WriteFile("fnac_debug.html", []byte(html), 0644)
	slog.Info("Fetched Fnac HTML", "length", len(html), "file", "fnac_debug.html")

	if cfg.OpenAIAPIKey == "" {
		return "", fmt.Errorf("No OPENAI_API_KEY configured")
	}

	// Optimization: Try to isolate only the main banner section to drastically shrink the payload
	reCarousel := regexp.MustCompile(`(?is)<section class="strate stratePlayer[^>]*>.*?</section>`)
	if carouselMatch := reCarousel.FindString(html); carouselMatch != "" {
		html = carouselMatch
		slog.Info("Isolated fnac main carousel section", "new_length", len(html))
	} else {
		slog.Warn("Could not isolate fnac main carousel, using full HTML")
	}

	// Strip scripts and styles to save tokens and avoid truncation of real content
	reScript := regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	reStyle := regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	cleanHTML := reScript.ReplaceAllString(html, "")
	cleanHTML = reStyle.ReplaceAllString(cleanHTML, "")

	slog.Info("Cleaned Fnac HTML (removed scripts/styles)", "old_length", len(html), "new_length", len(cleanHTML))

	// OpenAI has a high token limit for gpt-4o-mini, let's allow up to 150k chars
	limit := 50000 // Fnac might be larger and harder to isolate via regex easily, so we allow a bit more
	if len(cleanHTML) > limit {
		slog.Info("Trimmed the content of the fnac html to fit OpenAI limit", "limit", limit)
		cleanHTML = cleanHTML[:limit]
	}

	_ = os.WriteFile("fnac_debug_clean.html", []byte(cleanHTML), 0644)
	slog.Info("Ready to call OpenAI for Fnac", "payload_length", len(cleanHTML), "file", "fnac_debug_clean.html")

	promo, err := extractFnacPromoWithOpenAI(cfg, cleanHTML)
	if err != nil {
		return "", fmt.Errorf("extract with OpenAI: %w", err)
	}

	if promo == "" || strings.Contains(strings.ToLower(promo), "nenhuma promoção") {
		return "", nil
	}

	slog.Info("Current fnac promo found", "promo", promo)

	return fmt.Sprintf("🟡 *FNAC Promoção do Dia*\n\n%s\n\n🔗 https://www.fnac.pt", promo), nil
}

func fetchFnacHTML(timeoutSeconds int) (string, error) {
	return fetchHTML("https://www.fnac.pt/livro/h5", timeoutSeconds)
}

func extractFnacPromoWithOpenAI(cfg Config, html string) (string, error) {
	systemPrompt := "És um assistente útil e focado na extração de dados."
	userPrompt := fmt.Sprintf(
		"Aqui está o código HTML limpo da página inicial da Fnac (fnac.pt). "+
			"A tua tarefa é extrair TODAS as campanhas e ofertas especiais em destaque na homepage.\n"+
			"Apresenta as campanhas em formato de lista (bullet points), de forma limpa, direta e atraente.\n"+
			"Se não encontrares nenhuma promoção clara no HTML fornecido, responde apenas 'Nenhuma promoção encontrada'.\n\n"+
			"HTML:\n%s", html)

	return callOpenAI(cfg, systemPrompt, userPrompt, 400, 0.3)
}
