---
name: verify-key-rotation
description: Model-check and database-test Shield's project key rotation (encryption key, API key and API secret). Use after any change to the encryption key builders, reconstruction strategies, share persistence (sharerepo Create/Update/BulkUpdate/UpdateProjectEncryption), shamirjob, projectapp/projectrepo rotation code, projectsvc API-key or API-secret rotation or cache, the project authenticator, or the rotate-* routes and middleware.
---

# Verify key rotation

Two promises protect wallets here:

1. **No undecryptable share.** Every stored project-entropy share decrypts
   under the project's current key. Breaking it loses wallets.
2. **No stranded client.** A client that keeps what Shield sent it can always
   reach the current encryption key and authenticate with the current API key
   and API secret, even when responses are lost. Breaking it locks a project out.

`formal/rotation/Rotation.tla` checks both over every interleaving of share
writers, the shamir job, stale parts and sessions, rotations, expiry and lost
responses. The database tests check the Go code against real Postgres.

## Does my change touch the verified code?

It does if it changes any function in the coverage record below, or the order
of the database statements inside them.

## Run

```bash
# Model: baseline must pass, every mutant must violate its invariant.
JAVA=/opt/homebrew/opt/openjdk@17/bin/java formal/rotation/run.sh

# Database tests (Postgres 16 in Docker).
docker run -d --rm --name shield-test-pg -e POSTGRES_PASSWORD=password -e POSTGRES_DB=shield -p 55432:5432 postgres:16
export DB_HOST=localhost DB_PORT=55432 DB_USER=postgres DB_PASS=password DB_NAME=shield DB_SSL_MODE=disable
go run cmd/main.go db migrate
go test -race -count=1 ./internal/adapters/repositories/sql/... ./internal/applications/apptest/ ./di/ ./internal/core/services/projectsvc/
docker stop shield-test-pg
```

`run.sh` downloads `tla2tools.jar` v1.7.4 and verifies its SHA-256. It takes
about 3 minutes. Expected output: one `PASS baseline` line with the final
state count and `0 states left on queue`, and eight `PASS Mut... violates ...` lines.

## Reading a failure

- **Baseline fails with a trace:** the design admits a bad interleaving. Read
  the trace as steps in the code's words (WriterRead = a share write
  reconstructed its key; RotCommit = a rotation commit; and so on, see the
  record below). Fix the design or the code. Never weaken an invariant or the
  model to make it pass.
- **A mutant passes:** the model no longer detects that bug. Usually the
  model drifted from the code; fix the model.
- **A database test fails:** the Go code broke the promise the model assumes.
  `TestPartRowLockSemantics` failing means Postgres no longer behaves as the
  guard assumes; the whole design depends on it.

## Coverage record

| Model action / property | Code |
| --- | --- |
| `KeyAccepted` (key check) | `plain_builder.Build`, `session_builder.Build`, `cypher.VerifyKeyCheck` |
| `WriterRead` | `shareapp.reconstructEncryptionKey` (callers `RegisterShare`, `UpdateShare`) |
| `WriterPersist`, `GuardPasses` | `sharerepo.underGuard`, `sharerepo.checkGuard`; `Create`, `Update`, `UpdateProjectEncryption` |
| `ShamirRead` / `ShamirPersist` | `shamirjob.Execute`, `sharerepo.BulkUpdate` |
| `RotPrepare` (incl. legacy backfill) | `projectapp.PrepareEncryptionKeyRotation`, `projectapp.verifiedCurrentKey` (refuses legacy keys with no share), `projectrepo.PrepareEncryptionKeyRotation` |
| `RotCommit` | `projectapp.CommitEncryptionKeyRotation`, `projectrepo.CommitEncryptionKeyRotation`, `projectrepo.lockRotationState` |
| `PendingExpires` | `rotationPendingTTL`, `pending_expires_at` |
| `ApiPrepare` | `projectsvc.PrepareAPIKeyRotation`, `projectrepo.SetPendingAPIKey` |
| `ApiCommit` | `authmdw.AuthenticateAPIKeyRotationCommit`, `projauth.NewPendingProjectAuthenticator`, `projectsvc.CommitAPIKeyRotation`, `projectrepo.CommitAPIKeyRotation` |
| `SecPrepare` | `projectsvc.PrepareAPISecretRotation`, `projectrepo.SetPendingAPISecret` |
| `SecCommit` | `authmdw.AuthenticateAPISecretRotationCommit`, `projauth.NewPendingSecretProjectAuthenticator`, `projectsvc.CommitAPISecretRotation`, `projectrepo.CommitAPISecretRotation`, `projectrepo.lockProject` |
| `SecPendingExpires` | `rotationPendingTTL`, `pending_api_secret_expires_at` |
| Not modeled (DB tests instead) | share entropy changes racing a commit (`projectrepo.listProjectShares` FOR UPDATE OF s; `TestCommitEncryptionKeyRotation_UnguardedWriterNotOverwritten`); duplicate part rows (`TestGuardedWrites_DuplicatePartRows`); `ImportShare` refusing project entropy; `shareapp.migrateToKeychainIfRequired` (`TestKeychainMigration_DuringRotation_KeepsRotatedCiphertext`); cache re-population across an eviction (`TestService_GetByAPIKey_DoesNotRecacheAcrossEviction`); prepare with a revoked key (`TestAPIKeyRotation_PrepareWithRevokedKeyRefused`, `TestAPISecretRotation_PrepareWithRevokedKeyRefused`); the pending secret refused off the commit route and both rotations interleaved over HTTP (`TestAPISecretRotation_HTTP`, `TestAPISecretRotation_HTTP_InterleavedWithAPIKey`); secrets never logged (`TestAPISecretRotation_SecretsNeverLogged`) |
| Not covered | the 60 s API-key cache on other replicas (documented) |

Bounds: 2 writers, 4 keys of each kind, 3 API secrets. Projects whose key is not migrated
are refused by prepare and commit (`EC_NOT_MIGRATED`) and are outside the
model.

## Sync

Model confirmed against the code at: `e25f6cf33a2ff211df730ae716c3ad5ea8f0c493`

Move this commit forward only after you confirm that the model still
describes the code at the new commit: read the diff of every covered
function, update the model if the behaviour changed, and run both checks.
