# New Relic Grafana Plugin

A high-quality, production-ready Grafana data source plugin for New Relic that enables you to query and visualize your New Relic data directly in Grafana dashboards.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Build Status](https://github.com/your-org/newrelic-grafana-plugin/workflows/CI/badge.svg)](https://github.com/your-org/newrelic-grafana-plugin/actions)
[![Coverage Status](https://coveralls.io/repos/github/your-org/newrelic-grafana-plugin/badge.svg?branch=main)](https://coveralls.io/github/your-org/newrelic-grafana-plugin?branch=main)

## Features

- 🔍 **NRQL Query Support**: Full support for New Relic Query Language (NRQL)
- 🎨 **Visual Query Builder**: Intuitive query builder for common use cases
- 🔒 **Secure Configuration**: API keys and sensitive data are stored securely
- 🌍 **Multi-Region Support**: Support for both US and EU New Relic regions
- ♿ **Accessibility**: Full WCAG 2.1 AA compliance
- 🧪 **Comprehensive Testing**: 95%+ test coverage with unit and integration tests
- 📝 **Template Variables**: Full support for Grafana template variables
- 🚀 **Performance Optimized**: Efficient query processing and caching

## Quick Start

### Prerequisites

- Grafana 10.4.0 or later
- New Relic account with API access
- Node.js 22+ (for development)
- Go 1.21+ (for backend development)

### Installation

#### From Grafana Plugin Catalog (Recommended)

1. Open Grafana and navigate to **Configuration** → **Plugins**
2. Search for "New Relic"
3. Click **Install** on the New Relic plugin
4. Restart Grafana if required

#### Manual Installation

1. Download the latest release from the [releases page](https://github.com/your-org/newrelic-grafana-plugin/releases)
2. Extract the plugin to your Grafana plugins directory:
   ```bash
   unzip newrelic-grafana-plugin-v1.0.0.zip -d /var/lib/grafana/plugins/
   ```
3. Restart Grafana

### Configuration

1. Navigate to **Configuration** → **Data Sources** in Grafana
2. Click **Add data source** and select **New Relic**
3. Configure the following settings:

   - **API Key**: Your New Relic API key (found in New Relic → Account Settings → API Keys)
   - **Account ID**: Your New Relic account ID (visible in the URL when logged into New Relic)
   - **Region**: Select US or EU based on your New Relic account region

4. Click **Save & Test** to verify the connection

## Usage

### Basic NRQL Queries

The plugin supports full NRQL syntax. Here are some examples:

```sql
-- Basic transaction count
SELECT count(*) FROM Transaction SINCE 1 hour ago

-- Average response time by application
SELECT average(duration) FROM Transaction FACET appName SINCE 1 day ago

-- Error rate over time
SELECT percentage(count(*), WHERE error IS true) FROM Transaction TIMESERIES SINCE 1 hour ago

-- Custom attributes
SELECT count(*) FROM Transaction WHERE `custom.userId` = '12345' SINCE 1 hour ago
```

### Query Builder

For common queries, use the visual query builder:

1. Select **Use Query Builder** in the query editor
2. Choose your aggregation function (count, average, sum, etc.)
3. Select the event type (Transaction, Span, Metric, etc.)
4. Add WHERE conditions as needed
5. Configure time range and other options

### Template Variables

The plugin supports Grafana template variables in NRQL queries:

```sql
-- Using dashboard time range
SELECT count(*) FROM Transaction WHERE appName = '$app' $__timeFilter()

-- Using custom variables
SELECT average(duration) FROM Transaction WHERE region = '$region' SINCE $__from
```

### Supported Event Types

- **Transaction**: Application performance data
- **Span**: Distributed tracing data
- **Metric**: Custom and system metrics
- **Log**: Log data (if enabled)
- **Error**: Error tracking data
- **PageView**: Browser monitoring data
- **Mobile**: Mobile application data

## Development

### Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/your-org/newrelic-grafana-plugin.git
   cd newrelic-grafana-plugin
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Build the plugin:
   ```bash
   npm run build
   ```

4. Start development server:
   ```bash
   npm run dev
   ```

### Testing

Run the test suite:

```bash
# Unit tests
npm test

# Integration tests
npm run test:ci

# E2E tests
npm run e2e

# Test coverage
npm run test:coverage
```

### Code Quality

The project maintains high code quality standards:

```bash
# Linting
npm run lint

# Type checking
npm run typecheck

# Format code
npm run lint:fix
```

### Docker Development

Use Docker for isolated development:

```bash
# Start Grafana with the plugin
npm run server

# Access Grafana at http://localhost:3000
# Default credentials: admin/admin
```

## Architecture

### Frontend (React/TypeScript)

- **Components**: Modular React components with TypeScript
- **Validation**: Comprehensive input validation and sanitization
- **Accessibility**: WCAG 2.1 AA compliant
- **Testing**: Jest + React Testing Library

### Backend (Go)

- **API Client**: Secure New Relic GraphQL API integration
- **Query Processing**: NRQL query parsing and validation
- **Caching**: Intelligent response caching
- **Security**: Secure credential handling

### Key Files

```
src/
├── components/
│   ├── ConfigEditor.tsx      # Data source configuration
│   ├── QueryEditor.tsx       # Query editor interface
│   └── NRQLQueryBuilder.tsx  # Visual query builder
├── utils/
│   ├── validation.ts         # Input validation utilities
│   └── logger.ts            # Secure logging utilities
├── types.ts                 # TypeScript type definitions
└── datasource.ts           # Main data source implementation

pkg/
├── plugin/                  # Go backend implementation
├── client/                  # New Relic API client
└── models/                  # Data models
```

## Security

### Data Protection

- API keys are stored securely and never exposed to the frontend
- All user inputs are validated and sanitized
- Secure logging prevents sensitive data exposure
- HTTPS-only communication with New Relic APIs

### Best Practices

- Regular security audits
- Dependency vulnerability scanning
- Secure coding standards
- Input validation and output encoding

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes with tests
4. Run the test suite: `npm test`
5. Commit your changes: `git commit -m 'Add amazing feature'`
6. Push to the branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

### Code Standards

- Follow TypeScript/React best practices
- Maintain 95%+ test coverage
- Include comprehensive documentation
- Follow semantic versioning

## Troubleshooting

### Common Issues

#### Connection Failed

**Problem**: "Failed to connect to New Relic API"

**Solutions**:
- Verify your API key is correct and has proper permissions
- Check that your account ID matches your New Relic account
- Ensure you've selected the correct region (US/EU)
- Verify network connectivity to New Relic APIs

#### Query Errors

**Problem**: "Invalid NRQL query"

**Solutions**:
- Validate your NRQL syntax using New Relic's query builder
- Check that event types and attributes exist in your account
- Ensure proper escaping of special characters
- Verify time range syntax

#### Performance Issues

**Problem**: Slow query responses

**Solutions**:
- Optimize your NRQL queries with appropriate LIMIT clauses
- Use FACET sparingly for large datasets
- Consider shorter time ranges for complex queries
- Implement query caching where appropriate

### Debug Mode

Enable debug logging by setting the environment variable:

```bash
export NODE_ENV=development
```

This will provide detailed logging information in the browser console.

## API Reference

### Configuration Options

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `apiKey` | string | Yes | New Relic API key |
| `accountId` | number | Yes | New Relic account ID |
| `region` | 'US' \| 'EU' | No | New Relic region (default: US) |
| `apiUrl` | string | No | Custom API endpoint URL |

### Query Options

| Option | Type | Description |
|--------|------|-------------|
| `queryText` | string | NRQL query string |
| `accountID` | number | Override account ID for this query |

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a detailed history of changes.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](https://github.com/your-org/newrelic-grafana-plugin/wiki)
- 🐛 [Issue Tracker](https://github.com/your-org/newrelic-grafana-plugin/issues)
- 💬 [Discussions](https://github.com/your-org/newrelic-grafana-plugin/discussions)
- 📧 [Email Support](mailto:support@yourorg.com)

## Acknowledgments

- [Grafana](https://grafana.com/) for the excellent plugin framework
- [New Relic](https://newrelic.com/) for the comprehensive observability platform
- The open-source community for valuable feedback and contributions

---

Made with ❤️ by the [Your Organization](https://yourorg.com) team
