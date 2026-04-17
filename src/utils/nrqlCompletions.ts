/**
 * NRQL context-aware autocomplete provider for Monaco editor.
 * Registers a CompletionItemProvider on the 'sql' language that suggests
 * keywords, functions, event types, fields, and time expressions based
 * on the NRQL clause the cursor is currently in.
 *
 * Coverage based on the full NRQL reference:
 * https://docs.newrelic.com/docs/nrql/nrql-syntax-clauses-functions
 */

// ── Completion data ──────────────────────────────────────────────────

const NRQL_KEYWORDS = [
  // Required clauses
  'SELECT', 'FROM',
  // Filtering & logic
  'WHERE', 'AND', 'OR', 'NOT', 'IN', 'NOT IN', 'LIKE', 'NOT LIKE',
  'RLIKE', 'NOT RLIKE', 'IS', 'IS NOT', 'NULL',
  // Grouping & ordering
  'FACET', 'FACET CASES', 'LIMIT', 'OFFSET', 'ORDER BY', 'ASC', 'DESC',
  // Time
  'SINCE', 'UNTIL', 'TIMESERIES', 'AUTO', 'COMPARE WITH', 'SLIDE BY',
  'WITH TIMEZONE',
  // Labeling & variables
  'AS', 'WITH', 'WITH ... AS',
  // Joins
  'JOIN', 'INNER JOIN', 'LEFT JOIN', 'ON',
  // Metric format
  'WITH METRIC_FORMAT',
  // Other
  'EXTRAPOLATE', 'SHOW EVENT TYPES', 'PREDICT',
];

/** Aggregator functions — used after SELECT */
const NRQL_AGGREGATOR_FUNCTIONS = [
  // Counting
  { label: 'count', insert: 'count(${1:*})', detail: 'Count of events' },
  { label: 'uniqueCount', insert: 'uniqueCount(${1:attribute})', detail: 'Count of unique values' },
  // Math aggregations
  { label: 'sum', insert: 'sum(${1:attribute})', detail: 'Sum of a numeric attribute' },
  { label: 'average', insert: 'average(${1:attribute})', detail: 'Average of a numeric attribute' },
  { label: 'median', insert: 'median(${1:attribute})', detail: 'Median value (50th percentile)' },
  { label: 'min', insert: 'min(${1:attribute})', detail: 'Minimum value' },
  { label: 'max', insert: 'max(${1:attribute})', detail: 'Maximum value' },
  { label: 'stddev', insert: 'stddev(${1:attribute})', detail: 'Standard deviation' },
  { label: 'percentile', insert: 'percentile(${1:attribute}, ${2:95})', detail: 'Percentile of an attribute' },
  // Rate & derivative
  { label: 'rate', insert: 'rate(${1:sum(attribute)}, ${2:1 minute})', detail: 'Rate of change over time' },
  { label: 'derivative', insert: 'derivative(${1:attribute}, ${2:1 minute})', detail: 'Rate of change (derivative)' },
  // Selection
  { label: 'latest', insert: 'latest(${1:attribute})', detail: 'Most recent value' },
  { label: 'earliest', insert: 'earliest(${1:attribute})', detail: 'Oldest value' },
  { label: 'uniques', insert: 'uniques(${1:attribute})', detail: 'List of unique values' },
  // Distribution
  { label: 'histogram', insert: 'histogram(${1:attribute}, ${2:10}, ${3:20})', detail: 'Histogram buckets' },
  { label: 'cdfPercentage', insert: 'cdfPercentage(${1:attribute}, ${2:threshold})', detail: 'Cumulative distribution percentage' },
  { label: 'getCdfCount', insert: 'getCdfCount(${1:attribute}, ${2:threshold})', detail: 'Cumulative distribution count' },
  { label: 'percentage', insert: 'percentage(${1:count(*)}, WHERE ${2:condition})', detail: 'Percentage matching a condition' },
  // Performance
  { label: 'apdex', insert: 'apdex(${1:attribute}, ${2:0.5})', detail: 'Apdex score' },
  { label: 'funnel', insert: 'funnel(${1:attribute}, WHERE ${2:step1} AS ${3:\'Step 1\'}, WHERE ${4:step2} AS ${5:\'Step 2\'})', detail: 'Funnel analysis' },
  // Filtered aggregation
  { label: 'filter', insert: 'filter(${1:count(*)}, WHERE ${2:condition})', detail: 'Filtered aggregation' },
  // Metadata
  { label: 'keyset', insert: 'keyset()', detail: 'List of attributes on an event type' },
  { label: 'eventType', insert: 'eventType()', detail: 'Returns the event type' },
  { label: 'accountId', insert: 'accountId()', detail: 'Returns the account ID' },
  // Bucketing (used with FACET)
  { label: 'buckets', insert: 'buckets(${1:attribute}, ${2:ceiling}, ${3:count})', detail: 'Segment data into buckets' },
];

