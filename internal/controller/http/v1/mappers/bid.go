package httpmappers

import (
	hd "auction-platform/internal/controller/http/v1/dto"
	e "auction-platform/internal/entity"
	sd "auction-platform/internal/service/dto"
)

func ToPlaceBidServiceInput(in hd.PlaceBidInput) sd.PlaceBidInput {
	return sd.PlaceBidInput{
		BidID:     in.BidID,
		AuctionID: in.AuctionID,
		BidderID:  in.BidderID,
		Amount:    in.Amount,
	}
}

func ToBidDTO(b e.Bid) hd.BidDTO {
	return hd.BidDTO{
		BidID:     b.BidID,
		AuctionID: b.AuctionID,
		BidderID:  b.BidderID,
		Amount:    b.Amount,
		Status:    string(b.Status),
		CreatedAt: b.CreatedAt,
	}
}

func ToBidDTOs(bids []e.Bid) []hd.BidDTO {
	dtos := make([]hd.BidDTO, 0, len(bids))
	for _, b := range bids {
		dtos = append(dtos, ToBidDTO(b))
	}
	return dtos
}
