# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.2.24]
### What
This release changes project OTP generation rate limiting from minute window to hour window.
Also rate limiting of OTP generation per user was dropped because it was excessive.

## [v0.2.22]
### What
This release adds OTP as an optional secutify feature for the projects. If 2FA is enabled for the project Shield will require an OTP during encrypted session creation.

OTP may be sent either with email or SMS.

## [v0.2.21]
### What
This release fixes a bug which happens only in case one external user has two shield accounts and each of that accounts has own keychain. Usually it's not happening though.

### Why
So if user has two accounts and two keychains and he wants to select cold share by reference for key reconstruction API now may select a wrong shield user on the middleware level, and then as a result select wrong keychain which doesn't related to the share with reference which was sent in API method arguments.

It means that API throws share not found error.

## How
With these changes we start to save external user ID to the context, on the middleware level. And it adds a new endpoint to select a share by reference.

Once we selected a share by reference we check if external user really owns it and if yes return it.

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