package httpmappers

import (
	hd "auction-platform/internal/controller/http/v1/dto"
	e "auction-platform/internal/entity"
	sd "auction-platform/internal/service/dto"
)

func ToCreateAuctionServiceInput(in hd.CreateAuctionInput) sd.CreateAuctionInput {
	return sd.CreateAuctionInput{
		AuctionID:   in.AuctionID,
		Title:       in.Title,
		Description: in.Description,
		SellerID:    in.SellerID,
		StartPrice:  in.StartPrice,
		MinStep:     in.MinStep,
		DurationMin: in.DurationMin,
	}
}

func ToAuctionDTO(a e.Auction) hd.AuctionDTO {
	return hd.AuctionDTO{
		AuctionID:   a.AuctionID,
		Title:       a.Title,
		Description: a.Description,
		SellerID:    a.SellerID,
		StartPrice:  a.StartPrice,
		CurrentBid:  a.CurrentBid,
		MinStep:     a.MinStep,
		Status:      string(a.Status),
		WinnerID:    a.WinnerID,
		EndsAt:      a.EndsAt,
		CreatedAt:   a.CreatedAt,
	}
}

func ToAuctionDTOs(auctions []e.Auction) []hd.AuctionDTO {
	dtos := make([]hd.AuctionDTO, 0, len(auctions))
	for _, a := range auctions {
		dtos = append(dtos, ToAuctionDTO(a))
	}
	return dtos
}
