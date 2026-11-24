# Codebase improvement review

Recorded on 2026-11-25 from the codebase review performed on 2026-11-24.
These are proposed changes; none have been implemented as part of this review.
Prioritize the first three before performance tuning. Performance benefits still
need measurement on representative devices.

## 1. Expose retry headers to cross-origin clients

**Priority:** Medium — correctness for separately hosted frontends.

**Location:** [`internal/httpapi/router.go`](../internal/httpapi/router.go),
[`web/src/leaderboard/client.ts`](../web/src/leaderboard/client.ts).

The CORS configuration allows request headers but does not expose `Retry-After`.
The frontend reads that response header to schedule retries. Cross-origin
browser clients therefore fall back to their own delays, which can be shorter
than the server's requested wait and exhaust retry attempts.

**Proposed improvement:** Add `Retry-After` to CORS `ExposeHeaders`. Also expose
`X-Request-ID` so clients can access the request identifier for diagnostics.
Add coverage for an allowed cross-origin response and its exposed headers.

## 2. Preserve pending score saves across restarts

**Priority:** Medium — potential loss of a completed run's leaderboard result.

**Location:** [`web/src/ui/GameOverOverlay.ts`](../web/src/ui/GameOverOverlay.ts).

The game-over screen permits restarting while a score save is pending, then
aborts that save on scene shutdown. If the request has not reached the server,
or a temporary failure still requires a retry, restarting can abandon the
previous result.

**Proposed improvement:** Let finalization survive scene restarts and display a
pending-save status. Keep the same session credentials and immutable final
score for retries. Add a regression scenario where the player restarts during
a delayed or retrying save.

## 3. Include checkpoint limits in client eviction timing

**Priority:** Medium — correctness under custom rate-limit settings.

**Location:** [`internal/httpapi/ratelimit.go`](../internal/httpapi/ratelimit.go).

Client retention is calculated from the start, finish, and leaderboard buckets,
but excludes the checkpoint bucket. With a checkpoint burst requiring more
than the resulting retention period to refill, eviction recreates the bucket
with a full burst prematurely. Default settings are unaffected.

**Proposed improvement:** Include `config.Checkpoint` in the retention
calculation. Test a checkpoint configuration whose full refill takes longer
than ten minutes, checking that idle eviction cannot restore tokens early.

## 4. Detect and drain expired-session cleanup backlogs

**Priority:** Medium — capacity under sustained abandoned-session traffic.

**Location:** [`internal/maintenance/cleanup.go`](../internal/maintenance/cleanup.go).

With defaults, cleanup removes at most ten batches of 1,000 rows every five
minutes: approximately 33 sessions per second. The global rate limit allows
50 session starts per second across enough clients. Once those sessions age
past expiry and retention, sustained abandoned-session traffic can exceed
cleanup capacity and grow the backlog. The current success timestamp does not
distinguish an empty backlog from a sweep that reached its batch cap.

**Proposed improvement:** Add a backlog or saturation metric and schedule
additional bounded sweeps when the batch cap is reached. Preserve execution
deadlines, `SKIP LOCKED`, and protection for completed sessions. Verify behavior
with more expired rows than a single sweep can process.

## 5. Reduce synchronous high-score storage writes

**Priority:** Low — potential reduction in gameplay frame stalls.

**Location:** [`web/src/systems/ScoreSystem.ts`](../web/src/systems/ScoreSystem.ts).

Every increase to the personal best writes synchronously to `localStorage`.
After a player exceeds their previous record, this includes survival ticks and
each scoring event.

**Proposed improvement:** Update the displayed and in-memory high score
immediately, debounce persistence, and flush on pause or game over. Preserve
graceful handling of unavailable storage. Measure frame timing before and after
on a slower mobile device, and account for the durability tradeoff if the page
closes before a scheduled write.

## 6. Avoid temporary arrays in per-frame laser updates

**Priority:** Low — small allocation optimization, subject to profiling.

**Location:** [`web/src/GameScene.ts`](../web/src/GameScene.ts), `updateLaser`.

While the laser is active, `getMatching('active', true).forEach(...)` creates a
temporary array every frame.

**Proposed improvement:** Iterate the existing children array and check each
child's `active` state directly. Confirm that despawning enemies during
iteration remains safe. Profile allocations and mobile frame times before
expanding this pattern to less frequent operations.

## 7. Offer a rendering quality limit for dense displays

**Priority:** Low — potential GPU savings on constrained devices.

**Location:** [`web/src/utils/display.ts`](../web/src/utils/display.ts),
[`web/src/entities/Starfield.ts`](../web/src/entities/Starfield.ts).

The drawing buffer follows unrestricted device pixel ratio. At DPR 3 it
contains nine times as many pixels as at DPR 1 for the same CSS viewport,
including full-screen background layers.

**Proposed improvement:** Consider an optional quality setting or adaptive
rendering scale, retaining native resolution as the high-quality option.
Measure GPU/frame-time improvements and check text clarity, pointer alignment,
resizing, and changes in display density.

## Validation and limitations

The original review reported:

- `go test ./...`: passed.
- `go vet ./...`: passed.
- Frontend `npm run typecheck`: passed.
- Frontend `npm test`: passed.
- Frontend `npm run build`: produced no further output and was interrupted;
  production build verification remains incomplete.
- PostgreSQL integration tests and browser profiling were not run.

The findings are based on source inspection and the checks above. They do not
establish measured performance regressions. No source changes were made during
the original review.
