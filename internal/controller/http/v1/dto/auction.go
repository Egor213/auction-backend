package httpdto

import "time"

type CreateAuctionInput struct {
	AuctionID   string  `json:"auction_id" validate:"required,max=100"`
	Title       string  `json:"title" validate:"required,max=200"`
	Description string  `json:"description" validate:"max=2000"`
	SellerID    string  `json:"seller_id" validate:"required,max=100"`
	StartPrice  float64 `json:"start_price" validate:"required,gt=0"`
	MinStep     float64 `json:"min_step" validate:"required,gt=0"`
	DurationMin int     `json:"duration_min" validate:"required,min=1,max=10080"`
}

type AuctionDTO struct {
	AuctionID   string     `json:"auction_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	SellerID    string     `json:"seller_id"`
	StartPrice  float64    `json:"start_price"`
	CurrentBid  float64    `json:"current_bid"`
	MinStep     float64    `json:"min_step"`
	Status      string     `json:"status"`
	WinnerID    string     `json:"winner_id,omitempty"`
	EndsAt      *time.Time `json:"ends_at"`
	CreatedAt   *time.Time `json:"created_at"`
}

type CreateAuctionOutput struct {
	Auction AuctionDTO `json:"auction"`
}

type GetAuctionInput struct {
	AuctionID string `query:"auction_id" validate:"required,max=100"`
}

type GetAuctionOutput struct {
	Auction AuctionDTO `json:"auction"`
}

type ListAuctionsInput struct {
	Page     int `query:"page"`
	PageSize int `query:"page_size"`
}

type ListAuctionsOutput struct {
	Auctions   []AuctionDTO `json:"auctions"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
}
