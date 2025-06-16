import React, { useState, useEffect, useCallback, useRef } from 'react';
import { QueryEditorProps } from '@grafana/data';
import { InlineField, InlineFieldRow, Button, Alert, Switch, ButtonGroup, Icon, TextArea, Tooltip } from '@grafana/ui';
import { DataSource } from '../datasource';
import { NewRelicQuery, NewRelicDataSourceOptions } from '../types';
import { NRQLQueryBuilder } from './query/NRQLQueryBuilder';
import { validateNrqlQuery } from '../utils/validation';
import { logger } from '../utils/logger';
import { buildNRQLWithTimeIntegration, hasGrafanaTimeVariables, GRAFANA_TIME_VARIABLES } from '../utils/timeUtils';

type Props = QueryEditorProps<DataSource, NewRelicQuery, NewRelicDataSourceOptions>;

/**
 * Query editor component for New Relic NRQL queries
 * Provides both a visual query builder and raw text editor with time picker integration
 */
export function QueryEditor({ query, onChange, onRunQuery, range }: Props) {
  const [useQueryBuilder, setUseQueryBuilder] = useState(false);
  const [validationError, setValidationError] = useState<string>('');
  const [isValidating, setIsValidating] = useState(false);
  const [useGrafanaTime, setUseGrafanaTime] = useState(
    query.useGrafanaTime ?? !hasGrafanaTimeVariables(query.queryText || '')
  );

  // Local state for the raw NRQL text
  const [rawNRQL, setRawNRQL] = useState(query.queryText || '');
  
  // Track if user is currently typing to prevent cursor jumps
  const isUserTypingRef = useRef(false);
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  /**
   * Validates the NRQL query and updates validation state
   */
  const validateQuery = useCallback(async (queryText: string) => {
    if (!queryText.trim()) {
      setValidationError('');
      return;
    }

    setIsValidating(true);
    
    try {
      const validation = validateNrqlQuery(queryText);
      setValidationError(validation.isValid ? '' : validation.message || 'Invalid query');
      
      if (!validation.isValid) {
        logger.warn('Query validation failed', {
          refId: query.refId,
          error: validation.message,
        });
      }
    } catch (error) {
      logger.error('Error validating query', error as Error, {
        refId: query.refId,
      });
      setValidationError('Error validating query');
    } finally {
      setIsValidating(false);
    }
  }, [query.refId]);

  // Update rawNRQL when query changes externally (but not when user is typing)
  useEffect(() => {
    if (!isUserTypingRef.current && query.queryText !== rawNRQL) {
      setRawNRQL(query.queryText || '');
      validateQuery(query.queryText || '');
    }
  }, [query.queryText, rawNRQL, validateQuery]);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
      }
    };
  }, []);

  /**
   * Handles changes to the raw NRQL text
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
    
    // For time integration, we need to be careful not to modify the text that's being typed
    // Only apply time integration when the user stops typing
    const finalQuery = queryText; // Keep the raw user input
    
    const updatedQuery = { ...query, queryText: finalQuery, useGrafanaTime };
    onChange(updatedQuery);
    
    // Debounced validation
    setTimeout(() => {
      const queryToValidate = useGrafanaTime ? 
        buildNRQLWithTimeIntegration(queryText, true) : 
        queryText;
      validateQuery(queryToValidate);
    }, 300);
  }, [query, onChange, validateQuery, useGrafanaTime]);

  /**
   * Handles changes from the query builder
   */
  const handleBuilderQueryChange = useCallback((queryText: string) => {
    setRawNRQL(queryText); // Keep in sync
    
    // Apply time integration if enabled
    const finalQuery = useGrafanaTime ? 
      buildNRQLWithTimeIntegration(queryText, true) : 
      queryText;
    
    const updatedQuery = { ...query, queryText: finalQuery, useGrafanaTime };
    onChange(updatedQuery);
    
    validateQuery(finalQuery);
  }, [query, onChange, validateQuery, useGrafanaTime]);

  /**
   * Handles toggling Grafana time picker integration
   */
  const handleTimeIntegrationToggle = useCallback((enabled: boolean) => {
    setUseGrafanaTime(enabled);
    
    let updatedQueryText = rawNRQL;
    
    if (enabled) {
      // Enable Grafana time integration
      updatedQueryText = buildNRQLWithTimeIntegration(rawNRQL, true);
    } else {
      // Remove Grafana time variables and use manual time clauses
      updatedQueryText = rawNRQL.replace(
        /WHERE\s+timestamp\s*>=\s*\$__from\s*AND\s*timestamp\s*<=\s*\$__to/gi,
        'SINCE 1 hour ago'
      );
    }
    
    const updatedQuery = { 
      ...query, 
      queryText: updatedQueryText, 
      useGrafanaTime: enabled 
    };
    
    onChange(updatedQuery);
    setRawNRQL(updatedQueryText);
    validateQuery(updatedQueryText);
    
    logger.debug('Time integration toggled', {
      refId: query.refId,
      enabled,
      hasGrafanaVars: hasGrafanaTimeVariables(updatedQueryText),
    });
  }, [rawNRQL, query, onChange, validateQuery]);

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
          ? 'SELECT count(*) FROM Transaction WHERE timestamp >= $__from AND timestamp <= $__to'
          : 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
        setRawNRQL(defaultQuery);
        onChange({ ...query, queryText: defaultQuery, useGrafanaTime });
        validateQuery(defaultQuery);
      }
    }
  }, [useQueryBuilder, query, onChange, validateQuery, useGrafanaTime]);

  /**
   * Handles running the query
   */
  const handleRunQuery = useCallback(() => {
    try {
      // Validate before running
      if (validationError) {
        logger.warn('Attempted to run invalid query', {
          refId: query.refId,
          error: validationError,
        });
        return;
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
  }, [validationError, query.refId, query.queryText, useGrafanaTime, onRunQuery]);

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
            disabled={!!validationError || isValidating || !query.queryText?.trim()}
            icon="play"
          >
            {isValidating ? 'Validating...' : 'Run'}
          </Button>
        </div>
      </div>

      {/* Validation Error Alert */}
      {validationError && (
        <Alert title="Query Validation Error" severity="error" style={{ marginBottom: '8px' }}>
          {validationError}
        </Alert>
      )}

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
          <TextArea
            value={rawNRQL}
            onChange={(e) => handleNRQLChange(e.currentTarget.value)}
            placeholder={useGrafanaTime 
              ? "SELECT count(*) FROM Transaction WHERE timestamp >= $__from AND timestamp <= $__to"
              : "SELECT count(*) FROM Transaction SINCE 1 hour ago"
            }
            rows={8}
            invalid={!!validationError}
            data-testid="nrql-textarea"
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
          
          {/* Query Status - only show for text editor */}
          {query.queryText && !validationError && (
            <div style={{ 
              fontSize: '11px', 
              color: '#28a745', 
              marginTop: '6px',
              display: 'flex',
              alignItems: 'center',
              gap: '4px'
            }}>
              <Icon name="check" size="xs" />
              Query is valid and ready to execute
              {useGrafanaTime && hasGrafanaTimeVariables(query.queryText) && 
                ' (with dashboard time integration)'
              }
            </div>
          )}
        </div>
      )}
    </div>
  );
}