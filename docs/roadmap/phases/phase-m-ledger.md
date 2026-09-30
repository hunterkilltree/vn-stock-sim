# Phase M · 3. A safe ledger

Part of phase-m.md.

**Why:**
- Queued buys don't set cash aside, so an overspend is only caught when
  the order fills (phase-k.md "Still open").
- `matchOne` calls `ApplyFill` and then `store.Replace` in two separate
  transactions. Only the in-process `Service.mu` stops a double fill, or
  a cancel racing a fill.
- Store writes that have no error return only log the error
  (phase-persistence.md decision 9).

## Decisions

1. **Reserve without storing a balance.** The order service works out
   what's reserved from the portfolio's queued orders.
   - A queued buy reserves `quantity × price × (1 + fee rate)`, where the
     price is:
     - the limit price, for a limit;
     - the stop price, for a stop;
     - the higher of the two, for an OCO;
     - the day's ceiling (`symbol.Detail.Ceiling`), for an ATC.
   - A queued sell reserves its shares.
   - New orders, market orders included, must fit the *available* cash
     or shares (the total minus what's reserved). Otherwise they fail with
     `insufficient_funds` / `insufficient_shares`.
   - `ApplyFill` still checks real cash at fill time.
   - The portfolio summary gains `availableCash`, shown on Portfolio and
     in the order ticket.
2. **Claim before fill.** The order store gains `Claim(userID, id)`,
   which moves an order from `queued` to `filling` only if it's still
   queued.
   - On Postgres that's `UPDATE … WHERE status='queued' RETURNING`; the
     memory store does the same check under its mutex.
   - Only the caller that wins the claim calls `ApplyFill`, then records
     `filled` or `rejected`.
   - `Cancel` uses the same conditional change, so a cancel and a fill
     can't both succeed.
   - `status` is plain text, so no migration is needed. The UI shows
     `filling` as "Đang khớp".
   - **Known limit:** if a backend crashes between the claim and the
     result, the order stays `filling`. `/healthz` reports `stuckOrders`
     (stuck for more than 5 minutes) and the log names them; they're
     fixed by hand.
3. **Store writes return errors**, which become a 500
   `{code: "internal"}`:
   - order `Append`;
   - watchlist `Add` and `Remove`;
   - backtest and replay `Save`;
   - portfolio `Create` and `AppendEquitySnapshot`.
   Reads still log and return empty results.

## Verify

- **Two backends on one Postgres** (`ORDER_MATCH_INTERVAL=2s`), with 20
  orders that trigger:
  - each order fills once;
  - cash ends at the starting capital minus the fills.
- **A store contract test** races `Claim` against `Cancel` 50 times;
  there's one winner every time.
- **Reserved cash:**
  - Queue buys until available cash is nearly zero; the next queued buy
    and a market buy are both rejected, and cash is unchanged.
  - Cancelling one order frees its reservation.
- **Store errors:** stop the database mid-run; a watchlist add gets a
  500, not a silent 200.
