# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- Fix linter issues [#22] (https://github.com/xmidt-org/dms/pull/22)

## [v0.2.0]
- Refactored Switch into a simpler interface that removes internal concurrency
- Ensure that as much as possible test logging goes through testing.T
- Added tests for the main DI container

## [v0.1.0]
- README text
- Refactored the main package for modularity in case we want to migrate to a separate package
- Switch to --http, -h for consistency with the go tool chain
- Support dynamic HTTP ports

## [v0.0.2]
- Added the Postponer interface
- Better description

## [v0.0.1]
- Initial creation

[Unreleased]: https://github.com/xmidt-org/dms/compare/v0.2.0..HEAD
[v0.2.0]: https://github.com/xmidt-org/dms/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/xmidt-org/dms/compare/v0.0.2...v0.1.0
[v0.0.2]: https://github.com/xmidt-org/dms/compare/v0.0.1...v0.0.2
[v0.0.1]: https://github.com/xmidt-org/dms/compare/v0.0.0...v0.0.1
