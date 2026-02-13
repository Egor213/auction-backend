package servmappers

import (
	e "auction-platform/internal/entity"
	rd "auction-platform/internal/repo/dto"
	sd "auction-platform/internal/service/dto"
	"fmt"
	"time"
)

func ToCreateAuctionRepoInput(in sd.CreateAuctionInput) rd.CreateAuctionInput {
	endsAt := time.Now().UTC().Add(time.Duration(in.DurationMin) * time.Minute)
	return rd.CreateAuctionInput{
		AuctionID:   in.AuctionID,
		Title:       in.Title,
		Description: in.Description,
		SellerID:    in.SellerID,
		StartPrice:  in.StartPrice,
		MinStep:     in.MinStep,
		Status:      e.AuctionStatusActive,
		EndsAt:      fmt.Sprintf("%s", endsAt.Format(time.RFC3339)),
	}
}
