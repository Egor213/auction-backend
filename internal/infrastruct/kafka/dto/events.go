package kafkadto

import "time"

type BidPlacedEvent struct {
	BidID     string    `json:"bid_id"`
	AuctionID string    `json:"auction_id"`
	BidderID  string    `json:"bidder_id"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

type BidResultEvent struct {
	BidID     string  `json:"bid_id"`
	AuctionID string  `json:"auction_id"`
	BidderID  string  `json:"bidder_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	Reason    string  `json:"reason,omitempty"`
}

type AuctionEndedEvent struct {
	AuctionID  string  `json:"auction_id"`
	WinnerID   string  `json:"winner_id,omitempty"`
	FinalPrice float64 `json:"final_price"`
	TotalBids  int     `json:"total_bids"`
}
