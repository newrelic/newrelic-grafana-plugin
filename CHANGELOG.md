# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.1] - 2025-06-23

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
