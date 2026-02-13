package servdto

type CreateAuctionInput struct {
	AuctionID   string
	Title       string
	Description string
	SellerID    string
	StartPrice  float64
	MinStep     float64
	DurationMin int
}
