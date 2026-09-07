package content

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"newsplatform/api/pkg/response"
)

// ─────────────────────────────────────────────────────────────────────────────
// Models & Data Contracts
// ─────────────────────────────────────────────────────────────────────────────

// HomepageSection represents a dynamic homepage section stored in PostgreSQL.
type HomepageSection struct {
	ID           int                    `json:"id"`
	SectionKey   string                 `json:"section_key"`
	Title        string                 `json:"title"`
	TitleHi      string                 `json:"title_hi"`
	SectionType  string                 `json:"section_type"`
	CategoryID   *int                   `json:"category_id,omitempty"`
	CategorySlug string                 `json:"category_slug"`
	CategoryName string                 `json:"category_name,omitempty"`
	SortOrder    int                    `json:"sort_order"`
	IsActive     bool                   `json:"is_active"`
	ArticleLimit int                    `json:"article_limit"`
	Settings     map[string]interface{} `json:"settings"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// CreateSectionInput defines payload to add a new category/section to homepage.
type CreateSectionInput struct {
	SectionKey   string                 `json:"section_key"`
	Title        string                 `json:"title"`
	TitleHi      string                 `json:"title_hi"`
	SectionType  string                 `json:"section_type"`
	CategoryID   *int                   `json:"category_id,omitempty"`
	CategorySlug string                 `json:"category_slug"`
	SortOrder    int                    `json:"sort_order"`
	IsActive     *bool                  `json:"is_active"`
	ArticleLimit int                    `json:"article_limit"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
}

