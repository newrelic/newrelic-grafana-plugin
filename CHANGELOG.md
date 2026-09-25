# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.1] - 2026-09-25

### Fixed
- Timeout override not taking effect for queries running longer than 30 seconds.

## [0.3.0] - 2026-09-22

### Added
- Per-panel NRQL query timeout override (up to 120 seconds), for queries that time out under NerdGraph's default timeout during high-traffic events. Out-of-range values are clamped to New Relic's allowed range.

### Changed
- Bumped `newrelic-client-go` to v2.94.1.
- Bumped `glob`, `webpack`, `browserslist`, `js-yaml`, and `qs` to close several open high-severity CVEs.

## [0.2.1] - 2026-07-31

### Fixed
- EU region accounts receiving a `403 not authorized for account region` error. The region selected in the datasource configuration is now correctly propagated to the New Relic API client.

## [0.2.0] - 2026-04-17

### Added
- Query New Relic logs directly in Grafana, with color-coded severity, drill-down details, and a volume histogram in Explore.
- Smart NRQL autocomplete with 190+ context-aware suggestions across dashboards, Explore, and template variables.
- Editor theme now automatically matches Grafana's light/dark mode.
- Logs and alerting capabilities declared for full Grafana panel compatibility.

### Fixed
- Period-over-period comparisons, multi-dimensional facets, and complex query results now render correctly.

## [0.1.0] - 2025-06-23

### Added
- Initial release of the New Relic Grafana Plugin
- Complete NRQL (New Relic Query Language) support
- Support for all major aggregation functions: count, sum, average, min, max, percentile, median, etc.
- Faceted query support with proper grouping and time series handling
- Multi-aggregation query support (e.g., SELECT sum(duration), average(duration), count(*))
- Percentile query support with proper field extraction
- Filter function support for error rate calculations
- Histogram data visualization support
- Template variable support in NRQL queries
- Secure API key storage using Grafana's secure storage
- Multi-region support (US and EU New Relic regions)
- Comprehensive error handling and connection validation
- Field naming convention alignment with New Relic's API response format
- Nullable type handling for robust data visualization

### Features
- NRQL query editor with validation
- Secure data source configuration
- Advanced response formatting for all New Relic data types
- Intelligent field type detection and conversion
- Time field processing for accurate time series visualization
- Support for complex nested aggregation responses

### Technical Implementation
- Frontend: React + TypeScript with Grafana UI components
- Backend: Go with New Relic API integration
- Comprehensive test coverage (unit, integration, E2E)
- Performance optimized query processing
- Secure credential handling

### Supported Query Types
- Simple aggregation queries
- Faceted queries with grouping
- Time series queries with TIMESERIES clause
- Complex multi-field aggregations
- Percentile and statistical calculations
- Histogram data queries
- Error rate and success rate calculations

### Known Limitations
- Plugin requires manual installation (unsigned)
- Complex histogram queries may need additional handling
- Multi-facet query support is basic (single facet recommended)

This initial release provides comprehensive NRQL support with proper field handling and secure configuration for Grafana users to visualize their New Relic data.