/** Non-aggregator functions — used in SELECT, WHERE, FACET, WITH */
const NRQL_NON_AGGREGATOR_FUNCTIONS = [
  // String parsing
  { label: 'capture', insert: "capture(${1:attribute}, r'${2:pattern}')", detail: 'Extract values via regex' },
  { label: 'aparse', insert: "aparse(${1:attribute}, '${2:pattern}')", detail: 'Anchor parse — extract values from strings' },
  { label: 'concat', insert: "concat(${1:value1}, ${2:value2})", detail: 'Concatenate values into a string' },
  // Type conversion
  { label: 'numeric', insert: 'numeric(${1:value})', detail: 'Convert string to number' },
  { label: 'string', insert: 'string(${1:value})', detail: 'Convert value to string' },
  { label: 'boolean', insert: 'boolean(${1:value})', detail: 'Convert value to boolean' },
  // Conditional
  { label: 'if', insert: 'if(${1:condition}, ${2:trueVal}, ${3:falseVal})', detail: 'Conditional expression' },
  // String manipulation
  { label: 'length', insert: 'length(${1:attribute})', detail: 'String length' },
  { label: 'lower', insert: 'lower(${1:attribute})', detail: 'Convert to lowercase' },
  { label: 'upper', insert: 'upper(${1:attribute})', detail: 'Convert to uppercase' },
  { label: 'position', insert: 'position(${1:attribute}, ${2:substring})', detail: 'Find substring position' },
  { label: 'substring', insert: 'substring(${1:attribute}, ${2:start}, ${3:length})', detail: 'Extract substring' },
  // Math
  { label: 'abs', insert: 'abs(${1:value})', detail: 'Absolute value' },
  { label: 'ceil', insert: 'ceil(${1:value})', detail: 'Round up to nearest integer' },
  { label: 'floor', insert: 'floor(${1:value})', detail: 'Round down to nearest integer' },
  { label: 'round', insert: 'round(${1:value})', detail: 'Round to nearest integer' },
  { label: 'pow', insert: 'pow(${1:base}, ${2:exponent})', detail: 'Raise to a power' },
  { label: 'sqrt', insert: 'sqrt(${1:value})', detail: 'Square root' },
  { label: 'exp', insert: 'exp(${1:value})', detail: 'e raised to value' },
  { label: 'ln', insert: 'ln(${1:value})', detail: 'Natural logarithm' },
  { label: 'log', insert: 'log(${1:value})', detail: 'Logarithm (base 10)' },
  { label: 'log2', insert: 'log2(${1:value})', detail: 'Logarithm (base 2)' },
  { label: 'log10', insert: 'log10(${1:value})', detail: 'Logarithm (base 10)' },
  { label: 'mod', insert: 'mod(${1:value}, ${2:divisor})', detail: 'Modulo (remainder)' },
  { label: 'clamp_max', insert: 'clamp_max(${1:value}, ${2:max})', detail: 'Clamp value to maximum' },
  { label: 'clamp_min', insert: 'clamp_min(${1:value}, ${2:min})', detail: 'Clamp value to minimum' },
  // Date/time
  { label: 'toDatetime', insert: 'toDatetime(${1:timestamp})', detail: 'Convert epoch to datetime' },
  { label: 'dateOf', insert: 'dateOf(${1:timestamp})', detail: 'Date part of timestamp' },
  { label: 'monthOf', insert: 'monthOf(${1:timestamp})', detail: 'Month from timestamp' },
  { label: 'yearOf', insert: 'yearOf(${1:timestamp})', detail: 'Year from timestamp' },
  { label: 'dayOfMonthOf', insert: 'dayOfMonthOf(${1:timestamp})', detail: 'Day of month from timestamp' },
  { label: 'hourOf', insert: 'hourOf(${1:timestamp})', detail: 'Hour from timestamp' },
  { label: 'minuteOf', insert: 'minuteOf(${1:timestamp})', detail: 'Minute from timestamp' },
  { label: 'weekdayOf', insert: 'weekdayOf(${1:timestamp})', detail: 'Weekday from timestamp' },
  { label: 'weekOf', insert: 'weekOf(${1:timestamp})', detail: 'Week number from timestamp' },
  // Unit conversion
  { label: 'convert', insert: "convert(${1:attribute}, '${2:fromUnit}', '${3:toUnit}')", detail: 'Convert between units' },
  // Encoding & blobs
  { label: 'blob', insert: 'blob(${1:attribute})', detail: 'Base-64 encoded blob' },
  { label: 'decode', insert: 'decode(${1:attribute})', detail: 'Decode base-64 blob' },
  { label: 'encode', insert: 'encode(${1:attribute})', detail: 'Encode to base-64' },
  // Data lookup
  { label: 'lookup', insert: 'lookup(${1:tableName}, ${2:key})', detail: 'Lookup table query' },
  { label: 'mapKeys', insert: 'mapKeys(${1:attribute})', detail: 'Keys from a map attribute' },
  { label: 'mapValues', insert: 'mapValues(${1:attribute})', detail: 'Values from a map attribute' },
  { label: 'jparse', insert: "jparse(${1:attribute}, '${2:path}')", detail: 'Parse JSON attribute' },
];

