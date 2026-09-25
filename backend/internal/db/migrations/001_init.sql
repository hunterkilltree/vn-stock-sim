-- VN Stock Sim schema v1 (phase-persistence.md decision 6). Every table
-- mirrors one feature's in-memory store; IDs keep their prefixes and come
-- from the sequences below.

CREATE SEQUENCE user_id_seq;
CREATE SEQUENCE portfolio_id_seq;
CREATE SEQUENCE order_id_seq;
CREATE SEQUENCE backtest_id_seq;
CREATE SEQUENCE replay_id_seq;

CREATE TABLE users (
    id              text PRIMARY KEY,           -- "u12"
    email           text NOT NULL UNIQUE,
    display_name    text NOT NULL,
    market_interest text NOT NULL DEFAULT '',
    password_hash   bytea NOT NULL,             -- bcrypt
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE portfolios (
    id               text PRIMARY KEY,          -- "pf_3"
    user_id          text NOT NULL REFERENCES users(id),
    name             text NOT NULL,
    market           text NOT NULL,             -- stock | crypto
    kind             text NOT NULL,             -- trading | replay
    starting_capital double precision NOT NULL,
    currency         text NOT NULL,
    created_at       text NOT NULL,             -- RFC 3339, as served
    cash             double precision NOT NULL,
    is_default       boolean NOT NULL DEFAULT false,
    seq              bigserial
);
CREATE INDEX portfolios_user ON portfolios (user_id, seq);
-- One default (stock trading) portfolio per user.
CREATE UNIQUE INDEX portfolios_one_default ON portfolios (user_id) WHERE is_default;

CREATE TABLE positions (
    portfolio_id text NOT NULL REFERENCES portfolios(id),
    symbol       text NOT NULL,
    quantity     double precision NOT NULL,
    avg_cost     double precision NOT NULL,
    PRIMARY KEY (portfolio_id, symbol)
);

CREATE TABLE equity_points (
    id           bigserial PRIMARY KEY,
    portfolio_id text NOT NULL REFERENCES portfolios(id),
    ts           text NOT NULL,
    nav          double precision NOT NULL
);
CREATE INDEX equity_points_portfolio ON equity_points (portfolio_id, id);

CREATE TABLE orders (
    id            text PRIMARY KEY,             -- "ord_1f"
    user_id       text NOT NULL,
    portfolio_id  text NOT NULL,
    symbol        text NOT NULL,
    side          text NOT NULL,
    type          text NOT NULL,
    quantity      double precision NOT NULL,
    status        text NOT NULL,
    price         double precision NOT NULL DEFAULT 0,
    stop_price    double precision NOT NULL DEFAULT 0,
    filled_price  double precision NOT NULL DEFAULT 0,
    fee           double precision NOT NULL DEFAULT 0,
    filled_at     text NOT NULL DEFAULT '',
    created_at    text NOT NULL,
    triggered_by  text NOT NULL DEFAULT '',
    reject_reason text NOT NULL DEFAULT '',
    seq           bigserial
);
CREATE INDEX orders_user ON orders (user_id, seq);
-- The matcher's scan (phase-k.md decision 9).
CREATE INDEX orders_queued ON orders (created_at) WHERE status = 'queued';

CREATE TABLE watchlist_items (
    user_id  text NOT NULL,
    symbol   text NOT NULL,
    added_at timestamptz NOT NULL DEFAULT now(),
    seq      bigserial,
    PRIMARY KEY (user_id, symbol)
);

CREATE TABLE backtests (
    id         text PRIMARY KEY,                -- "bt_a"
    user_id    text NOT NULL,
    body       jsonb NOT NULL,                  -- the whole Backtest, equity and trades included
    seq        bigserial
);
CREATE INDEX backtests_user ON backtests (user_id, seq);

CREATE TABLE replay_sessions (
    id         text PRIMARY KEY,                -- "rp_4"
    user_id    text NOT NULL,
    body       jsonb NOT NULL,                  -- the whole Session, bars included
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX replay_sessions_user ON replay_sessions (user_id);
