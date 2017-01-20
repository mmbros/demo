package main

type StockPriceSource struct {
	ScraperName string
	URL         string
	Disabled    bool
}

type Stock struct {
	ID       string
	Isin     string
	Name     string
	Disabled bool
	Sources  []StockPriceSource

	// enabled is set to true if:
	// - stock is enabled
	// - stock has at least one enabled source whose scraper is enabled
	enabled bool
}

type Stocks map[string]*Stock
