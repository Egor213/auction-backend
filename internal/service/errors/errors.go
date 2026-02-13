package serverrs

import "errors"

var (
	ErrNotFoundAuction = errors.New("auction not found")
	ErrNotFoundBid     = errors.New("bid not found")

	ErrCannotCreateAuction = errors.New("cannot create auction")
	ErrCannotGetAuction    = errors.New("cannot get auction")
	ErrCannotListAuctions  = errors.New("cannot list auctions")
	ErrCannotUpdateBid     = errors.New("cannot update bid")
	ErrCannotCreateBid     = errors.New("cannot create bid")
	ErrCannotGetBids       = errors.New("cannot get bids")
	ErrCannotPublishEvent  = errors.New("cannot publish event")

	ErrAuctionAlreadyExists = errors.New("auction already exists")
	ErrAuctionNotActive     = errors.New("auction is not active")
	ErrAuctionEnded         = errors.New("auction has ended")
	ErrBidTooLow            = errors.New("bid is too low")
	ErrSellerCannotBid      = errors.New("seller cannot bid on own auction")
)