const NRQL_EVENT_TYPES = [
  // APM
  'Transaction', 'TransactionError', 'Span', 'Metric', 'Error',
  // Logs
  'Log',
  // Browser
  'PageView', 'PageAction', 'BrowserInteraction', 'JavaScriptError', 'AjaxRequest',
  // Mobile
  'Mobile', 'MobileSession', 'MobileCrash', 'MobileRequest', 'MobileHandledException', 'MobileRequestError',
  // Infrastructure
  'SystemSample', 'ProcessSample', 'NetworkSample', 'StorageSample',
  'ContainerSample', 'InfrastructureEvent',
  // Kubernetes
  'K8sContainerSample', 'K8sPodSample', 'K8sNodeSample', 'K8sClusterSample',
  'K8sDeploymentSample', 'K8sNamespaceSample', 'K8sDaemonsetSample',
  // Synthetics
  'SyntheticCheck', 'SyntheticRequest', 'SyntheticMonitor',
  // Serverless
  'AwsLambdaInvocation', 'ServerlessSample',
  // Platform & audit
  'NrAuditEvent', 'NrConsumption', 'NrDailyUsage', 'NrdbQuery',
  'NrIntegrationError', 'Relationship',
];

const NRQL_FIELDS = [
  // Common
  'appName', 'service.name', 'entity.name', 'entityGuid', 'name', 'environment',
  'host', 'hostname',
  // Timing
  'duration', 'responseTime', 'totalTime', 'webDuration',
  'databaseDuration', 'externalDuration', 'databaseCallCount', 'externalCallCount',
  // Errors
  'error', 'error.message', 'error.class', 'errorMessage', 'errorType',
  // HTTP
  'httpResponseCode', 'request.uri', 'request.method', 'http.statusCode',
  // Transaction
  'transactionType', 'transactionSubType',
  // Tracing
  'traceId', 'spanId', 'parentId', 'span.kind', 'sampled', 'priority', 'category',
  // Timestamps & identity
  'timestamp', 'message', 'level', 'userId', 'sessionId', 'session',
  // Browser
  'pageUrl', 'browserTransactionName', 'userAgentName', 'userAgentOS', 'deviceType',
  'backendDuration', 'networkDuration', 'connectionSetupDuration',
  'domProcessingDuration', 'pageRenderingDuration',
  // Geo
  'city', 'regionCode', 'countryCode', 'asnLatitude', 'asnLongitude',
  // Infrastructure
  'cpuPercent', 'memoryUsedPercent', 'memoryUsedBytes', 'diskUsedPercent',
  'containerName', 'containerId',
  // Kubernetes
  'clusterName', 'namespaceName', 'podName', 'deploymentName', 'nodeName',
  // Performance
  'nr.apdexPerfZone',
];

