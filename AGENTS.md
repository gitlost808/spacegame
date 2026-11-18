# Working in this repository

- The Go API lives in `cmd/server` and `internal`; the Phaser/TypeScript game lives in `web/src`.
- Read `README.md` for the session protocol and local setup, `web/README.md` for the frontend, and `docs/container.md` for deployment.
- Keep session authentication, timing, score validation, and finalization authoritative on the server. Preserve generic rejection responses, immutable completed scores, and protection against concurrent requests. Add regression coverage when changing these rules.
- Store schema changes in a new numbered Goose migration with Up and Down sections. Edit `internal/store/queries.sql`, then run `make generate`; do not hand-edit sqlc output. Update readiness checks when required columns change.
- For Go changes, run `go test ./...` and `go vet ./...`. For database behavior, also run `make test-integration` against a disposable PostgreSQL database using `DATABASE_URL`; integration tests create isolated schemas.
- For frontend changes, run `npm ci` in `web` when dependencies are missing, then `npm run typecheck`, `npm test`, and `npm run build` there.
- Keep protocol and configuration documentation in sync with behavior. Do not commit credentials, `.env`, build output, or dependency directories.
- After making repository changes, commit them, push a branch, and open a pull request by default unless the user requests otherwise. If publishing is blocked, report the blocker and preserve the completed work.
