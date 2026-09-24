import React, { useState, useEffect, useCallback, useRef } from 'react';
import { QueryEditorProps } from '@grafana/data';
import { Button, Switch, ButtonGroup, Icon, Tooltip, CodeEditor, InlineField, InlineFieldRow, Input } from '@grafana/ui';
import { DataSource } from '../datasource';
import { NewRelicQuery, NewRelicDataSourceOptions } from '../types';
import { NRQLQueryBuilder } from './query/NRQLQueryBuilder';
import { validateNrqlQuery } from '../utils/validation';
import { logger } from '../utils/logger';
import { buildNRQLWithTimeIntegration, hasGrafanaTimeVariables, GRAFANA_TIME_VARIABLES } from '../utils/timeUtils';
import { registerNrqlCompletionProvider } from '../utils/nrqlCompletions';
type Props = QueryEditorProps<DataSource, NewRelicQuery, NewRelicDataSourceOptions>;

// New Relic's allowed NRQL query timeout range, matching the backend clamp.
const MIN_QUERY_TIMEOUT_SECONDS = 5;
const MAX_QUERY_TIMEOUT_SECONDS = 120;

/**
 * Query editor component for New Relic NRQL queries
 * Provides both a visual query builder and raw text editor with time picker integration
 */
export function QueryEditor({ query, onChange, onRunQuery, range }: Props) {
  const [useQueryBuilder, setUseQueryBuilder] = useState(false);
  // Starts open when a timeout override is already set, so a remount (e.g. from
  // toggling Auto time) can't visually hide a value that's still in effect.
  const [advancedExpanded, setAdvancedExpanded] = useState(() => query.timeoutSeconds != null);
  // Hidden while typing, so partial input (e.g. "1" on the way to "130") doesn't flash a notice.
  const [showTimeoutCapNotice, setShowTimeoutCapNotice] = useState(true);
  const [validationError, setValidationError] = useState<string>('');
  const [useGrafanaTime, setUseGrafanaTime] = useState(
    query.useGrafanaTime ?? !hasGrafanaTimeVariables(query.queryText || '')
  );

  // Tracks the latest timeout override outside of the query prop. Other
  // controls (e.g. the Auto time switch) build their next query object from
  // `query`, which can still reflect a render Grafana hasn't caught up to yet
  // if it arrives right after typing here — this ref is always current, so
  // those handlers can re-assert it explicitly instead of trusting the prop.
  const timeoutSecondsRef = useRef(query.timeoutSeconds);
  useEffect(() => {
    timeoutSecondsRef.current = query.timeoutSeconds;
  }, [query.timeoutSeconds]);

  //On Editor Change Callback function
  function handleEditorChange(queryString: string) {
    handleNRQLChange(queryString);
  }

  //On Editor did mount life cycle hook function
  function handleEditorDidMount(editor: any, monaco: any) {
    // Sync editor content with query text on mount — getDefaultQuery() may
    // populate query.queryText after the initial rawNRQL state is set to ''.
    if (query.queryText && editor.getValue() !== query.queryText) {
      editor.setValue(query.queryText);
      setRawNRQL(query.queryText);
    }
  }
  // Local state for the raw NRQL text
  const [rawNRQL, setRawNRQL] = useState(query.queryText || '');

  // Track if user is currently typing to prevent cursor jumps
  const isUserTypingRef = useRef(false);
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Validates the NRQL query and updates validation state (only called when running)
   */
  const validateQuery = useCallback((queryText: string) => {
    if (!queryText.trim()) {
      setValidationError('Query cannot be empty');
      return false;
    }

    try {
      const validation = validateNrqlQuery(queryText);
      setValidationError(validation.isValid ? '' : validation.message || 'Invalid query');

      if (!validation.isValid) {
        logger.warn('Query validation failed', {
          refId: query.refId,
          error: validation.message,
        });
      }

      return validation.isValid;
    } catch (error) {
      logger.error('Error validating query', error as Error, {
        refId: query.refId,
      });
      setValidationError('Error validating query');
      return false;
    }
  }, [query.refId]);

  // Populate default query on mount if editor is empty
  useEffect(() => {
    if (!query.queryText || query.queryText.trim() === '') {
      const defaultQuery = useGrafanaTime
        ? 'SELECT count(*) FROM Transaction SINCE $__from UNTIL $__to'
        : 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
      setRawNRQL(defaultQuery);
      onChange({ ...query, queryText: defaultQuery, useGrafanaTime });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Update rawNRQL when query changes externally (but don't validate)
  useEffect(() => {
    if (!isUserTypingRef.current && query.queryText !== rawNRQL) {
      setRawNRQL(query.queryText || '');
      // Clear any existing validation errors when query changes externally
      setValidationError('');
    }
  }, [query.queryText, rawNRQL]);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
      }
    };
  }, []);

  /**
   * Handles changes to the raw NRQL text (no validation while typing)
   */
  const handleNRQLChange = useCallback((queryText: string) => {
    // Mark that user is actively typing
    isUserTypingRef.current = true;

    // Clear existing timeout
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }

    // Set timeout to mark typing as finished
    typingTimeoutRef.current = setTimeout(() => {
      isUserTypingRef.current = false;
    }, 500);

    setRawNRQL(queryText);

    // Clear validation errors while typing - don't show errors until user tries to run
    setValidationError('');

    // For time integration, we need to be careful not to modify the text that's being typed
    // Only apply time integration when the user stops typing
    const finalQuery = queryText; // Keep the raw user input

    const updatedQuery = { ...query, queryText: finalQuery, useGrafanaTime, timeoutSeconds: timeoutSecondsRef.current };
    onChange(updatedQuery);
  }, [query, onChange, useGrafanaTime]);

  /**
   * Handles changes from the query builder (also no validation while building)
   */
  const handleBuilderQueryChange = useCallback((queryText: string) => {
    setRawNRQL(queryText); // Keep in sync

    // Clear validation errors while building query
    setValidationError('');

    // Apply time integration if enabled
    const finalQuery = useGrafanaTime ?
      buildNRQLWithTimeIntegration(queryText, true) :
      queryText;

    const updatedQuery = { ...query, queryText: finalQuery, useGrafanaTime, timeoutSeconds: timeoutSecondsRef.current };
    onChange(updatedQuery);
  }, [query, onChange, useGrafanaTime]);

  /**
   * Handles toggling Grafana time picker integration (no validation)
   */
  const handleTimeIntegrationToggle = useCallback((enabled: boolean) => {
    setUseGrafanaTime(enabled);

    let updatedQueryText = rawNRQL;

    if (enabled) {
      // Enable Grafana time integration
      updatedQueryText = buildNRQLWithTimeIntegration(rawNRQL, true);
    } else {
      // Remove Grafana time variables and use manual time clauses
      // Handle the new SINCE/UNTIL format
      updatedQueryText = rawNRQL.replace(
        /SINCE\s+\$__from\s+UNTIL\s+\$__to/gi,
        'SINCE 1 hour ago'
      );

      // Also handle any remaining old WHERE format (for backwards compatibility)
      updatedQueryText = updatedQueryText.replace(
        /WHERE\s+timestamp\s*>=\s*\$__from\s*AND\s*timestamp\s*<=\s*\$__to/gi,
        'SINCE 1 hour ago'
      );
    }

    const updatedQuery = {
      ...query,
      queryText: updatedQueryText,
      useGrafanaTime: enabled,
      // Re-assert explicitly rather than trusting `query` to still carry it —
      // this can fire before Grafana's re-render from the timeout field's own
      // onChange has landed, so `query` here may be one step behind.
      timeoutSeconds: timeoutSecondsRef.current,
    };

    onChange(updatedQuery);
    setRawNRQL(updatedQueryText);
    // Clear validation errors when changing time integration
    setValidationError('');

    logger.debug('Time integration toggled', {
      refId: query.refId,
      enabled,
      hasGrafanaVars: hasGrafanaTimeVariables(updatedQueryText),
    });
  }, [rawNRQL, query, onChange]);

  /**
   * Handles typing into the per-panel query timeout override. Blank clears it,
   * so no timeout is sent and NerdGraph's own default applies.
   */
  const handleTimeoutOverrideChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    const next = value === '' ? undefined : Number(value);
    setShowTimeoutCapNotice(false);
    timeoutSecondsRef.current = next;
    onChange({ ...query, timeoutSeconds: next });
  }, [query, onChange]);

  // The entered value is kept as-is; the backend clamps it, and this notice says what will apply.
  const timeoutCappedTo =
    query.timeoutSeconds == null || Number.isNaN(query.timeoutSeconds)
      ? null
      : query.timeoutSeconds > MAX_QUERY_TIMEOUT_SECONDS
        ? MAX_QUERY_TIMEOUT_SECONDS
        : query.timeoutSeconds < MIN_QUERY_TIMEOUT_SECONDS
          ? MIN_QUERY_TIMEOUT_SECONDS
          : null;
  const timeoutCappedMessage =
    showTimeoutCapNotice && timeoutCappedTo != null
      ? `Capped to ${timeoutCappedTo}s (New Relic's ${timeoutCappedTo === MAX_QUERY_TIMEOUT_SECONDS ? 'max' : 'min'})`
      : '';

  /**
   * Toggles between query builder and text editor
   */
  const toggleQueryBuilder = useCallback(() => {
    const newMode = !useQueryBuilder;
    setUseQueryBuilder(newMode);

    logger.debug('Query editor mode changed', {
      refId: query.refId,
      mode: newMode ? 'builder' : 'text',
    });

    if (newMode) {
      // When switching to query builder, ensure the query is in a valid format
      if (!query.queryText || query.queryText.trim() === '') {
        const defaultQuery = useGrafanaTime
          ? 'SELECT count(*) FROM Transaction SINCE $__from UNTIL $__to'
          : 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
        setRawNRQL(defaultQuery);
        onChange({ ...query, queryText: defaultQuery, useGrafanaTime });
      }
    }

    // Clear validation errors when switching modes
    setValidationError('');
  }, [useQueryBuilder, query, onChange, useGrafanaTime]);

  /**
   * Handles running the query - validate only when attempting to run
   */
  const handleRunQuery = useCallback(() => {
    try {
      // Validate only when running the query
      const queryToValidate = useGrafanaTime ?
        buildNRQLWithTimeIntegration(query.queryText || '', true) :
        query.queryText || '';

      const isValid = validateQuery(queryToValidate);

      if (!isValid) {
        logger.warn('Attempted to run invalid query', {
          refId: query.refId,
          error: validationError,
        });
        return; // Don't run the query if validation fails
      }

      logger.debug('Running query', {
        refId: query.refId,
        useGrafanaTime,
        hasTimeVars: hasGrafanaTimeVariables(query.queryText || ''),
      });
      onRunQuery();
    } catch (error) {
      logger.error('Error running query', error as Error, {
        refId: query.refId,
      });
    }
  }, [query.refId, query.queryText, useGrafanaTime, onRunQuery, validateQuery, validationError]);

  return (
    <div style={{ padding: '8px 0' }}>
      {/* Header Controls */}
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: '12px',
        gap: '8px'
      }}>
        {/* Left side - Editor mode toggle */}
        <ButtonGroup>
          <Button
            variant={!useQueryBuilder ? 'primary' : 'secondary'}
            size="sm"
            onClick={() => !useQueryBuilder || toggleQueryBuilder()}
          >
            <Icon name="edit" style={{ marginRight: '4px' }} />
            NRQL Editor
          </Button>
          <Button
            variant={useQueryBuilder ? 'primary' : 'secondary'}
            size="sm"
            onClick={() => useQueryBuilder || toggleQueryBuilder()}
          >
            <Icon name="apps" style={{ marginRight: '4px' }} />
            Query Builder
          </Button>
        </ButtonGroup>

        {/* Right side - Time picker toggle and run button */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
            <Icon name="clock-nine" size="sm" />
            <span style={{ fontSize: '12px', color: '#8e8e8e' }}>Auto time</span>
            <Tooltip content={useGrafanaTime ? `Query uses dashboard time range. Variables: ${Object.values(GRAFANA_TIME_VARIABLES).slice(0, 3).join(', ')}...` : "Enable to use Grafana's dashboard time picker instead of manual time clauses"}>
              <Icon name="info-circle" size="xs" style={{ cursor: 'help', color: '#8e8e8e' }} />
            </Tooltip>
            <Switch
              value={useGrafanaTime}
              onChange={(e) => handleTimeIntegrationToggle(e.currentTarget.checked)}
              data-testid="grafana-time-toggle"
            />
          </div>

          <Button
            variant="primary"
            size="sm"
            onClick={handleRunQuery}
            disabled={!!validationError || !query.queryText?.trim()}
            icon="play"
          >
            {validationError ? 'Invalid query' : 'Run'}
          </Button>
        </div>
      </div>

      {/* Advanced: per-panel query timeout override */}
      <div style={{ marginBottom: '12px' }}>
        <Button
          variant="secondary"
          fill="text"
          size="sm"
          icon={advancedExpanded ? 'angle-down' : 'angle-right'}
          onClick={() => setAdvancedExpanded((prev) => !prev)}
          data-testid="query-editor-advanced-toggle"
        >
          Advanced
        </Button>
        {advancedExpanded && (
          <InlineFieldRow>
            <InlineField
              label="Override Timeout"
              labelWidth={20}
              tooltip="Overrides the default query timeout for this panel, up to 120 seconds."
              invalid={!!timeoutCappedMessage}
              error={timeoutCappedMessage}
            >
              <Input
                id="query-editor-timeout-override"
                data-testid="query-timeout-override-input"
                type="number"
                min={MIN_QUERY_TIMEOUT_SECONDS}
                max={MAX_QUERY_TIMEOUT_SECONDS}
                value={query.timeoutSeconds ?? ''}
                width={30}
                suffix="seconds"
                onChange={handleTimeoutOverrideChange}
                onBlur={() => setShowTimeoutCapNotice(true)}
                aria-label="Override Timeout in seconds"
              />
            </InlineField>
          </InlineFieldRow>
        )}
      </div>

      {/* Query Editor Content */}
      {useQueryBuilder ? (
        <div role="region" aria-label="NRQL Query Builder">
          <NRQLQueryBuilder
            value={query.queryText || ''}
            onChange={handleBuilderQueryChange}
            onRunQuery={handleRunQuery}
            useGrafanaTime={useGrafanaTime}
          />
        </div>
      ) : (
        <div role="region" aria-label="NRQL Text Editor">

          <CodeEditor
            height="150px"
            language="sql"
            value={rawNRQL}
            onBlur={handleEditorChange}
            onChange={handleEditorChange}
            onBeforeEditorMount={(monaco) => registerNrqlCompletionProvider(monaco)}
            onEditorDidMount={handleEditorDidMount}
            showMiniMap={false}
            monacoOptions={{ wordWrap: 'on' }}
          />
         

          {/* Help Text */}
          <div id="nrql-help" style={{
            fontSize: '11px',
            color: '#8e8e8e',
            marginTop: '6px',
            display: 'flex',
            alignItems: 'center',
            gap: '4px'
          }}>
            <Icon name="question-circle" size="xs" />
            {useGrafanaTime
              ? 'Use $__from, $__to variables for automatic time picker integration'
              : 'Use manual time clauses like "SINCE 1 hour ago"'
            }
          </div>

          {/* Status Indicator - only show validation errors */}
          <div style={{
            marginTop: '8px',
            minHeight: '20px', // Reserve space to prevent layout shift
            display: 'flex',
            alignItems: 'center',
            gap: '6px'
          }}>
            {validationError ? (
              // Error state only - no success messages
              <div style={{
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                fontSize: '12px',
                color: '#d63031'
              }}>
                <Icon name="exclamation-triangle" size="sm" />
                <span>{validationError}</span>
              </div>
            ) : null}
          </div>
        </div>
      )}
    </div>
  );
}
