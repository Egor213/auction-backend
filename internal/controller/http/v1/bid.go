package httpapi

import (
	"errors"
	"net/http"

	hd "auction-platform/internal/controller/http/v1/dto"
	he "auction-platform/internal/controller/http/v1/errors"
	hmap "auction-platform/internal/controller/http/v1/mappers"
	ut "auction-platform/internal/controller/http/v1/utils"
	"auction-platform/internal/service"
	se "auction-platform/internal/service/errors"

	"github.com/labstack/echo/v4"
)

type bidRoutes struct {
	bidService service.Bids
}

func newBidRoutes(g *echo.Group, bServ service.Bids) {
	r := &bidRoutes{bidService: bServ}

	g.POST("/place", r.placeBid)
	g.GET("/list", r.listByAuction)
}

func (r *bidRoutes) placeBid(c echo.Context) error {
	var input hd.PlaceBidInput
	if err := c.Bind(&input); err != nil {
		return ut.NewErrReasonJSON(c, http.StatusBadRequest, he.ErrCodeInvalidParams, he.ErrInvalidParams.Error())
	}
	if err := c.Validate(input); err != nil {
		return ut.NewErrReasonJSON(c, http.StatusBadRequest, he.ErrCodeInvalidParams, err.Error())
	}

	bid, err := r.bidService.PlaceBid(c.Request().Context(), hmap.ToPlaceBidServiceInput(input))
	if err != nil {
		switch {
		case errors.Is(err, se.ErrNotFoundAuction):
			return ut.NewErrReasonJSON(c, http.StatusNotFound, he.ErrCodeNotFound, err.Error())
		case errors.Is(err, se.ErrBidTooLow):
			return ut.NewErrReasonJSON(c, http.StatusBadRequest, he.ErrCodeBidTooLow, err.Error())
		case errors.Is(err, se.ErrAuctionNotActive):
			return ut.NewErrReasonJSON(c, http.StatusConflict, he.ErrCodeAuctionEnded, err.Error())
		default:
			return ut.NewErrReasonJSON(c, http.StatusInternalServerError, he.ErrCodeInternalServer, he.ErrInternalServer.Error())
		}
	}

	return c.JSON(http.StatusAccepted, hd.PlaceBidOutput{
		Bid: hmap.ToBidDTO(bid),
	})
}

func (r *bidRoutes) listByAuction(c echo.Context) error {
	var input hd.GetBidsInput
	if err := c.Bind(&input); err != nil {
		return ut.NewErrReasonJSON(c, http.StatusBadRequest, he.ErrCodeInvalidParams, he.ErrInvalidParams.Error())
	}
	if err := c.Validate(input); err != nil {
		return ut.NewErrReasonJSON(c, http.StatusBadRequest, he.ErrCodeInvalidParams, err.Error())
	}

	bids, err := r.bidService.GetBidsByAuction(c.Request().Context(), input.AuctionID, input.Limit)
	if err != nil {
		return ut.NewErrReasonJSON(c, http.StatusInternalServerError, he.ErrCodeInternalServer, he.ErrInternalServer.Error())
	}

	return c.JSON(http.StatusOK, hd.GetBidsOutput{
		Bids: hmap.ToBidDTOs(bids),
	})
}
