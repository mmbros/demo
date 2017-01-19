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
}

type Stocks map[string]*Stock
