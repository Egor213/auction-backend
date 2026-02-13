package repodto

import e "auction-platform/internal/entity"

type CreateAuctionInput struct {
	AuctionID   string
	Title       string
	Description string
	SellerID    string
	StartPrice  float64
	MinStep     float64
	Status      e.AuctionStatus
	EndsAt      string
}
