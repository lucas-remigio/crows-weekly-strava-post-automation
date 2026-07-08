package main

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

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

	// Optimization: Try to isolate only the main banner section to drastically shrink the payload
	reCarousel := regexp.MustCompile(`(?is)<section class="personalized header-banner.*?</section>`)
	if carouselMatch := reCarousel.FindString(html); carouselMatch != "" {
		html = carouselMatch
		slog.Info("Isolated main carousel section", "new_length", len(html))
	} else {
		slog.Warn("Could not isolate main carousel, using full HTML")
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
	return fetchHTML("https://www.wook.pt", timeoutSeconds)
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
