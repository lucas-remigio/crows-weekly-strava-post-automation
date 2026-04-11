package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
)

const openAIURL = "https://api.openai.com/v1/chat/completions"

func generateWeeklyRoast(cfg Config, athletes []Athlete, abovePace bool, diffKM float64) string {
	if cfg.OpenAIAPIKey == "" {
		slog.Info("OPENAI_API_KEY not configured — skipping weekly roast.")
		return ""
	}
	if len(athletes) == 0 {
		slog.Info("Athletes list is empty — skipping weekly roast.")
		return ""
	}

	athlete := athletes[rand.Intn(len(athletes))]
	slog.Info("Generating roast for athlete", "name", athlete.Name)

	systemPrompt := "O teu humor é inspirado em Ricardo Araújo Pereira: inteligente, irónico e absolutamente certeiro. " +
		"Usas o absurdo com precisão cirúrgica. As tuas frases têm sempre uma lógica interna impecável que " +
		"torna o disparate completamente inevitável. Não explicas, não exageras, não usas pontos de exclamação. " +
		"O humor nasce da observação fria de factos ridículos, dita com a seriedade de quem está a ler uma acta. " +
		"Escreves em português europeu, culto mas acessível, sem calão e sem emojis. " +
		"Começas sempre com um 'Bom dia' muito caloroso e entusiástico, como quem acorda um grupo às 5 da manhã com carinho. " +
		"Logo a seguir, fazes uma pequena observação divertida e contextual sobre como a semana correu. "

	var situation string
	if abovePace {
		situation = fmt.Sprintf("O clube está %.0f km acima do ritmo anual.", diffKM)
	} else {
		situation = fmt.Sprintf("O clube está %.0f km abaixo do ritmo anual.", diffKM)
	}

	userPrompt := fmt.Sprintf(
		"%s %s é conhecido por %s. "+
			"Se o clube está acima do ritmo anual, deixa isso transparecer com um tom triunfal e brincalhão. "+
			"Se está abaixo, deixa isso transparecer com um tom simpático, energético e ligeiramente alarmado, mas sempre caloroso. "+
			"Podes referir que podem agradecer ao %s, mas também podes variar para elogiar, ironizar, ou virar a frase de outra forma. "+
			"Escreve uma única frase sobre %s que relacione a sua personalidade com este resultado. "+
			"Não expliques a piada. "+
			"Surpreende-nos. A mensagem deve começar com uma saudação tipo 'Bom dia'. Responde apenas com a frase.",
		situation, athlete.Name, athlete.Characteristic, athlete.Name, athlete.Name,
	)

	body, err := json.Marshal(map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens":  120,
		"temperature": 1.1,
	})
	if err != nil {
		slog.Error("Failed to marshal OpenAI request", "error", err)
		return ""
	}

	req, err := http.NewRequest(http.MethodPost, openAIURL, bytes.NewReader(body))
	if err != nil {
		slog.Error("Failed to create OpenAI request", "error", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIAPIKey)

	client := httpClient(cfg.HTTPTimeoutSeconds)
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("OpenAI request failed", "error", err)
		return ""
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		slog.Error("OpenAI API error", "error", err)
		return ""
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Error("Failed to decode OpenAI response", "error", err)
		return ""
	}

	if len(result.Choices) == 0 {
		return ""
	}

	roast := strings.TrimSpace(result.Choices[0].Message.Content)
	slog.Info("Roast generated", "roast", roast)
	return roast
}
