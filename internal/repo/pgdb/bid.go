package pgdb

import (
	"context"
	"errors"

	e "auction-platform/internal/entity"
	rd "auction-platform/internal/repo/dto"
	re "auction-platform/internal/repo/errors"
	errutils "auction-platform/pkg/errors"
	"auction-platform/pkg/postgres"

	"github.com/jackc/pgx/v5"
)

type BidRepo struct {
	*postgres.Postgres
}

func NewBidRepo(pg *postgres.Postgres) *BidRepo {
	return &BidRepo{pg}
}

func (r *BidRepo) Create(ctx context.Context, in rd.CreateBidInput) (e.Bid, error) {
	sql, args, _ := r.Builder.
		Insert("bids").
		Columns("bid_id", "auction_id", "bidder_id", "amount", "status").
		Values(in.BidID, in.AuctionID, in.BidderID, in.Amount, in.Status).
		Suffix("RETURNING bid_id, auction_id, bidder_id, amount, status, created_at").
		ToSql()

	conn := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool)

	var b e.Bid
	err := conn.QueryRow(ctx, sql, args...).Scan(
		&b.BidID, &b.AuctionID, &b.BidderID, &b.Amount, &b.Status, &b.CreatedAt,
	)
	if err != nil {
		return e.Bid{}, errutils.WrapPathErr(err)
	}
	return b, nil
}

func (r *BidRepo) UpdateStatus(ctx context.Context, bidID string, status e.BidStatus) error {
	sql, args, _ := r.Builder.
		Update("bids").
		Set("status", status).
		Where("bid_id = ?", bidID).
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

func (r *BidRepo) GetHighestByAuction(ctx context.Context, auctionID string) (e.Bid, error) {
	sql, args, _ := r.Builder.
		Select("bid_id", "auction_id", "bidder_id", "amount", "status", "created_at").
		From("bids").
		Where("auction_id = ? AND status IN (?, ?)", auctionID, e.BidStatusAccepted, e.BidStatusPending).
		OrderBy("amount DESC").
		Limit(1).
		ToSql()

	conn := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool)

	var b e.Bid
	err := conn.QueryRow(ctx, sql, args...).Scan(
		&b.BidID, &b.AuctionID, &b.BidderID, &b.Amount, &b.Status, &b.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return e.Bid{}, re.ErrNotFound
		}
		return e.Bid{}, errutils.WrapPathErr(err)
	}
	return b, nil
}

func (r *BidRepo) ListByAuction(ctx context.Context, auctionID string, limit int) ([]e.Bid, error) {
	sql, args, _ := r.Builder.
		Select("bid_id", "auction_id", "bidder_id", "amount", "status", "created_at").
		From("bids").
		Where("auction_id = ?", auctionID).
		OrderBy("amount DESC").
		Limit(uint64(limit)).
		ToSql()

	conn := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool)
	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, errutils.WrapPathErr(err)
	}
	defer rows.Close()

	var bids []e.Bid
	for rows.Next() {
		var b e.Bid
		if err := rows.Scan(&b.BidID, &b.AuctionID, &b.BidderID, &b.Amount, &b.Status, &b.CreatedAt); err != nil {
			return nil, errutils.WrapPathErr(err)
		}
		bids = append(bids, b)
	}
	return bids, nil
}

func (r *BidRepo) CountByAuction(ctx context.Context, auctionID string) (int, error) {
	sql, args, _ := r.Builder.
		Select("COUNT(*)").
		From("bids").
		Where("auction_id = ?", auctionID).
		ToSql()

	conn := r.CtxGetter.DefaultTrOrDB(ctx, r.Pool)
	var count int
	err := conn.QueryRow(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, errutils.WrapPathErr(err)
	}
	return count, nil
}
