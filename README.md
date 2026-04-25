# slot-art

Slot art image generation service.

## Development checks

The repository is moving to a Buf + gRPC + grpc-gateway API contract. Generated
Go gateway code and OpenAPI output are committed so CI can verify that generated
artifacts stay in sync with proto changes.
OpenAPI v3 output is generated with grpc-gateway's local
`protoc-gen-openapiv3` plugin, pinned in the Makefile/CI setup until the plugin
is available in a stable grpc-gateway release.

Useful local commands:

- `make proto-lint` — run `buf lint`.
- `make proto-generate` — regenerate protobuf, gRPC, gateway and OpenAPI files.
- `make ci` — run the local validation sequence: proto checks, frontend lint/typecheck/build, and Go tests.
- `make build` — build the embedded frontend and server binary.

CI runs three gates on every pull request and push to `main`:

1. Proto contract: `buf lint`, `buf generate`, and a generated-file diff check.
2. Frontend: `pnpm install --frozen-lockfile`, `pnpm lint`, `pnpm typecheck`, `pnpm build`.
3. Backend: downloads the frontend build artifact, verifies `go mod tidy`, runs `go test ./...`, and builds the server.

## Current refactor direction

- Public API is being consolidated into `slot.v1.SlotArtService`.
- Admin API is being consolidated into `slot.v1.AdminService`.
- Credits are granted by Admin-generated CDKeys and redeemed by anonymous users.
- Provider integration is being reduced to OpenAI Image2 only.
- Legacy API/storage compatibility is intentionally out of scope.
