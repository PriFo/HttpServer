package websearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	"httpserver/websearch/types"
)

// SearchHTML выполняет HTML-поиск через DuckDuckGo
// Этот метод парсит HTML-страницы результатов поиска и извлекает ссылки и сниппеты
func (c *Client) SearchHTML(ctx context.Context, query string) (*SearchResult, error) {
	// Валидация и санитизация запроса
	query = sanitizeQuery(query)
	if query == "" {
		return nil, fmt.Errorf("empty query after sanitization")
	}

	// Проверка кэша (используем отдельный ключ для HTML-поиска)
	cacheKey := generateCacheKey("html:" + query)
	if c.cache != nil {
		if cached, found := c.cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	// Проверка лимита запросов
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Формирование URL для HTML-поиска
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	// Создание запроса с контекстом
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем заголовки для имитации браузера
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")

	// Выполнение запроса
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Парсинг HTML
	result, err := c.parseHTMLResults(resp.Body, query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Сохранение в кэш
	if c.cache != nil {
		c.cache.Set(cacheKey, result)
	}

	return result, nil
}

// parseHTMLResults парсит HTML-страницу результатов поиска DuckDuckGo
func (c *Client) parseHTMLResults(body io.Reader, query string) (*types.SearchResult, error) {
	result := &types.SearchResult{
		Query:     query,
		Source:    "duckduckgo-html",
		Timestamp: time.Now(),
		Results:   make([]types.SearchItem, 0),
	}

	// Парсим HTML
	doc, err := html.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Ищем результаты поиска
	// DuckDuckGo использует класс "result" для результатов поиска
	c.extractResults(doc, result)

	// Определяем уверенность на основе количества результатов
	if len(result.Results) > 0 {
		result.Found = true
		result.Confidence = c.calculateConfidence(result.Results, query)
	} else {
		result.Found = false
		result.Confidence = 0.0
	}

	return result, nil
}

// extractResults извлекает результаты поиска из HTML-дерева
func (c *Client) extractResults(n *html.Node, result *types.SearchResult) {
	if n.Type == html.ElementNode {
		// Ищем элементы с классом "result" или "web-result"
		if c.isResultNode(n) {
			item := c.extractResultItem(n)
			if item != nil && item.URL != "" {
				result.Results = append(result.Results, *item)
			}
		}
	}

	// Рекурсивно обходим дочерние узлы
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		c.extractResults(child, result)
	}
}

// isResultNode проверяет, является ли узел результатом поиска
func (c *Client) isResultNode(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	// Проверяем класс элемента
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			classes := strings.Fields(attr.Val)
			for _, class := range classes {
				// DuckDuckGo использует различные классы для результатов
				if strings.Contains(class, "result") ||
					strings.Contains(class, "web-result") ||
					strings.Contains(class, "links_main") {
					return true
				}
			}
		}
	}

	return false
}

// extractResultItem извлекает информацию о результате из HTML-узла
func (c *Client) extractResultItem(n *html.Node) *types.SearchItem {
	item := &types.SearchItem{
		Relevance: 0.5, // Базовая релевантность
	}

	// Ищем ссылку (тег <a>)
	link := c.findLink(n)
	if link != nil {
		item.URL = link.URL
		item.Title = link.Title
	}

	// Ищем сниппет (описание результата)
	item.Snippet = c.findSnippet(n)

	// Если нет заголовка, используем сниппет
	if item.Title == "" && item.Snippet != "" {
		// Берем первые 100 символов как заголовок
		if len(item.Snippet) > 100 {
			item.Title = item.Snippet[:100] + "..."
		} else {
			item.Title = item.Snippet
		}
	}

	// Если нет URL, результат невалиден
	if item.URL == "" {
		return nil
	}

	return item
}

// linkInfo информация о ссылке
type linkInfo struct {
	URL   string
	Title string
}

