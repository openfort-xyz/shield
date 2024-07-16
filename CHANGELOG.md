# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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