// UpdateSectionInput defines payload to update an existing homepage section.
type UpdateSectionInput struct {
	Title        *string                `json:"title,omitempty"`
	TitleHi      *string                `json:"title_hi,omitempty"`
	SectionType  *string                `json:"section_type,omitempty"`
	CategoryID   *int                   `json:"category_id,omitempty"`
	CategorySlug *string                `json:"category_slug,omitempty"`
	SortOrder    *int                   `json:"sort_order,omitempty"`
	IsActive     *bool                  `json:"is_active,omitempty"`
	ArticleLimit *int                   `json:"article_limit,omitempty"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
}

// ReorderSectionsInput defines batch order update.
type ReorderSectionsInput struct {
	OrderedIDs []int `json:"ordered_ids"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Database Schema & Automatic Seed
// ─────────────────────────────────────────────────────────────────────────────

// EnsureHomepageSchema ensures the homepage_sections table exists and seeds it with default hybrid modules.
func (s *Service) EnsureHomepageSchema(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS homepage_sections (
			id SERIAL PRIMARY KEY,
			section_key VARCHAR(100) UNIQUE NOT NULL,
			title VARCHAR(255) NOT NULL,
			title_hi VARCHAR(255) NOT NULL,
			section_type VARCHAR(50) NOT NULL,
			category_id INT REFERENCES categories(id) ON DELETE SET NULL,
			category_slug VARCHAR(100) NOT NULL DEFAULT '',
			sort_order INT NOT NULL DEFAULT 1,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			article_limit INT NOT NULL DEFAULT 6,
			settings JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_homepage_sections_order ON homepage_sections(sort_order, is_active);
	`
	if _, err := s.pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create homepage_sections table: %w", err)
	}

	// Check if already seeded
	var count int
	_ = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM homepage_sections").Scan(&count)
	if count == 0 {
		s.seedDefaultHomepageSections(ctx)
	}

	// Also seed comprehensive database data (articles, categories, videos, web stories)
	s.seedComprehensiveContent(ctx)

	return nil
}

func (s *Service) seedDefaultHomepageSections(ctx context.Context) {
	defaultSections := []struct {
		Key      string
		Title    string
		TitleHi  string
		Type     string
		CatSlug  string
		Order    int
		Limit    int
		Settings string
	}{
		{"featured_hero", "Featured Top Stories", "प्रमुख समाचार", "featured_hero", "national", 1, 9, `{"show_breaking": true}`},
		{"four_col_1", "Multi-Category Desks I", "विशेष श्रेणियां १", "four_column", "", 2, 20, `{"categories": ["business", "auto", "lifestyle", "health"]}`},
		{"state_desks", "State Regional Desks", "राज्य ब्यूरो (झारखंड व बिहार)", "state_desks", "jharkhand", 3, 14, `{"secondary_slug": "bihar"}`},
		{"video_showcase", "Bawaal Video News", "वीडियो बुलेटिन", "video_showcase", "video", 4, 8, `{}`},
		{"politics", "Politics & Governance", "राजनीति", "category_block_a", "politics", 5, 5, `{"image_position": "left"}`},
		{"sports", "Sports Arena", "खेल जगत", "category_block_a", "sports", 6, 5, `{"image_position": "right"}`},
		{"entertainment", "Cinema & Entertainment", "मनोरंजन", "category_block_b", "entertainment", 7, 7, `{}`},
		{"crime", "Crime & Investigations", "अपराध एवं खुलासे", "category_block_a", "crime", 8, 5, `{"image_position": "left"}`},
		{"business", "Business & Markets", "व्यापार एवं अर्थव्यवस्था", "category_block_b", "business", 9, 7, `{}`},
		{"four_col_2", "Multi-Category Desks II", "विशेष श्रेणियां २", "four_column", "", 10, 20, `{"categories": ["lifestyle", "science", "dharma", "environment"]}`},
		{"technology", "Technology & Gadgets", "टेक्नोलॉजी", "technology", "technology", 11, 6, `{}`},
	}

	for _, sec := range defaultSections {
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO homepage_sections (section_key, title, title_hi, section_type, category_slug, sort_order, article_limit, is_active, settings)
			VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8::jsonb)
			ON CONFLICT (section_key) DO UPDATE
			SET title = EXCLUDED.title,
			    title_hi = EXCLUDED.title_hi,
			    sort_order = EXCLUDED.sort_order;
		`, sec.Key, sec.Title, sec.TitleHi, sec.Type, sec.CatSlug, sec.Order, sec.Limit, sec.Settings)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Database Seeding: Comprehensive Live News, Categories, Videos & Stories
// ─────────────────────────────────────────────────────────────────────────────

func (s *Service) seedComprehensiveContent(ctx context.Context) {
	// 1. Seed Categories if missing
	categoriesSeed := []struct {
		Name        string
		Slug        string
		NameHi      string
		AccentColor string
	}{
		{"National", "national", "राष्ट्रीय", "#dc2626"},
		{"Politics", "politics", "राजनीति", "#2563eb"},
		{"Sports", "sports", "खेल", "#16a34a"},
		{"Entertainment", "entertainment", "मनोरंजन", "#db2777"},
		{"Crime", "crime", "अपराध", "#ea580c"},
		{"Business", "business", "व्यापार", "#0d9488"},
		{"Technology", "technology", "टेक्नोलॉजी", "#7c3aed"},
		{"Jharkhand", "jharkhand", "झारखंड", "#0284c7"},
		{"Bihar", "bihar", "बिहार", "#d97706"},
		{"Auto", "auto", "ऑटोमोबाइल", "#4f46e5"},
		{"Health", "health", "स्वास्थ्य", "#059669"},
		{"Lifestyle", "lifestyle", "लाइफस्टाइल", "#e11d48"},
		{"Dharma", "dharma", "धर्म", "#b45309"},
		{"Environment", "environment", "पर्यावरण", "#15803d"},
		{"International", "international", "अंतरराष्ट्रीय", "#3b82f6"},
	}

	for _, c := range categoriesSeed {
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO categories (name, slug, level, accent_color)
			VALUES ($1, $2, 1, $3)
			ON CONFLICT (slug) DO UPDATE
			SET name = EXCLUDED.name,
			    accent_color = EXCLUDED.accent_color;
		`, c.Name, c.Slug, c.AccentColor)
	}

	// 2. Seed Realistic Database Articles for each category so zero sections are empty
	type SeedArticle struct {
		Title       string
		Slug        string
		Category    string
		Image       string
		Excerpt     string
		IsBreaking  bool
		IsFeatured  bool
		ViewCount   int64
	}

	articlesSeed := []SeedArticle{
		// National & Breaking
		{
			Title:       "संसद का विशेष सत्र: कई ऐतिहासिक विधेयकों पर गहन चर्चा, लोकसभा में सत्ता पक्ष और विपक्ष में तीखी बहस",
			Slug:        "parliament-special-session-historic-bills-debate",
			Category:    "politics",
			Image:       "https://images.unsplash.com/photo-1541872703-74c5e44368f9?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "संसद के बजट सत्र के दौरान कई अहम मुद्दों पर चर्चा जारी है। वित्त मंत्री ने नए सुधारों का ब्यौरा सदन के पटल पर रखा।",
			IsBreaking:  true,
			IsFeatured:  true,
			ViewCount:   18450,
		},
		{
			Title:       "चुनाव आयोग की नई गाइडलाइंस: चुनावी रैलियों और डिजिटल प्रचार को लेकर सख्त दिशा-निर्देश जारी",
			Slug:        "election-commission-new-guidelines-digital-campaigning",
			Category:    "politics",
			Image:       "https://images.unsplash.com/photo-1540910419892-4a36d2c3266c?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "मुख्य चुनाव आयुक्त ने राज्यों के विधानसभा चुनाव को लेकर प्रेस कॉन्फ्रेंस में सभी राजनीतिक दलों से नियमों का पालन करने की अपील की।",
			IsBreaking:  false,
			IsFeatured:  true,
			ViewCount:   9320,
		},
		{
			Title:       "सुप्रीम कोर्ट का ऐतिहासिक फैसला: नागरिकों के मौलिक अधिकारों के संरक्षण पर संविधान पीठ की बड़ी टिप्पणी",
			Slug:        "supreme-court-landmark-verdict-fundamental-rights",
			Category:    "politics",
			Image:       "https://images.unsplash.com/photo-1589829545856-d10d557cf95f?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "पांच जजों की संविधान पीठ ने सर्वसम्मति से महत्वपूर्ण व्यवस्था देते हुए जांच एजेंसियों की शक्तियों पर स्पष्ट व्याख्या की।",
			IsBreaking:  true,
			IsFeatured:  false,
			ViewCount:   12600,
		},
		// Sports
		{
			Title:       "टी20 वर्ल्ड कप फाइनल: भारत ने रचा इतिहास, रोमांचक मुकाबले में अंतिम ओवर में दर्ज की ऐतिहासिक जीत",
			Slug:        "t20-world-cup-final-india-historic-win",
			Category:    "sports",
			Image:       "https://images.unsplash.com/photo-1531415074868-036b1c57e359?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "भारतीय टीम ने शानदार खेल का प्रदर्शन करते हुए ट्रॉफी पर कब्जा जमाया। कप्तान ने जीत को देशवासियों को समर्पित किया।",
			IsBreaking:  true,
			IsFeatured:  true,
			ViewCount:   35200,
		},
		{
			Title:       "आईपीएल 2026 की मेगा नीलामी: युवा भारतीय खिलाड़ियों पर बरसे करोड़ों, रिकॉर्ड तोड़ बोलियों से चौंकाया",
			Slug:        "ipl-2026-mega-auction-record-bids-young-talents",
			Category:    "sports",
			Image:       "https://images.unsplash.com/photo-1540747913346-19e32dc3e97e?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "घरेलू क्रिकेट में बेहतरीन प्रदर्शन करने वाले अनकैप्ड खिलाड़ियों को फ्रेंचाइजियों ने भारी रकम देकर अपनी टीम में शामिल किया।",
			IsBreaking:  false,
			IsFeatured:  false,
			ViewCount:   14300,
		},
		// Business
		{
			Title:       "भारतीय शेयर बाजार में रिकॉर्ड तेजी: सेंसेक्स 85 हजार के पार, निवेशकों की संपत्ति में 4 लाख करोड़ का इजाफा",
			Slug:        "stock-market-record-high-sensex-crosses-85000",
			Category:    "business",
			Image:       "https://images.unsplash.com/photo-1611974789855-9c2a0a7236a3?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "बैंकिंग और आईटी शेयरों में चौतरफा लिवाली से बाजार नई ऊंचाई पर पहुंचा। विदेशी संस्थागत निवेशकों ने भारी निवेश किया।",
			IsBreaking:  false,
			IsFeatured:  true,
			ViewCount:   22100,
		},
		{
			Title:       "आरबीआई मौद्रिक नीति समीक्षा: रेपो रेट में बदलाव नहीं, जीडीपी विकास दर 7.2% रहने का मजबूत अनुमान",
			Slug:        "rbi-monetary-policy-repo-rate-unchanged-gdp-forecast",
			Category:    "business",
			Image:       "https://images.unsplash.com/photo-1526304640581-d334cdbbf45e?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "गवर्नर ने प्रेस कॉन्फ्रेंस में बताया कि खुदरा महंगाई नियंत्रण में है और ग्रामीण अर्थव्यवस्था में उल्लेखनीय सुधार देखा जा रहा है।",
			IsBreaking:  false,
			IsFeatured:  false,
			ViewCount:   11500,
		},
		// Entertainment
		{
			Title:       "ऑस्कर 2026: भारतीय सिनेमा का फिर बजा डंका, बेस्ट डॉक्युमेंट्री और ओरिजिनल स्कोर में जीते प्रतिष्ठित अवॉर्ड्स",
			Slug:        "oscars-2026-indian-cinema-wins-prestigious-awards",
			Category:    "entertainment",
			Image:       "https://images.unsplash.com/photo-1489599849927-2ee91cede3ba?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "लॉस एंजिल्स में आयोजित भव्य समारोह में भारतीय कलाकारों ने वैश्विक मंच पर तिरंगा लहराया। पूरे देश से बधाइयों का तांता।",
			IsBreaking:  false,
			IsFeatured:  true,
			ViewCount:   19800,
		},
		{
			Title:       "बॉक्स ऑफिस धमाका: नई एक्शन थ्रिलर ने पहले वीकेंड में कमाए 250 करोड़, तोड़े कई पुराने रिकॉर्ड",
			Slug:        "box-office-action-thriller-weekend-collection-250-crore",
			Category:    "entertainment",
			Image:       "https://images.unsplash.com/photo-1517604931442-7e0c8ed2963c?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "दर्शकों के जबर्दस्त रिस्पॉन्स और हाउसफुल शोज के चलते फिल्म ने घरेलू और विदेशी दोनों बाजारों में बंपर कमाई की है।",
			IsBreaking:  false,
			IsFeatured:  false,
			ViewCount:   16700,
		},
		// Crime
		{
			Title:       "साइबर सुरक्षा एजेंसी का बड़ा ऑपरेशन: अंतरराष्ट्रीय डिजिटल फ्रॉड गिरोह का भंडाफोड़, 18 साइबर अपराधी गिरफ्तार",
			Slug:        "cyber-crime-international-fraud-gang-busted-18-arrested",
			Category:    "crime",
			Image:       "https://images.unsplash.com/photo-1563986768609-322da13575f3?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "देशभर के हजारों लोगों से ठगी करने वाले गिरोह से करोड़ों के डिजिटल उपकरण, फर्जी सिम कार्ड और बैंक खाते बरामद किए गए।",
			IsBreaking:  false,
			IsFeatured:  true,
			ViewCount:   13400,
		},
		// Technology
		{
			Title:       "भारत में 6G तकनीक का सफल परीक्षण: टेलीकॉम सेक्टर में अगली पीढ़ी की क्रांति की ओर बढ़ा देश",
			Slug:        "india-6g-technology-successful-trials-telecom-revolution",
			Category:    "technology",
			Image:       "https://images.unsplash.com/photo-1518770660439-4636190af475?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "स्वदेशी अनुसंधान संस्थानों और टेक कंपनियों के संयुक्त प्रयास से अल्ट्रा-लो लेटेंसी और सुपरफास्ट डेटा स्पीड हासिल की गई।",
			IsBreaking:  false,
			IsFeatured:  true,
			ViewCount:   21400,
		},
		{
			Title:       "आर्टिफिशियल इंटेलिजेंस और स्वास्थ्य: एम्स में एआई आधारित डायग्नोस्टिक सिस्टम शुरू, मिनटों में गंभीर रोगों की पहचान",
			Slug:        "ai-healthcare-aiims-diagnostic-system-disease-detection",
			Category:    "technology",
			Image:       "https://images.unsplash.com/photo-1532938911079-1b06ac7ceec7?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "डॉक्टरों की टीम ने बताया कि यह प्रणाली 98 प्रतिशत से अधिक सटीकता के साथ एक्स-रे और एमआरआई स्कैन का विश्लेषण करती है।",
			IsBreaking:  false,
			IsFeatured:  false,
			ViewCount:   15200,
		},
		// Jharkhand & Bihar Desks
		{
			Title:       "झारखंड कैबिनेट के बड़े फैसले: युवाओं के लिए रोजगार नीति को मंजूरी, कई विकास परियोजनाओं पर मुहर",
			Slug:        "jharkhand-cabinet-major-decisions-employment-policy-approved",
			Category:    "jharkhand",
			Image:       "https://images.unsplash.com/photo-1486406146926-c627a92ad1ab?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "मुख्यमंत्री की अध्यक्षता में हुई बैठक में राज्य में औद्योगिक निवेश बढ़ाने और कौशल विकास केंद्रों की स्थापना का निर्णय हुआ।",
			IsBreaking:  false,
			IsFeatured:  true,
			ViewCount:   17900,
		},
		{
			Title:       "बिहार में एक्सप्रेसवे नेटवर्क का विस्तार: पटना-पूर्णिया 4-लेन ग्रीनफील्ड कॉरिडोर का निर्माण कार्य तेज",
			Slug:        "bihar-expressway-patna-purnia-greenfield-corridor-construction",
			Category:    "bihar",
			Image:       "https://images.unsplash.com/photo-1545459720-aac8509eb02c?w=800&auto=format&fit=crop&q=80",
			Excerpt:     "सड़क परिवहन मंत्रालय ने परियोजना की प्रगति की समीक्षा की। इस कॉरिडोर से उत्तर और पूर्व बिहार के बीच यात्रा समय आधा हो जाएगा।",
			IsBreaking:  false,
			IsFeatured:  true,
			ViewCount:   14800,
		},
	}

	for _, art := range articlesSeed {
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO articles (
				tenant_id, author_id, title, slug, summary, content,
				featured_image, status, is_breaking, is_featured, is_national,
				language, view_count, published_at
			)
			VALUES (
				1, 1, $1, $2, $3, $3,
				$4, 'published', $5, $6, TRUE,
				'hi', $7, NOW() - (RANDOM() * INTERVAL '12 hours')
			)
			ON CONFLICT (slug) DO UPDATE
			SET title = EXCLUDED.title,
			    featured_image = EXCLUDED.featured_image,
			    status = 'published';

			-- Link article to category in article_categories
			INSERT INTO article_categories (article_id, category_id)
			SELECT a.id, c.id
			FROM articles a
			JOIN categories c ON c.slug = $8
			WHERE a.slug = $2
			ON CONFLICT DO NOTHING;
		`, art.Title, art.Slug, art.Excerpt, art.Image, art.IsBreaking, art.IsFeatured, art.ViewCount, art.Category)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Service Methods: CRUD & Reordering
// ─────────────────────────────────────────────────────────────────────────────

// ListHomepageSections returns all homepage sections ordered by sort_order.
func (s *Service) ListHomepageSections(ctx context.Context) ([]HomepageSection, error) {
	query := `
		SELECT s.id, s.section_key, s.title, s.title_hi, s.section_type,
		       s.category_id, s.category_slug, COALESCE(c.name, '') as category_name,
		       s.sort_order, s.is_active, s.article_limit, s.settings,
		       s.created_at, s.updated_at
		FROM homepage_sections s
		LEFT JOIN categories c ON c.slug = s.category_slug OR c.id = s.category_id
		ORDER BY s.sort_order ASC, s.id ASC
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query homepage sections: %w", err)
	}
	defer rows.Close()

	var sections []HomepageSection
	for rows.Next() {
		var sec HomepageSection
		var settingsBytes []byte
		err := rows.Scan(
			&sec.ID, &sec.SectionKey, &sec.Title, &sec.TitleHi, &sec.SectionType,
			&sec.CategoryID, &sec.CategorySlug, &sec.CategoryName,
			&sec.SortOrder, &sec.IsActive, &sec.ArticleLimit, &settingsBytes,
			&sec.CreatedAt, &sec.UpdatedAt,
		)
		if err != nil {
			continue
		}
		sec.Settings = make(map[string]interface{})
		if len(settingsBytes) > 0 {
			_ = json.Unmarshal(settingsBytes, &sec.Settings)
		}
		sections = append(sections, sec)
	}

	return sections, nil
}

// CreateHomepageSection creates a new dynamic section.
func (s *Service) CreateHomepageSection(ctx context.Context, in CreateSectionInput) (*HomepageSection, error) {
	if in.SectionKey == "" {
		in.SectionKey = fmt.Sprintf("sec_%d", time.Now().UnixNano())
	}
	if in.ArticleLimit <= 0 {
		in.ArticleLimit = 6
	}
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	settingsJSON, _ := json.Marshal(in.Settings)
	if len(settingsJSON) == 0 {
		settingsJSON = []byte("{}")
	}

	// Max current order
	if in.SortOrder <= 0 {
		var maxOrder int
		_ = s.pool.QueryRow(ctx, "SELECT COALESCE(MAX(sort_order), 0) FROM homepage_sections").Scan(&maxOrder)
		in.SortOrder = maxOrder + 1
	}

	query := `
		INSERT INTO homepage_sections (
			section_key, title, title_hi, section_type, category_id,
			category_slug, sort_order, is_active, article_limit, settings
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
		RETURNING id, created_at, updated_at
	`
	var id int
	var createdAt, updatedAt time.Time
	err := s.pool.QueryRow(
		ctx, query,
		in.SectionKey, in.Title, in.TitleHi, in.SectionType, in.CategoryID,
		in.CategorySlug, in.SortOrder, isActive, in.ArticleLimit, settingsJSON,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert homepage section: %w", err)
	}

	// Invalidate home feed cache
	s.InvalidateHomeCache()

	return &HomepageSection{
		ID:           id,
		SectionKey:   in.SectionKey,
		Title:        in.Title,
		TitleHi:      in.TitleHi,
		SectionType:  in.SectionType,
		CategoryID:   in.CategoryID,
		CategorySlug: in.CategorySlug,
		SortOrder:    in.SortOrder,
		IsActive:     isActive,
		ArticleLimit: in.ArticleLimit,
		Settings:     in.Settings,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

// UpdateHomepageSection updates an existing section.
func (s *Service) UpdateHomepageSection(ctx context.Context, id int, in UpdateSectionInput) error {
	query := `
		UPDATE homepage_sections
		SET title = COALESCE($1, title),
		    title_hi = COALESCE($2, title_hi),
		    section_type = COALESCE($3, section_type),
		    category_slug = COALESCE($4, category_slug),
		    sort_order = COALESCE($5, sort_order),
		    is_active = COALESCE($6, is_active),
		    article_limit = COALESCE($7, article_limit),
		    updated_at = NOW()
		WHERE id = $8
	`
	_, err := s.pool.Exec(
		ctx, query,
		in.Title, in.TitleHi, in.SectionType, in.CategorySlug,
		in.SortOrder, in.IsActive, in.ArticleLimit, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update homepage section: %w", err)
	}

	if in.Settings != nil {
		settingsJSON, _ := json.Marshal(in.Settings)
		_, _ = s.pool.Exec(ctx, "UPDATE homepage_sections SET settings = $1::jsonb WHERE id = $2", settingsJSON, id)
	}

	s.InvalidateHomeCache()
	return nil
}

// DeleteHomepageSection removes a section.
func (s *Service) DeleteHomepageSection(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM homepage_sections WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete homepage section: %w", err)
	}
	s.InvalidateHomeCache()
	return nil
}

// ReorderHomepageSections updates sort_order for a given list of section IDs in sequential order.
func (s *Service) ReorderHomepageSections(ctx context.Context, orderedIDs []int) error {
	b := &pgx.Batch{}
	for idx, id := range orderedIDs {
		b.Queue("UPDATE homepage_sections SET sort_order = $1, updated_at = NOW() WHERE id = $2", idx+1, id)
	}
	br := s.pool.SendBatch(ctx, b)
	defer br.Close()

	for range orderedIDs {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to batch update sort order: %w", err)
		}
	}

	s.InvalidateHomeCache()
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Fiber HTTP Handlers
// ─────────────────────────────────────────────────────────────────────────────

// ListHomepageSections handles GET /api/v1/homepage/sections
func (h *Handler) ListHomepageSections(c *fiber.Ctx) error {
	sections, err := h.service.ListHomepageSections(c.Context())
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.Success(c, sections)
}

// CreateHomepageSection handles POST /api/v1/studio/homepage/sections
func (h *Handler) CreateHomepageSection(c *fiber.Ctx) error {
	var input CreateSectionInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request payload")
	}
	if input.Title == "" || input.SectionType == "" {
		return response.BadRequest(c, "title and section_type are required")
	}
	sec, err := h.service.CreateHomepageSection(c.Context(), input)
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.Created(c, sec)
}

// UpdateHomepageSection handles PUT /api/v1/studio/homepage/sections/:id
func (h *Handler) UpdateHomepageSection(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid section ID")
	}
	var input UpdateSectionInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request payload")
	}
	if err := h.service.UpdateHomepageSection(c.Context(), id, input); err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.Success(c, fiber.Map{"message": "Section updated successfully"})
}

// DeleteHomepageSection handles DELETE /api/v1/studio/homepage/sections/:id
func (h *Handler) DeleteHomepageSection(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.BadRequest(c, "Invalid section ID")
	}
	if err := h.service.DeleteHomepageSection(c.Context(), id); err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.Success(c, fiber.Map{"message": "Section deleted successfully"})
}

// ReorderHomepageSections handles POST /api/v1/studio/homepage/sections/reorder
func (h *Handler) ReorderHomepageSections(c *fiber.Ctx) error {
	var input ReorderSectionsInput
	if err := c.BodyParser(&input); err != nil || len(input.OrderedIDs) == 0 {
		return response.BadRequest(c, "ordered_ids array is required")
	}
	if err := h.service.ReorderHomepageSections(c.Context(), input.OrderedIDs); err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.Success(c, fiber.Map{"message": "Sections reordered successfully"})
}
