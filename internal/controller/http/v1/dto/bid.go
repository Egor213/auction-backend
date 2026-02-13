package httpdto

import "time"

type PlaceBidInput struct {
	BidID     string  `json:"bid_id" validate:"required,max=100"`
	AuctionID string  `json:"auction_id" validate:"required,max=100"`
	BidderID  string  `json:"bidder_id" validate:"required,max=100"`
	Amount    float64 `json:"amount" validate:"required,gt=0"`
}

type BidDTO struct {
	BidID     string    `json:"bid_id"`
	AuctionID string    `json:"auction_id"`
	BidderID  string    `json:"bidder_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type PlaceBidOutput struct {
	Bid BidDTO `json:"bid"`
}

type GetBidsInput struct {
	AuctionID string `query:"auction_id" validate:"required,max=100"`
	Limit     int    `query:"limit"`
}

type GetBidsOutput struct {
	Bids []BidDTO `json:"bids"`
}
