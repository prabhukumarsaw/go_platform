package content

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BlockType defines supported rich content node types.
type BlockType string

const (
	BlockParagraph    BlockType = "paragraph"
	BlockHeading      BlockType = "heading"
	BlockImageGallery BlockType = "image_gallery"
	BlockTable        BlockType = "table"
	BlockYouTube      BlockType = "youtube"
	BlockTwitter      BlockType = "twitter"
	BlockInstagram    BlockType = "instagram"
	BlockFacebook     BlockType = "facebook"
	BlockPoll         BlockType = "poll"
	BlockQuote        BlockType = "quote"
	BlockFactCheck    BlockType = "fact_check"
	BlockLiveUpdate   BlockType = "live_update"
	BlockCallout      BlockType = "callout"
)

// ContentBlock represents a generic node in the modern editorial block tree.
type ContentBlock struct {
	ID   string          `json:"id"`
	Type BlockType       `json:"type"`
	Data json.RawMessage `json:"data"`
}

// ─── Specific Node Payloads ─────────────────────

type ParagraphNode struct {
	Text string `json:"text"`
}

type HeadingNode struct {
	Level int    `json:"level"` // 1 - 6
	Text  string `json:"text"`
}

type GalleryImageItem struct {
	URL     string `json:"url"`
	Caption string `json:"caption,omitempty"`
	AltText string `json:"alt_text,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
}

type ImageGalleryNode struct {
	Layout string             `json:"layout"` // grid, carousel, masonry
	Images []GalleryImageItem `json:"images"`
}

type TableNode struct {
	Caption string     `json:"caption,omitempty"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

type EmbedYouTubeNode struct {
	URL     string `json:"url"`
	VideoID string `json:"video_id"`
	Title   string `json:"title,omitempty"`
}

type EmbedTwitterNode struct {
	TweetURL     string `json:"tweet_url"`
	TweetID      string `json:"tweet_id,omitempty"`
	AuthorHandle string `json:"author_handle,omitempty"`
}

type EmbedInstagramNode struct {
	PostURL string `json:"post_url"`
	PostID  string `json:"post_id,omitempty"`
}

type EmbedFacebookNode struct {
	PostURL string `json:"post_url"`
}

type EmbedPollNode struct {
	PollID   string `json:"poll_id"`
	Question string `json:"question,omitempty"`
}

type QuoteNode struct {
	Quote       string `json:"quote"`
	Author      string `json:"author,omitempty"`
	Designation string `json:"designation,omitempty"`
}

type FactCheckNode struct {
	Claim       string `json:"claim"`
	Claimant    string `json:"claimant,omitempty"`
	Verdict     string `json:"verdict"` // True, False, Misleading, Partially True
	Explanation string `json:"explanation"`
}

type LiveUpdateNode struct {
	Timestamp string `json:"timestamp"`
	Headline  string `json:"headline"`
	Body      string `json:"body"`
}

type CalloutNode struct {
	Style string `json:"style"` // info, warning, breaking, success
	Title string `json:"title,omitempty"`
	Text  string `json:"text"`
}

// ─── Block Parser & Transformer ─────────────────

// ParseBlocks parses and validates a raw JSON block document.
func ParseBlocks(raw json.RawMessage) ([]ContentBlock, error) {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return []ContentBlock{}, nil
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		// Attempt parsing as object with `blocks` array (Editor.js standard)
		var wrapper struct {
			Blocks []ContentBlock `json:"blocks"`
		}
		if wrapErr := json.Unmarshal(raw, &wrapper); wrapErr == nil && len(wrapper.Blocks) > 0 {
			return wrapper.Blocks, nil
		}
		return nil, fmt.Errorf("invalid blocks structure: %w", err)
	}

	return blocks, nil
}

// BlocksToPlainText extracts searchable plain text and calculates word count from blocks.
func BlocksToPlainText(raw json.RawMessage) (string, int) {
	blocks, err := ParseBlocks(raw)
	if err != nil || len(blocks) == 0 {
		return string(raw), len(strings.Fields(string(raw)))
	}

	var sb strings.Builder
	for _, b := range blocks {
		switch b.Type {
		case BlockParagraph:
			var p ParagraphNode
			if err := json.Unmarshal(b.Data, &p); err == nil {
				sb.WriteString(p.Text)
				sb.WriteString("\n\n")
			}
		case BlockHeading:
			var h HeadingNode
			if err := json.Unmarshal(b.Data, &h); err == nil {
				sb.WriteString(h.Text)
				sb.WriteString("\n\n")
			}
		case BlockQuote:
			var q QuoteNode
			if err := json.Unmarshal(b.Data, &q); err == nil {
				sb.WriteString(q.Quote)
				sb.WriteString("\n\n")
			}
		case BlockFactCheck:
			var f FactCheckNode
			if err := json.Unmarshal(b.Data, &f); err == nil {
				sb.WriteString(f.Claim)
				sb.WriteString(" ")
				sb.WriteString(f.Explanation)
				sb.WriteString("\n\n")
			}
		case BlockLiveUpdate:
			var l LiveUpdateNode
			if err := json.Unmarshal(b.Data, &l); err == nil {
				sb.WriteString(l.Headline)
				sb.WriteString(" ")
				sb.WriteString(l.Body)
				sb.WriteString("\n\n")
			}
		}
	}

	text := strings.TrimSpace(sb.String())
	words := len(strings.Fields(text))
	return text, words
}

