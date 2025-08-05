<a href="https://opensource.newrelic.com/oss-category/#community-plus"><picture><source media="(prefers-color-scheme: dark)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/dark/Community_Plus.png"><source media="(prefers-color-scheme: light)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Plus.png"><img alt="New Relic Open Source community plus project banner." src="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Plus.png"></picture></a>

[![GitHub release](https://img.shields.io/github/release/newrelic/newrelic-grafana-plugin.svg)](https://github.com/newrelic/newrelic-grafana-plugin/releases)
[![License](https://img.shields.io/badge/License-AGPL%203.0-orange.svg)](https://opensource.org/license/agpl-v3)

# New Relic Grafana Plugin

This plugin allows you to visualize New Relic data directly in Grafana dashboards using NRQL (New Relic Query Language). The plugin provides comprehensive support for all New Relic data types, aggregation functions, and advanced query capabilities.

## Features

* Full NRQL (New Relic Query Language) support
* Comprehensive aggregation functions (count, sum, average, percentile, etc.)
* Faceted queries with proper grouping and time series handling
* Multi-aggregation query support
* Percentile calculations with object handling
* Filter function support for error rate calculations
* Template variable integration
* Secure API key storage using Grafana's secure storage
* Multi-region support (US and EU New Relic regions)
* Time series data visualization with accurate time field handling

## Current Support:

This project targets Grafana data source plugins and supports:
- Grafana 10.4.0 or higher
- New Relic accounts with API access
- Both US and EU New Relic regions
- All major NRQL query types and aggregation functions

## Installation

We recommend installing the plugin directly from the Grafana Catalog. For air-gapped or offline environments, a manual installation method is also available.

### Prerequisites

- Grafana 10.4.0 or later
- A New Relic account with a User API Key

### From the Grafana Catalog (Recommended)
This is the simplest way to install for most users.

#### Using the Grafana UI:
1. Navigate to Administration → Plugins and Data → Plugins in your Grafana instance.
2. Search for "New Relic".
3. Click on the plugin, then click the Install button.


#### Using the Command Line:
Run the following command on your Grafana server:
```bash
grafana-cli plugins install nrlabs-newrelic-datasource
```
After installing via either method, you must restart the Grafana server for the plugin to be recognized.

For detailed instructions on how to install the plugin on Grafana Cloud or locally, please check out the [Plugin installation docs](https://grafana.com/docs/grafana/latest/administration/plugin-management/).

## Grafana Setup

1. Navigate to **Configuration** → **Data Sources** in Grafana
2. Click **Add data source** and select **New Relic**
3. Configure the following settings (don't forget to put proper credentials):

```bash
API Key: Your New Relic User API Key
Account ID: Your New Relic account ID  
Region: US or EU (based on your New Relic account region)
```

4. Click **Save & Test** to verify the connection

### Finding Your New Relic Credentials

- **Account ID**: Found in the URL when logged into New Relic (e.g., `https://one.newrelic.com/accounts/YOUR_ACCOUNT_ID`)
- **API Key**: Create a User API Key in New Relic → User menu → API keys

## Usage

See the examples below, and for more detail, see [New Relic NRQL documentation](https://docs.newrelic.com/docs/query-your-data/nrql-new-relic-query-language/get-started/introduction-nrql-new-relics-query-language/).

### Basic NRQL Queries

The plugin supports full NRQL syntax. Here are some examples:

```sql
-- Basic transaction count
SELECT count(*) FROM Transaction SINCE 1 hour ago

-- Average response time by application
SELECT average(duration) FROM Transaction FACET appName SINCE 1 day ago

-- Error rate over time
SELECT percentage(count(*), WHERE error IS true) FROM Transaction TIMESERIES SINCE 1 hour ago
```

### [Aggregation Functions](https://docs.newrelic.com/docs/query-your-data/nrql-new-relic-query-language/get-started/nrql-syntax-clauses-functions/#functions)

The plugin recognizes and properly handles all New Relic aggregation functions:

```sql
-- Basic aggregations
SELECT count(*), sum(duration), average(duration), min(duration), max(duration) FROM Transaction

-- Statistical functions
SELECT percentile(duration, 95), median(duration), stddev(duration) FROM Transaction

-- Unique value functions
SELECT uniqueCount(userId), uniques(appName) FROM Transaction

-- Time-based functions
SELECT latest(timestamp), earliest(timestamp), rate(count(*), 1 minute) FROM Transaction
```

### [Faceted Queries](https://docs.newrelic.com/docs/query-your-data/nrql-new-relic-query-language/get-started/nrql-syntax-clauses-functions/#sel-facet)

Advanced support for faceted queries with proper grouping:

```sql
-- Faceted aggregation
SELECT average(duration) FROM Transaction FACET appName SINCE 1 day ago

-- Faceted time series
SELECT count(*) FROM Transaction FACET request.uri TIMESERIES 1 hour SINCE 1 day ago

-- Multi-aggregation with facets
SELECT sum(duration), average(duration), count(*) FROM Transaction FACET appName TIMESERIES SINCE 1 hour ago
```

### [Time Series Queries](https://docs.newrelic.com/docs/query-your-data/nrql-new-relic-query-language/get-started/nrql-syntax-clauses-functions/#sel-timeseries)

Native support for TIMESERIES queries:

```sql
-- Basic time series
SELECT count(*) FROM Transaction TIMESERIES SINCE 1 hour ago

-- Time series with custom intervals
SELECT average(duration) FROM Transaction TIMESERIES 5 minutes SINCE 1 day ago

-- Percentile analysis over time
SELECT percentile(duration, 95) FROM Transaction TIMESERIES 1 hour SINCE 1 day ago
```

### [Template Variables](https://grafana.com/docs/grafana/latest/dashboards/variables/)

The plugin supports Grafana template variables in NRQL queries:

```sql
-- Using dashboard variables
SELECT count(*) FROM Transaction WHERE appName = '$app' SINCE $__from UNTIL $__to

-- Using facet variables
SELECT average(duration) FROM Transaction WHERE region = '$region' TIMESERIES
```

### [Filter Functions](https://docs.newrelic.com/docs/query-your-data/nrql-new-relic-query-language/get-started/nrql-syntax-clauses-functions/#func-filter)

Support for filter() function results:

```sql
-- Error rate calculation
SELECT filter(count(*), WHERE error IS true) as 'Error Count', 
       filter(count(*), WHERE error IS false) as 'Success Count' 
FROM Transaction TIMESERIES
```

### Field Naming

The plugin preserves New Relic's field naming conventions:
- Aggregations: `sum.duration`, `average.responseTime`, `count`
- Percentiles: `percentile.duration.95`, `percentile.duration.99`
- Apdex: `apdex.score`, `apdex.s`, `apdex.t`, `apdex.f`
- Filters: `ErrorCount`, `SuccessCount`, `Error Rate`

## Development

### Prerequisites

- Node.js 22+
- Go 1.21+
- Git
- Docker (optional, for containerized development)

### Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/newrelic/newrelic-grafana-plugin.git
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

### Build Commands

```bash
# Development build with watching
npm run dev

# Production build
npm run build

# Run tests
npm run test:ci

# Lint code
npm run lint

# Type checking
npm run typecheck
```

### Docker Development

Start a complete development environment with Grafana:

```bash
npm run server
```

Access Grafana at `http://localhost:3000` (admin/admin)

## Troubleshooting

### Connection Failed
**Problem**: "Failed to connect to New Relic API"

**Solutions**:
- Verify your API key is correct and has proper permissions
- Check that your account ID matches your New Relic account
- Ensure you've selected the correct region (US/EU)
- Verify network connectivity to New Relic APIs

### Query Errors
**Problem**: "Invalid NRQL query"

**Solutions**:
- Validate your NRQL syntax using New Relic's query builder
- Check that event types and attributes exist in your account
- Ensure proper escaping of special characters in strings
- Verify time range syntax

### Empty Results
**Problem**: Query returns no data

**Solutions**:
- Verify the time range contains data for your query
- Check that the event type and filters match your data
- Ensure your account has data for the specified time range
- Try a simpler query first to verify connectivity

### Debug Mode

Enable debug logging by setting the log level to "debug" in Grafana configuration. This will provide detailed logging information for troubleshooting.

## Support

New Relic hosts and moderates an online forum where customers, users, maintainers, contributors, and New Relic employees can discuss and collaborate:

[forum.newrelic.com](https://forum.newrelic.com/).

## Contribute

We encourage your contributions to improve this project! Keep in mind that when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.

If you have any questions, or to execute our corporate CLA (which is required if your contribution is on behalf of a company), drop us an email at opensource@newrelic.com.

**A note about vulnerabilities**

As noted in our [security policy](SECURITY.md), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

If you would like to contribute to this project, review [these guidelines](./CONTRIBUTING.md).

To all contributors, we thank you! Without your contribution, this project would not be what it is today.

## License

This project is licensed under the terms of the GNU Affero General Public License v3.0 or New Relic Software License v. 1.0.

Please see the [LICENSE](LICENSE) file for full details on both licenses.

The `newrelic-grafana-plugin` may use source code from third-party libraries. When used, these libraries will be outlined in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
