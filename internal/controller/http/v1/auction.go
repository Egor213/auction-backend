package httpapi

import (
	"errors"
	"math"
	"net/http"

	hd "auction-platform/internal/controller/http/v1/dto"
	he "auction-platform/internal/controller/http/v1/errors"
	hmap "auction-platform/internal/controller/http/v1/mappers"
	ut "auction-platform/internal/controller/http/v1/utils"
	"auction-platform/internal/service"
	se "auction-platform/internal/service/errors"

	"github.com/labstack/echo/v4"
)

type auctionRoutes struct {
	auctionService service.Auctions
}

func newAuctionRoutes(g *echo.Group, aServ service.Auctions) {
	r := &auctionRoutes{auctionService: aServ}

	g.POST("/create", r.create)
	g.GET("/get", r.get)
	g.GET("/list", r.list)
}

func (r *auctionRoutes) create(c echo.Context) error {
	var input hd.CreateAuctionInput
	if err := c.Bind(&input); err != nil {
		return ut.NewErrReasonJSON(c, http.StatusBadRequest, he.ErrCodeInvalidParams, he.ErrInvalidParams.Error())
	}
	if err := c.Validate(input); err != nil {
		return ut.NewErrReasonJSON(c, http.StatusBadRequest, he.ErrCodeInvalidParams, err.Error())
	}

	auction, err := r.auctionService.CreateAuction(c.Request().Context(), hmap.ToCreateAuctionServiceInput(input))
	if err != nil {
		if errors.Is(err, se.ErrAuctionAlreadyExists) {
			return ut.NewErrReasonJSON(c, http.StatusConflict, he.ErrCodeAlreadyExists, err.Error())
		}
		return ut.NewErrReasonJSON(c, http.StatusInternalServerError, he.ErrCodeInternalServer, he.ErrInternalServer.Error())
	}

	return c.JSON(http.StatusCreated, hd.CreateAuctionOutput{
		Auction: hmap.ToAuctionDTO(auction),
	})
}

func (r *auctionRoutes) get(c echo.Context) error {
	var input hd.GetAuctionInput
	if err := c.Bind(&input); err != nil {
		return ut.NewErrReasonJSON(c, http.StatusBadRequest, he.ErrCodeInvalidParams, he.ErrInvalidParams.Error())
	}
	if err := c.Validate(input); err != nil {
		return ut.NewErrReasonJSON(c, http.StatusBadRequest, he.ErrCodeInvalidParams, err.Error())
	}

	auction, err := r.auctionService.GetAuction(c.Request().Context(), input.AuctionID)
	if err != nil {
		if errors.Is(err, se.ErrNotFoundAuction) {
			return ut.NewErrReasonJSON(c, http.StatusNotFound, he.ErrCodeNotFound, he.ErrNotFound.Error())
		}
		return ut.NewErrReasonJSON(c, http.StatusInternalServerError, he.ErrCodeInternalServer, he.ErrInternalServer.Error())
	}

	return c.JSON(http.StatusOK, hd.GetAuctionOutput{
		Auction: hmap.ToAuctionDTO(auction),
	})
}

func (r *auctionRoutes) list(c echo.Context) error {
	var input hd.ListAuctionsInput
	if err := c.Bind(&input); err != nil {
		input.Page = 1
		input.PageSize = 20
	}
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	auctions, total, err := r.auctionService.ListActive(c.Request().Context(), input.Page, input.PageSize)
	if err != nil {
		return ut.NewErrReasonJSON(c, http.StatusInternalServerError, he.ErrCodeInternalServer, he.ErrInternalServer.Error())
	}

	if auctions == nil {
		auctions = nil
	}

	return c.JSON(http.StatusOK, hd.ListAuctionsOutput{
		Auctions:   hmap.ToAuctionDTOs(auctions),
		Total:      total,
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: int(math.Ceil(float64(total) / float64(input.PageSize))),
	})
}
