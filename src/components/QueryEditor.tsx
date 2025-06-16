import React, { useState, useEffect, useCallback } from 'react';
import { QueryEditorProps } from '@grafana/data';
import { InlineField, InlineFieldRow, TextArea, Button, Alert } from '@grafana/ui';
import { DataSource } from '../datasource';
import { NewRelicQuery, NewRelicDataSourceOptions } from '../types';
import { NRQLQueryBuilder } from './NRQLQueryBuilder';
import { validateNrqlQuery } from '../utils/validation';
import { logger } from '../utils/logger';

type Props = QueryEditorProps<DataSource, NewRelicQuery, NewRelicDataSourceOptions>;

/**
 * Query editor component for New Relic NRQL queries
 * Provides both a visual query builder and raw text editor
 */
export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const [useQueryBuilder, setUseQueryBuilder] = useState(false);
  const [validationError, setValidationError] = useState<string>('');
  const [isValidating, setIsValidating] = useState(false);

  // Local state for the raw NRQL text
  const [rawNRQL, setRawNRQL] = useState(query.queryText || '');

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

  // Update rawNRQL when query changes externally
  useEffect(() => {
    if (query.queryText !== rawNRQL) {
      setRawNRQL(query.queryText || '');
      validateQuery(query.queryText || '');
    }
  }, [query.queryText, rawNRQL, validateQuery]);

  /**
   * Handles changes to the raw NRQL text
   */
  const handleNRQLChange = useCallback((queryText: string) => {
    setRawNRQL(queryText);
    
    const updatedQuery = { ...query, queryText };
    onChange(updatedQuery);
    
    // Debounced validation
    const timeoutId = setTimeout(() => {
      validateQuery(queryText);
    }, 300);

    return () => clearTimeout(timeoutId);
  }, [query, onChange, validateQuery]);

  /**
   * Handles changes from the query builder
   */
  const handleBuilderQueryChange = useCallback((queryText: string) => {
    setRawNRQL(queryText); // Keep in sync
    
    const updatedQuery = { ...query, queryText };
    onChange(updatedQuery);
    
    validateQuery(queryText);
  }, [query, onChange, validateQuery]);

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
        const defaultQuery = 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
        setRawNRQL(defaultQuery);
        onChange({ ...query, queryText: defaultQuery });
        validateQuery(defaultQuery);
      }
    }
  }, [useQueryBuilder, query, onChange, validateQuery]);

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

      logger.debug('Running query', { refId: query.refId });
      onRunQuery();
    } catch (error) {
      logger.error('Error running query', error as Error, {
        refId: query.refId,
      });
    }
  }, [validationError, query.refId, onRunQuery]);

  return (
    <div>
      {/* Query Editor Controls */}
      <InlineFieldRow>
        <InlineField>
          <Button
            variant={useQueryBuilder ? 'primary' : 'secondary'}
            onClick={toggleQueryBuilder}
            aria-label={`Switch to ${useQueryBuilder ? 'text editor' : 'query builder'}`}
            data-testid="query-mode-toggle"
          >
            {useQueryBuilder ? 'Switch to Text Editor' : 'Use Query Builder'}
          </Button>
        </InlineField>
        <InlineField>
          <Button
            variant="primary"
            onClick={handleRunQuery}
            disabled={!!validationError || isValidating}
            aria-label="Run NRQL query"
            data-testid="run-query-button"
          >
            {isValidating ? 'Validating...' : 'Run Query'}
          </Button>
        </InlineField>
      </InlineFieldRow>

      {/* Validation Error Alert */}
      {validationError && (
        <Alert title="Query Validation Error" severity="error">
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
          />
        </div>
      ) : (
        <div role="region" aria-label="NRQL Text Editor" style={{ position: 'relative' }}>
          <TextArea
            value={rawNRQL}
            onChange={e => handleNRQLChange(e.currentTarget.value)}
            placeholder="Enter NRQL query... (e.g., SELECT count(*) FROM Transaction SINCE 1 hour ago)"
            rows={5}
            aria-label="NRQL Query Text"
            aria-describedby="nrql-help"
            aria-invalid={!!validationError}
            data-testid="nrql-textarea"
          />
          
          {/* Help Text */}
          <div id="nrql-help" style={{ fontSize: '12px', color: '#6c757d', marginTop: '8px' }}>
            Enter your NRQL query. Use Grafana template variables like $__timeFilter() for dynamic queries.
          </div>
          
          <Button 
            onClick={handleRunQuery} 
            icon="play" 
            variant="primary" 
            style={{ marginTop: 8 }}
            disabled={!!validationError || isValidating}
            aria-label="Run NRQL query"
            data-testid="run-query-text-button"
          >
            {isValidating ? 'Validating...' : 'Run Query'}
          </Button>
        </div>
      )}

      {/* Query Information */}
      {query.queryText && !validationError && (
        <div style={{ fontSize: '12px', color: '#28a745', marginTop: '8px' }}>
          ✓ Query is valid and ready to execute
        </div>
      )}
    </div>
  );
}