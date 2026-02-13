CREATE TABLE IF NOT EXISTS auctions (
    auction_id VARCHAR(100) PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    description TEXT DEFAULT '',
    seller_id VARCHAR(100) NOT NULL,
    start_price DECIMAL(12,2) NOT NULL CHECK (start_price > 0),
    current_bid DECIMAL(12,2) NOT NULL,
    min_step DECIMAL(12,2) NOT NULL CHECK (min_step > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    winner_id VARCHAR(100),
    ends_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS bids (
    bid_id VARCHAR(100) PRIMARY KEY,
    auction_id VARCHAR(100) NOT NULL REFERENCES auctions(auction_id),
    bidder_id VARCHAR(100) NOT NULL,
    amount DECIMAL(12,2) NOT NULL CHECK (amount > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auctions_status_ends ON auctions(status, ends_at);
CREATE INDEX idx_bids_auction_id ON bids(auction_id);