const NRQL_TIME_EXPRESSIONS = [
  '1 minute ago', '5 minutes ago', '15 minutes ago', '30 minutes ago',
  '1 hour ago', '2 hours ago', '6 hours ago', '12 hours ago',
  '1 day ago', '2 days ago', '1 week ago', '1 month ago',
  'today', 'yesterday', 'this week', 'this month',
];

const GRAFANA_VARIABLES = ['$__from', '$__to', '$__interval'];

// ── Context detection ────────────────────────────────────────────────

type NrqlContext = 'select' | 'from' | 'where' | 'facet' | 'since' | 'limit' | 'join' | 'default';

/**
 * Determines the NRQL clause context from the text before the cursor.
 * Exported for testing.
 */
export function getContextFromLine(lineText: string): NrqlContext {
  const upper = lineText.toUpperCase();

  // Walk backwards through the keywords that delimit clauses.
  // Order matters — later clauses take priority when they appear last.
  const clausePatterns: Array<{ pattern: RegExp; context: NrqlContext }> = [
    { pattern: /\bLIMIT\s+$/i, context: 'limit' },
    { pattern: /\b(?:SINCE|UNTIL)\s+$/i, context: 'since' },
    { pattern: /\bFACET\s+$/i, context: 'facet' },
    { pattern: /\b(?:WHERE|AND|OR)\s+$/i, context: 'where' },
    { pattern: /\b(?:INNER\s+JOIN|LEFT\s+JOIN|JOIN)\s*\(\s*$/i, context: 'join' },
    { pattern: /\bFROM\s+$/i, context: 'from' },
    // After SELECT or after a comma following SELECT (still in SELECT clause)
    { pattern: /\bSELECT\s+$/i, context: 'select' },
    { pattern: /,\s*$/i, context: 'select' },
  ];

  for (const { pattern, context } of clausePatterns) {
    if (pattern.test(lineText)) {
      // For comma context, only treat as 'select' if SELECT preceded it
      if (context === 'select' && pattern.source === ',\\s*$') {
        if (upper.includes('SELECT') && !upper.includes('FROM')) {
          return 'select';
        }
        continue;
      }
      return context;
    }
  }

  return 'default';
}

// ── Provider registration ────────────────────────────────────────────

/**
 * Tracks which Monaco instances have already had the NRQL provider registered.
 * WeakSet allows garbage collection of old instances (e.g., when navigating
 * between Dashboard and Explore) while preventing duplicate registrations
 * on the same instance.
 */
const registeredInstances = new WeakSet<any>();

/**
 * Registers the NRQL completion item provider on the 'sql' language.
 * Safe to call multiple times — registers once per Monaco instance.
 * Uses WeakSet to handle multiple Monaco instances (Dashboard vs Explore).
 *
 * @param monaco  The Monaco namespace (received from onBeforeEditorMount)
 */
export function registerNrqlCompletionProvider(monaco: any): void {
  if (registeredInstances.has(monaco)) {
    return;
  }
  registeredInstances.add(monaco);

  monaco.languages.registerCompletionItemProvider('sql', {
    triggerCharacters: [' ', ',', '$', '('],

    provideCompletionItems(model: any, position: any) {
      const lineContent = model.getLineContent(position.lineNumber);
      const textBeforeCursor = lineContent.substring(0, position.column - 1);

      const word = model.getWordUntilPosition(position);
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };

      const context = getContextFromLine(textBeforeCursor);

      const Kind = monaco.languages.CompletionItemKind;
      const SnippetRule = monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet;

      const suggestions: any[] = [];

      switch (context) {
        case 'select': {
          // Aggregator functions first, then non-aggregator functions
          NRQL_AGGREGATOR_FUNCTIONS.forEach((fn, i) => {
            suggestions.push({
              label: fn.label,
              kind: Kind.Function,
              insertText: fn.insert,
              insertTextRules: SnippetRule,
              detail: fn.detail,
              sortText: 'a' + String(i).padStart(3, '0'),
              range,
            });
          });
          NRQL_NON_AGGREGATOR_FUNCTIONS.forEach((fn, i) => {
            suggestions.push({
              label: fn.label,
              kind: Kind.Function,
              insertText: fn.insert,
              insertTextRules: SnippetRule,
              detail: fn.detail,
              sortText: 'b' + String(i).padStart(3, '0'),
              range,
            });
          });
          // Also suggest * for SELECT *
          suggestions.push({
            label: '*',
            kind: Kind.Constant,
            insertText: '*',
            detail: 'All attributes',
            sortText: 'a000',
            range,
          });
          break;
        }

        case 'from':
          NRQL_EVENT_TYPES.forEach((et, i) => {
            suggestions.push({
              label: et,
              kind: Kind.Class,
              insertText: et,
              detail: 'Event type',
              sortText: String(i).padStart(3, '0'),
              range,
            });
          });
          break;

        case 'where':
        case 'facet':
          NRQL_FIELDS.forEach((f, i) => {
            suggestions.push({
              label: f,
              kind: Kind.Field,
              insertText: f,
              detail: 'Attribute',
              sortText: 'a' + String(i).padStart(3, '0'),
              range,
            });
          });
          // Non-aggregator functions are also valid in WHERE/FACET
          NRQL_NON_AGGREGATOR_FUNCTIONS.forEach((fn, i) => {
            suggestions.push({
              label: fn.label,
              kind: Kind.Function,
              insertText: fn.insert,
              insertTextRules: SnippetRule,
              detail: fn.detail,
              sortText: 'b' + String(i).padStart(3, '0'),
              range,
            });
          });
          break;

        case 'since':
          NRQL_TIME_EXPRESSIONS.forEach((t, i) => {
            suggestions.push({
              label: t,
              kind: Kind.Unit,
              insertText: t,
              detail: 'Time expression',
              sortText: String(i).padStart(3, '0'),
              range,
            });
          });
          GRAFANA_VARIABLES.forEach((v, i) => {
            suggestions.push({
              label: v,
              kind: Kind.Variable,
              insertText: v,
              detail: 'Grafana variable',
              sortText: String(100 + i).padStart(3, '0'),
              range,
            });
          });
          break;

        case 'limit':
          [10, 50, 100, 500, 1000, 5000, 'MAX'].forEach((n, i) => {
            suggestions.push({
              label: String(n),
              kind: Kind.Value,
              insertText: String(n),
              detail: 'Limit value',
              sortText: String(i).padStart(3, '0'),
              range,
            });
          });
          break;

        case 'join':
          // After JOIN(, suggest a subquery starting with FROM
          suggestions.push({
            label: 'FROM',
            kind: Kind.Keyword,
            insertText: 'FROM ',
            detail: 'Start subquery',
            sortText: '000',
            range,
          });
          NRQL_EVENT_TYPES.forEach((et, i) => {
            suggestions.push({
              label: et,
              kind: Kind.Class,
              insertText: et,
              detail: 'Event type',
              sortText: String(i + 1).padStart(3, '0'),
              range,
            });
          });
          break;

        default:
          NRQL_KEYWORDS.forEach((kw, i) => {
            suggestions.push({
              label: kw,
              kind: Kind.Keyword,
              insertText: kw + ' ',
              detail: 'NRQL keyword',
              sortText: String(i).padStart(3, '0'),
              range,
            });
          });
          break;
      }

      return { suggestions };
    },
  });
}

/**
 * No-op for backward compatibility with tests.
 * WeakSet handles deduplication automatically — each fresh mock Monaco
 * object in tests is a new reference that won't be in the set.
 */
export function _resetRegistration(): void {
  // No action needed — WeakSet auto-handles new mock objects in tests
}
