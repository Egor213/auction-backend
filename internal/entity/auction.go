package entity

import "time"

type AuctionStatus string

const (
	AuctionStatusActive   AuctionStatus = "ACTIVE"
	AuctionStatusFinished AuctionStatus = "FINISHED"
)

type Auction struct {
	CreatedAt   *time.Time    `db:"created_at"`
	EndsAt      *time.Time    `db:"ends_at"`
	FinishedAt  *time.Time    `db:"finished_at"`
	AuctionID   string        `db:"auction_id"`
	Title       string        `db:"title"`
	Description string        `db:"description"`
	SellerID    string        `db:"seller_id"`
	WinnerID    string        `db:"winner_id"`
	StartPrice  float64       `db:"start_price"`
	CurrentBid  float64       `db:"current_bid"`
	MinStep     float64       `db:"min_step"`
	Status      AuctionStatus `db:"status"`
}