// BlocksToHTML generates standard, clean HTML markup from rich blocks for SEO and RSS feeds.
func BlocksToHTML(raw json.RawMessage) string {
	blocks, err := ParseBlocks(raw)
	if err != nil || len(blocks) == 0 {
		return fmt.Sprintf("<p>%s</p>", string(raw))
	}

	var sb strings.Builder
	for _, b := range blocks {
		switch b.Type {
		case BlockParagraph:
			var p ParagraphNode
			_ = json.Unmarshal(b.Data, &p)
			sb.WriteString(fmt.Sprintf("<p>%s</p>\n", p.Text))

		case BlockHeading:
			var h HeadingNode
			_ = json.Unmarshal(b.Data, &h)
			if h.Level < 1 || h.Level > 6 {
				h.Level = 2
			}
			sb.WriteString(fmt.Sprintf("<h%d>%s</h%d>\n", h.Level, h.Text, h.Level))

		case BlockImageGallery:
			var g ImageGalleryNode
			_ = json.Unmarshal(b.Data, &g)
			sb.WriteString("<div class=\"article-gallery\">\n")
			for _, img := range g.Images {
				sb.WriteString(fmt.Sprintf("  <figure><img src=\"%s\" alt=\"%s\" /><figcaption>%s</figcaption></figure>\n", img.URL, img.AltText, img.Caption))
			}
			sb.WriteString("</div>\n")

		case BlockTable:
			var t TableNode
			_ = json.Unmarshal(b.Data, &t)
			sb.WriteString("<table>\n")
			if len(t.Headers) > 0 {
				sb.WriteString("  <thead><tr>\n")
				for _, th := range t.Headers {
					sb.WriteString(fmt.Sprintf("    <th>%s</th>\n", th))
				}
				sb.WriteString("  </tr></thead>\n")
			}
			sb.WriteString("  <tbody>\n")
			for _, row := range t.Rows {
				sb.WriteString("  <tr>\n")
				for _, td := range row {
					sb.WriteString(fmt.Sprintf("    <td>%s</td>\n", td))
				}
				sb.WriteString("  </tr>\n")
			}
			sb.WriteString("  </tbody>\n</table>\n")

		case BlockYouTube:
			var y EmbedYouTubeNode
			_ = json.Unmarshal(b.Data, &y)
			sb.WriteString(fmt.Sprintf("<div class=\"embed-video\"><iframe src=\"https://www.youtube.com/embed/%s\" allowfullscreen></iframe></div>\n", y.VideoID))

		case BlockTwitter:
			var tw EmbedTwitterNode
			_ = json.Unmarshal(b.Data, &tw)
			sb.WriteString(fmt.Sprintf("<blockquote class=\"twitter-tweet\"><a href=\"%s\"></a></blockquote>\n", tw.TweetURL))

		case BlockInstagram:
			var ig EmbedInstagramNode
			_ = json.Unmarshal(b.Data, &ig)
			sb.WriteString(fmt.Sprintf("<blockquote class=\"instagram-media\" data-instgrm-permalink=\"%s\"></blockquote>\n", ig.PostURL))

		case BlockQuote:
			var q QuoteNode
			_ = json.Unmarshal(b.Data, &q)
			sb.WriteString(fmt.Sprintf("<blockquote><p>%s</p><cite>%s</cite></blockquote>\n", q.Quote, q.Author))

		case BlockFactCheck:
			var f FactCheckNode
			_ = json.Unmarshal(b.Data, &f)
			sb.WriteString(fmt.Sprintf("<div class=\"fact-check-box\"><div class=\"verdict\">%s</div><p class=\"claim\">%s</p><p>%s</p></div>\n", f.Verdict, f.Claim, f.Explanation))

		case BlockLiveUpdate:
			var l LiveUpdateNode
			_ = json.Unmarshal(b.Data, &l)
			if l.Timestamp == "" {
				l.Timestamp = time.Now().Format("15:04 IST")
			}
			sb.WriteString(fmt.Sprintf("<div class=\"live-event\"><time>%s</time><h4>%s</h4><p>%s</p></div>\n", l.Timestamp, l.Headline, l.Body))
		}
	}

	return sb.String()
}
