# Integration testing

Unit tests (`go test ./...`) run entirely against `httptest` mocks. The live
integration suite is opt-in and exercises a real domain end to end.

## Configure a fresh clone

From the repository root:

```bash
cd v2
cp .env.integration.example .env.integration
chmod 600 .env.integration
```

Edit `.env.integration` and set `EGNYTE_DOMAIN` plus one authentication
method. The filled file is ignored by Git; never commit it.

If you already have a cached OAuth access token, provide it directly:

```bash
export EGNYTE_DOMAIN=acme
export EGNYTE_TOKEN=xxxxxxxx
```

The access token needs the `Egnyte.filesystem`, `Egnyte.link`, and
`Egnyte.permission` scopes for the full suite. An unscoped internal-application
token also works. `Egnyte.ai` enables the optional hybrid-search check.

You do not need to obtain an access token manually. Leave `EGNYTE_TOKEN` empty
and provide your internal application's API key (OAuth client ID), client
secret, username, and password. The suite calls `Client.RequestToken`, keeps the
returned access token in memory, and reuses it for the entire test run:

```bash
export EGNYTE_DOMAIN=acme
export EGNYTE_KEY=your_api_key
export EGNYTE_SECRET=your_api_secret
export EGNYTE_USERNAME=admin
export EGNYTE_PASSWORD='...'
export EGNYTE_SCOPES='Egnyte.filesystem Egnyte.link Egnyte.permission Egnyte.ai'
```

Load the completed file into the current shell without printing its contents,
then run the suite:

```bash
set -a
. ./.env.integration
set +a
go test -tags integration -run TestIntegration -v .
```

To omit the approximately 11 MB chunked-upload scenario during routine local
development, add `-short`:

```bash
go test -short -tags integration -run TestIntegration -v .
```

The API key must be an internal-application key registered for the target
domain at developers.egnyte.com. Password-flow tokens are limited to 10 token
requests per user per hour, and the suite mints at most one per run. Optional
API families that are unavailable or outside the token's scopes are skipped.
See [Authentication](authentication.md) for the SDK's supported OAuth flows and
the [Egnyte getting-started guide](https://developers.egnyte.com/integration/cfs/api-docs/getting-started)
for application registration and access-token details.

## What it does

1. `Tokens.UserInfo` — authentication check
2. Creates `/Shared/sdk-integration-<timestamp>` and works only inside it
3. Uploads a file whose name contains `$` and `?` (exercises strict path
   encoding), downloads it back, compares bytes
4. Lists the folder; logs live rate-limit quota
5. Adds, lists and deletes a comment
6. Creates and deletes a **domain-restricted** share link (no emails)
7. Reads the events cursor
8. Cleanup deletes the scratch folder — contents go to the trash, so
   nothing is unrecoverable

Calls are paced ~0.6 s apart to stay under the 2 calls/second per-token
limit. Prefer a test/sandbox domain over production. Additional suites
cover a read-only sweep across API families (features your domain lacks
are skipped, not failed), a file lifecycle (versions, copy/move/rename,
folder options, metadata), live-contract checks (file vs folder
metadata shapes, by-ID lookups, folder stats, lock/unlock, effective
permissions, hybrid search), and an ~11 MB chunked upload (skipped in
`-short` mode).

## For external contributors

You need your own Egnyte domain — a developer trial domain from
[developers.egnyte.com](https://developers.egnyte.com) works — plus an
internal-application API key registered for it. The tests only ever
write inside self-cleaning `sdk-integration-*` scratch folders and never
permanently delete anything (cleanup goes to the trash).

## CI note

If continuous integration is ever set up for this repository, it should
only compile-check the integration build tag (`go vet -tags integration`)
rather than execute these tests — and if integration runs with stored
credentials are ever added, they must not be triggered by pull requests
from forks.
