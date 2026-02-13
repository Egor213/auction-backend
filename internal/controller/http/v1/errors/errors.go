package httperrs

import "errors"

type ErrorCode string

const (
	ErrCodeInvalidParams  ErrorCode = "INVALID_REQUEST_PARAMETERS"
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists  ErrorCode = "ALREADY_EXISTS"
	ErrCodeInternalServer ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrCodeRateLimited    ErrorCode = "RATE_LIMITED"
	ErrCodeAuctionEnded   ErrorCode = "AUCTION_ENDED"
	ErrCodeBidTooLow      ErrorCode = "BID_TOO_LOW"
)

var (
	ErrInvalidParams  = errors.New("invalid request parameters")
	ErrNotFound       = errors.New("resource not found")
	ErrAlreadyExists  = errors.New("resource already exists")
	ErrInternalServer = errors.New("internal server error")
)
