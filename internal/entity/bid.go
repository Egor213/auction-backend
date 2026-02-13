package entity

import "time"

type BidStatus string

const (
	BidStatusPending  BidStatus = "PENDING"
	BidStatusAccepted BidStatus = "ACCEPTED"
	BidStatusRejected BidStatus = "REJECTED"
)

type Bid struct {
	CreatedAt time.Time `db:"created_at"`
	BidID     string    `db:"bid_id"`
	AuctionID string    `db:"auction_id"`
	BidderID  string    `db:"bidder_id"`
	Amount    float64   `db:"amount"`
	Status    BidStatus `db:"status"`
}