// findLink находит ссылку в узле и его дочерних элементах
func (c *Client) findLink(n *html.Node) *linkInfo {
	if n.Type == html.ElementNode && n.Data == "a" {
		var href, title string
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				href = attr.Val
				// DuckDuckGo использует редиректы, извлекаем реальный URL
				if strings.HasPrefix(href, "/l/") || strings.HasPrefix(href, "//duckduckgo.com/l/") {
					// Это редирект DuckDuckGo, извлекаем реальный URL из параметра uddg
					parsed, err := url.Parse(href)
					if err == nil {
						if uddg := parsed.Query().Get("uddg"); uddg != "" {
							if decoded, err := url.QueryUnescape(uddg); err == nil {
								href = decoded
							}
						} else {
							// Альтернативный формат: ищем uddg в строке
							if idx := strings.Index(href, "uddg="); idx != -1 {
								encodedURL := href[idx+5:]
								// Убираем параметры после &
								if ampIdx := strings.Index(encodedURL, "&"); ampIdx != -1 {
									encodedURL = encodedURL[:ampIdx]
								}
								if decoded, err := url.QueryUnescape(encodedURL); err == nil {
									href = decoded
								}
							}
						}
					}
				} else if !strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://") {
					// Относительный URL, делаем абсолютным
					if strings.HasPrefix(href, "//") {
						href = "https:" + href
					} else if strings.HasPrefix(href, "/") {
						href = "https://html.duckduckgo.com" + href
					}
				}
			}
			if attr.Key == "title" {
				title = attr.Val
			}
		}

		// Если нет title в атрибутах, берем текст ссылки
		if title == "" {
			title = c.extractText(n)
		}

		if href != "" {
			return &linkInfo{URL: href, Title: strings.TrimSpace(title)}
		}
	}

	// Рекурсивно ищем в дочерних элементах
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if link := c.findLink(child); link != nil {
			return link
		}
	}

	return nil
}

// findSnippet находит сниппет (описание) результата
func (c *Client) findSnippet(n *html.Node) string {
	// Ищем элементы с классом, содержащим "snippet" или "result__snippet"
	if n.Type == html.ElementNode {
		for _, attr := range n.Attr {
			if attr.Key == "class" {
				classes := strings.Fields(attr.Val)
				for _, class := range classes {
					if strings.Contains(class, "snippet") ||
						strings.Contains(class, "result__snippet") ||
						strings.Contains(class, "result__body") {
						text := c.extractText(n)
						if text != "" {
							return strings.TrimSpace(text)
						}
					}
				}
			}
		}
	}

	// Рекурсивно ищем в дочерних элементах
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if snippet := c.findSnippet(child); snippet != "" {
			return snippet
		}
	}

	return ""
}

// extractText извлекает текст из узла и его дочерних элементов
func (c *Client) extractText(n *html.Node) string {
	var text strings.Builder

	var extract func(*html.Node)
	extract = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			extract(child)
		}
	}

	extract(n)
	return strings.TrimSpace(text.String())
}

// calculateConfidence вычисляет уверенность на основе результатов
func (c *Client) calculateConfidence(results []SearchItem, query string) float64 {
	if len(results) == 0 {
		return 0.0
	}

	// Базовая уверенность зависит от количества результатов
	baseConfidence := 0.5
	if len(results) >= 5 {
		baseConfidence = 0.8
	} else if len(results) >= 3 {
		baseConfidence = 0.7
	} else if len(results) >= 1 {
		baseConfidence = 0.6
	}

	// Увеличиваем уверенность, если запрос встречается в результатах
	queryLower := strings.ToLower(query)
	matches := 0
	for _, result := range results {
		titleLower := strings.ToLower(result.Title)
		snippetLower := strings.ToLower(result.Snippet)
		if strings.Contains(titleLower, queryLower) || strings.Contains(snippetLower, queryLower) {
			matches++
		}
	}

	// Добавляем бонус за совпадения
	if matches > 0 {
		matchBonus := float64(matches) / float64(len(results)) * 0.2
		baseConfidence += matchBonus
		if baseConfidence > 1.0 {
			baseConfidence = 1.0
		}
	}

	return baseConfidence
}

