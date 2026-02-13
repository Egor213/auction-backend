package servmappers

import (
	e "auction-platform/internal/entity"
	rd "auction-platform/internal/repo/dto"
	sd "auction-platform/internal/service/dto"
)

func ToCreateBidRepoInput(in sd.PlaceBidInput) rd.CreateBidInput {
	return rd.CreateBidInput{
		BidID:     in.BidID,
		AuctionID: in.AuctionID,
		BidderID:  in.BidderID,
		Amount:    in.Amount,
		Status:    e.BidStatusPending,
	}
}
