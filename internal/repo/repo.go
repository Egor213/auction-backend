package repo

import (
	"auction-platform/internal/repo/pgdb"
	"auction-platform/pkg/postgres"
	"context"

	e "auction-platform/internal/entity"
	rd "auction-platform/internal/repo/dto"
)

type Auctions interface {
	Create(ctx context.Context, in rd.CreateAuctionInput) (e.Auction, error)
	GetByID(ctx context.Context, auctionID string) (e.Auction, error)
	ListActive(ctx context.Context, limit, offset int) ([]e.Auction, int64, error)
	UpdateCurrentBid(ctx context.Context, auctionID string, amount float64) error
	FinishAuction(ctx context.Context, auctionID string, winnerID string, finalPrice float64) error
	GetExpired(ctx context.Context) ([]e.Auction, error)
}

type Bids interface {
	Create(ctx context.Context, in rd.CreateBidInput) (e.Bid, error)
	UpdateStatus(ctx context.Context, bidID string, status e.BidStatus) error
	GetHighestByAuction(ctx context.Context, auctionID string) (e.Bid, error)
	ListByAuction(ctx context.Context, auctionID string, limit int) ([]e.Bid, error)
	CountByAuction(ctx context.Context, auctionID string) (int, error)
}

type Repositories struct {
	Auctions
	Bids
}

func NewRepositories(pg *postgres.Postgres) *Repositories {
	return &Repositories{
		Auctions: pgdb.NewAuctionRepo(pg),
		Bids:     pgdb.NewBidRepo(pg),
	}
}
