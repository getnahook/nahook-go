# Changelog

All notable changes to this SDK are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/) and
this project follows [Semantic Versioning](https://semver.org/).

## [0.3.0] - 2026-06-12


### Features

- Application maxEndpoints + showEventTypes across all 8 SDKs

## [0.2.1] - 2026-05-31

### Features

- Close() for graceful connection-pool drain

## [0.2.0] - 2026-05-31

### Features

- Tuned http.Transport defaults + WithHTTPClient option
- Add Deliveries resource to the management client

### Bug Fixes

- Classify http.Client.Timeout via net.Error.Timeout()

## [0.1.1] - 2026-05-25

### Features

- Expose optional environmentId on endpoints.create
- Add environments resource to the management client
- Embed workspace region in API keys for SDK auto-routing

## [0.1.0] - 2026-04-10

### Features

- Initial release of the Nahook Go SDK
