package pgdb

import (
	"context"
	"errors"

	e "auction-platform/internal/entity"
	rd "auction-platform/internal/repo/dto"
	re "auction-platform/internal/repo/errors"
	errutils "auction-platform/pkg/errors"
	"auction-platform/pkg/postgres"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type AuctionRepo struct {
	*postgres.Postgres
}

func NewAuctionRepo(pg *postgres.Postgres) *AuctionRepo {
	return &AuctionRepo{pg}
}

func (r *AuctionRepo) Create(ctx context.Context, in rd.CreateAuctionInput) (e.Auction, error) {
	sql, args, _ := r.Builder.
		Insert("auctions").
		Columns("auction_id", "title", "description", "seller_id", "start_price", "current_bid", "min_step", "status", "ends_at").
		Values(in.AuctionID, in.Title, in.Description, in.SellerID, in.StartPrice, in.StartPrice, in.MinStep, in.Status, in.EndsAt).
		Suffix("RETURNING auction_id, title, description, seller_id, start_price, current_bid, min_step, status, ends_at, created_at").
		ToSql()

	conn := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool)

	var a e.Auction
	err := conn.QueryRow(ctx, sql, args...).Scan(
		&a.AuctionID, &a.Title, &a.Description, &a.SellerID,
		&a.StartPrice, &a.CurrentBid, &a.MinStep, &a.Status,
		&a.EndsAt, &a.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return e.Auction{}, re.ErrAlreadyExists
		}
		return e.Auction{}, errutils.WrapPathErr(err)
	}
	return a, nil
}

func (r *AuctionRepo) GetByID(ctx context.Context, auctionID string) (e.Auction, error) {
	sql, args, _ := r.Builder.
		Select("auction_id", "title", "description", "seller_id", "start_price",
			"current_bid", "min_step", "status", "COALESCE(winner_id, '') AS winner_id",
			"ends_at", "created_at", "finished_at").
		From("auctions").
		Where("auction_id = ?", auctionID).
		ToSql()

	conn := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool)

	var a e.Auction
	err := conn.QueryRow(ctx, sql, args...).Scan(
		&a.AuctionID, &a.Title, &a.Description, &a.SellerID,
		&a.StartPrice, &a.CurrentBid, &a.MinStep, &a.Status,
		&a.WinnerID, &a.EndsAt, &a.CreatedAt, &a.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return e.Auction{}, re.ErrNotFound
		}
		return e.Auction{}, errutils.WrapPathErr(err)
	}
	return a, nil
}

func (r *AuctionRepo) ListActive(ctx context.Context, limit, offset int) ([]e.Auction, int64, error) {
	conn := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool)

	var total int64
	countSQL, countArgs, _ := r.Builder.
		Select("COUNT(*)").
		From("auctions").
		Where("status = ? AND ends_at > NOW()", e.AuctionStatusActive).
		ToSql()

	if err := conn.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, errutils.WrapPathErr(err)
	}

	sql, args, _ := r.Builder.
		Select("auction_id", "title", "description", "seller_id", "start_price",
			"current_bid", "min_step", "status", "ends_at", "created_at").
		From("auctions").
		Where("status = ? AND ends_at > NOW()", e.AuctionStatusActive).
		OrderBy("ends_at ASC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, errutils.WrapPathErr(err)
	}
	defer rows.Close()

	var auctions []e.Auction
	for rows.Next() {
		var a e.Auction
		if err := rows.Scan(
			&a.AuctionID, &a.Title, &a.Description, &a.SellerID,
			&a.StartPrice, &a.CurrentBid, &a.MinStep, &a.Status,
			&a.EndsAt, &a.CreatedAt,
		); err != nil {
			return nil, 0, errutils.WrapPathErr(err)
		}
		auctions = append(auctions, a)
	}

	return auctions, total, nil
}

func (r *AuctionRepo) UpdateCurrentBid(ctx context.Context, auctionID string, amount float64) error {
	sql, args, _ := r.Builder.
		Update("auctions").
		Set("current_bid", amount).
		Where("auction_id = ?", auctionID).
		ToSql()

	conn := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool)
	cmdTag, err := conn.Exec(ctx, sql, args...)
	if err != nil {
		return errutils.WrapPathErr(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return re.ErrNotFound
	}
	return nil
}

func (r *AuctionRepo) FinishAuction(ctx context.Context, auctionID string, winnerID string, finalPrice float64) error {
	builder := r.Builder.
		Update("auctions").
		Set("status", e.AuctionStatusFinished).
		Set("current_bid", finalPrice).
		Set("finished_at", "NOW()").
		Where("auction_id = ?", auctionID)

	if winnerID != "" {
		builder = builder.Set("winner_id", winnerID)
	}

	sql, args, _ := builder.ToSql()
	conn := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool)
	_, err := conn.Exec(ctx, sql, args...)
	if err != nil {
		return errutils.WrapPathErr(err)
	}
	return nil
}

func (r *AuctionRepo) GetExpired(ctx context.Context) ([]e.Auction, error) {
	sql, args, _ := r.Builder.
		Select("auction_id", "title", "seller_id", "start_price", "current_bid", "min_step", "status", "ends_at").
		From("auctions").
		Where("status = ? AND ends_at <= NOW()", e.AuctionStatusActive).
		ToSql()

	conn := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool)
	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, errutils.WrapPathErr(err)
	}
	defer rows.Close()

	var auctions []e.Auction
	for rows.Next() {
		var a e.Auction
		if err := rows.Scan(
			&a.AuctionID, &a.Title, &a.SellerID, &a.StartPrice,
			&a.CurrentBid, &a.MinStep, &a.Status, &a.EndsAt,
		); err != nil {
			return nil, errutils.WrapPathErr(err)
		}
		auctions = append(auctions, a)
	}
	return auctions, nil
}
