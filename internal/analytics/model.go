package analytics

type AnalyticsOverview struct {
	TotalArticles       int64                `json:"total_articles"`
	TotalPublished      int64                `json:"total_published"`
	TotalViews          int64                `json:"total_views"`
	TotalBreaking       int64                `json:"total_breaking"`
	TotalSubscribers    int64                `json:"total_subscribers"`
	StateDistribution   []RegionalReadership `json:"state_distribution"`
	CategoryBreakdown   []CategoryReadership `json:"category_breakdown"`
	TopTrendingArticles []TrendingStat       `json:"top_trending_articles"`
}

type RegionalReadership struct {
	RegionID   int    `json:"region_id"`
	RegionName string `json:"region_name"`
	Views      int64  `json:"views"`
	Articles   int64  `json:"articles"`
}

// Backward compatibility alias
type TenantReadership = RegionalReadership

type CategoryReadership struct {
	CategoryName string `json:"category_name"`
	Count        int64  `json:"count"`
}

type TrendingStat struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	ViewCount int64  `json:"view_count"`
	Language  string `json:"language"`
}

type AuthorLeaderboard struct {
	AuthorID    int64  `json:"author_id"`
	DisplayName string `json:"display_name"`
	BureauName  string `json:"bureau_name"`
	TotalViews  int64  `json:"total_views"`
	Articles    int64  `json:"articles"`
}
