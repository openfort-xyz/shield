# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.2.3]
### Updated
- JWT token library

## [v0.2.0]
### Added
- Keychains

## [v0.1.27]
### Added
- MySQL Certificate for SSL connection compatibility
### Updated
- Logger to be compatible with Google Cloud Logging

## [v0.1.26]
### Added
- Metrics for HTTP requests and Prometheus to expose them
### Updated
- Deployment pipeline

## [v0.1.25]
### Added
- Index on external users table

## [v0.1.24]
### Added
- Health check endpoint

## [v0.1.23]
### Fixed
- Update crypto to v0.31.0 package because of CVE-2024-45337

## [v0.1.22]
### Added
- Endpoint to get the encryption type of share

## [v0.1.21]
### Fixed
- Register Share encryption validation switch

## [v0.1.20]
### Fixed
- Register Share encryption validation for entropy `none`

## [v0.1.19]
### Fixed
- Control error invalid encryption part
- Fix docs register share endpoint

## [v0.1.18]
### Fixed
- Invalid API authentication message
- PEM/Key type parsing for custom authentication
- PEM/Key type null values on database
- X-Request-Code value
### Updated
- README documentation

## [v0.1.17]
### Updated
- Added X-Request-ID header to third party openfort authentication

## [v0.1.16]
### Fixed
- On share validator add encryption_session to validate project encryption

## [v0.1.15]
### Fixed
- Added X-Encryption-Session to allowed headers

## [v0.1.14]
### Fixed
- Added X-Request-ID to allowed headers
## Updated
- UUID used for X-Request-ID updated to v7

## [v0.1.13] - 2024-07-15
### Fixed
- Bulk update Incorrect datetime value: '0000-00-00' for column 'created_at'

## [v0.1.12] - 2024-07-15
### Added
- Using Openfort SSS library to split/reconstruct encryption keys. 
- Add `shld_shamir_migrations` to manage the migrations of the Shamir secret sharing library.
- Add Migration Jobs to manage the migrations of the Shamir secret sharing library.


## [v0.1.11] - 2024-07-11
### Added
- Encryption Sessions, allow projects to register a on time use session with an encryption part to encrypt/decrypt a secret.