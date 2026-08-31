package ai

import (
	"context"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/rs/zerolog"
)

type Service struct {
	logger zerolog.Logger
}

func NewService(logger zerolog.Logger) *Service {
	return &Service{
		logger: logger.With().Str("module", "ai").Logger(),
	}
}

type TranslateInput struct {
	Text           string `json:"text"`
	SourceLanguage string `json:"source_language"`
	TargetLanguage string `json:"target_language"`
}

type TranslateResult struct {
	TranslatedText string `json:"translated_text"`
	SourceLanguage string `json:"source_language"`
	TargetLanguage string `json:"target_language"`
	Confidence     float64 `json:"confidence"`
}

type HeadlineOptions struct {
	Headline         string   `json:"headline"`
	AlternativeTitle string   `json:"alternative_title"`
	SEOKeywords      []string `json:"seo_keywords"`
	ReadingTimeMin   int      `json:"reading_time_min"`
	Excerpt          string   `json:"excerpt"`
}

// GenerateEditorialAssistance analyzes content body and produces SEO metadata, excerpts, and reading times.
func (s *Service) GenerateEditorialAssistance(ctx context.Context, title, bodyText string) (*HeadlineOptions, error) {
	wordCount := len(strings.Fields(bodyText))
	readingTime := int(math.Ceil(float64(wordCount) / 200.0))
	if readingTime < 1 {
		readingTime = 1
	}

	// Smart excerpt extraction
	sentences := strings.Split(bodyText, ".")
	excerpt := ""
	if len(sentences) > 0 {
		excerpt = strings.TrimSpace(sentences[0])
		if utf8.RuneCountInString(excerpt) > 160 {
			runes := []rune(excerpt)
			excerpt = string(runes[:157]) + "..."
		}
	}

	keywords := []string{"Breaking News", "Latest Updates", "Exclusive Report"}
	if strings.Contains(strings.ToLower(bodyText), "delhi") || strings.Contains(strings.ToLower(bodyText), "mumbai") {
		keywords = append(keywords, "National News")
	}
	if strings.Contains(strings.ToLower(bodyText), "cricket") || strings.Contains(strings.ToLower(bodyText), "match") {
		keywords = append(keywords, "Sports")
	}

	return &HeadlineOptions{
		Headline:         title,
		AlternativeTitle: fmt.Sprintf("EXPLAINED: %s", title),
		SEOKeywords:      keywords,
		ReadingTimeMin:   readingTime,
		Excerpt:          excerpt,
	}, nil
}
