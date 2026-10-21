-- Risk rules slice 1 (phase-risk-rules.md): per-portfolio thresholds,
-- trade plans, and the plan fields and coach-mode violations on orders.

CREATE TABLE portfolio_risk_settings (
    portfolio_id    text PRIMARY KEY REFERENCES portfolios(id),
    mode            text NOT NULL,              -- coach | strict
    risk_pct        double precision NOT NULL,
    max_risk_pct    double precision NOT NULL,
    min_reward_risk double precision NOT NULL
);

CREATE SEQUENCE trade_plan_id_seq;

CREATE TABLE trade_plans (
    id            text PRIMARY KEY,             -- "tp_1a"
    user_id       text NOT NULL,
    portfolio_id  text NOT NULL,
    symbol        text NOT NULL,
    setup         text NOT NULL DEFAULT '',
    planned_entry double precision NOT NULL,
    initial_stop  double precision NOT NULL,    -- never updated after insert
    current_stop  double precision NOT NULL,
    target        double precision NOT NULL DEFAULT 0,
    opened_at     text NOT NULL,
    closed_at     text NOT NULL DEFAULT '',
    exit_price    double precision NOT NULL DEFAULT 0,
    r_multiple    double precision NOT NULL DEFAULT 0,
    seq           bigserial
);
CREATE INDEX trade_plans_user ON trade_plans (user_id, seq);
-- At most one open plan per portfolio and symbol.
CREATE UNIQUE INDEX trade_plans_one_open ON trade_plans (portfolio_id, symbol) WHERE closed_at = '';

ALTER TABLE orders
    ADD COLUMN stop_loss  double precision NOT NULL DEFAULT 0,
    ADD COLUMN target     double precision NOT NULL DEFAULT 0,
    ADD COLUMN setup      text NOT NULL DEFAULT '',
    ADD COLUMN plan_id    text NOT NULL DEFAULT '',
    ADD COLUMN violations jsonb NOT NULL DEFAULT '[]';
