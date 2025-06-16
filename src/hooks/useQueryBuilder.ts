import { useState, useEffect, useCallback } from 'react';
import { QueryComponents, QueryValidationResult } from '../types/query/types';
import { DEFAULT_QUERY_COMPONENTS } from '../types/query/constants';
import { logger } from '../utils/logger';

interface UseQueryBuilderProps {
  initialQuery: string;
  onChange: (query: string) => void;
}

interface UseQueryBuilderResult {
  queryComponents: QueryComponents;
  updateComponents: (update: Partial<QueryComponents>) => void;
  validationResult: QueryValidationResult;
}

export function useQueryBuilder({ initialQuery, onChange }: UseQueryBuilderProps): UseQueryBuilderResult {
  const [queryComponents, setQueryComponents] = useState<QueryComponents>(() => parseQuery(initialQuery));
  const [isUpdatingFromQuery, setIsUpdatingFromQuery] = useState(false);
  const [validationResult, setValidationResult] = useState<QueryValidationResult>({ isValid: true });

  // Parse NRQL query into components
  const parseQuery = useCallback((query: string): QueryComponents => {
    if (!query || typeof query !== 'string') {
      return DEFAULT_QUERY_COMPONENTS;
    }

    try {
      const components: QueryComponents = { ...DEFAULT_QUERY_COMPONENTS };
      
      // Parse SELECT clause
      const selectMatch = query.match(/SELECT\s+(.+?)\s+FROM/i);
      if (selectMatch && selectMatch[1]) {
        const selectClause = selectMatch[1].trim();
        
        if (selectClause === '*') {
          components.aggregation = 'raw';
          components.field = '';
        } else if (selectClause === 'count(*)') {
          components.aggregation = 'count';
          components.field = '';
        } else {
          const funcMatch = selectClause.match(/(\w+)\(([^)]+)\)/);
          if (funcMatch) {
            components.aggregation = funcMatch[1];
            components.field = funcMatch[2].trim();
          }
        }
      }

      // Parse FROM clause
      const fromMatch = query.match(/FROM\s+(\w+)/i);
      if (fromMatch && fromMatch[1]) {
        components.from = fromMatch[1];
      }

      // Parse WHERE clause
      const whereMatch = query.match(/WHERE\s+(.+?)(?:\s+FACET|\s+SINCE|\s+UNTIL|\s+TIMESERIES|\s+LIMIT|$)/i);
      if (whereMatch && whereMatch[1]) {
        components.where = whereMatch[1].trim();
      }

      // Parse FACET clause
      const facetMatch = query.match(/FACET\s+(.+?)(?:\s+SINCE|\s+UNTIL|\s+TIMESERIES|\s+LIMIT|$)/i);
      if (facetMatch && facetMatch[1]) {
        components.facet = facetMatch[1].split(',').map(s => s.trim()).filter(Boolean);
      }

      // Parse SINCE clause
      const sinceMatch = query.match(/SINCE\s+(.+?)\s+ago/i);
      if (sinceMatch && sinceMatch[1]) {
        components.since = sinceMatch[1].trim();
      }

      // Parse UNTIL clause
      const untilMatch = query.match(/UNTIL\s+(.+?)\s+ago/i);
      if (untilMatch && untilMatch[1]) {
        components.until = untilMatch[1].trim();
      }

      // Parse TIMESERIES
      components.timeseries = /TIMESERIES/i.test(query);

      // Parse LIMIT
      const limitMatch = query.match(/LIMIT\s+(\d+)/i);
      if (limitMatch && limitMatch[1]) {
        const limitValue = parseInt(limitMatch[1], 10);
        if (!isNaN(limitValue)) {
          components.limit = limitValue;
        }
      }

      return components;
    } catch (error) {
      logger.error('Error parsing NRQL query:', error as Error);
      return DEFAULT_QUERY_COMPONENTS;
    }
  }, []);

  // Build NRQL query from components
  const buildQuery = useCallback((components: QueryComponents): string => {
    if (!components || !components.aggregation || !components.from) {
      return 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
    }

    let selectClause = '';
    if (components.aggregation === 'count') {
      selectClause = 'count(*)';
    } else if (components.aggregation === 'raw') {
      selectClause = '*';
    } else {
      const field = components.field || 'duration';
      if (components.aggregation === 'percentile') {
        selectClause = `percentile(${field}, 95)`;
      } else {
        selectClause = `${components.aggregation}(${field})`;
      }
    }

    let query = `SELECT ${selectClause} FROM ${components.from}`;
    
    if (components.where && components.where.trim()) {
      query += ` WHERE ${components.where}`;
    }
    
    if (components.facet && components.facet.length > 0) {
      query += ` FACET ${components.facet.join(', ')}`;
    }
    
    if (components.since && components.since.trim()) {
      query += ` SINCE ${components.since} ago`;
    }
    
    if (components.until && components.until.trim()) {
      query += ` UNTIL ${components.until} ago`;
    }
    
    if (components.timeseries) {
      query += ' TIMESERIES AUTO';
    }
    
    if (components.limit && components.limit > 0) {
      query += ` LIMIT ${components.limit}`;
    }
    
    return query;
  }, []);

  // Update query when components change
  useEffect(() => {
    if (!isUpdatingFromQuery && queryComponents) {
      const newQuery = buildQuery(queryComponents);
      if (newQuery !== initialQuery) {
        onChange(newQuery);
      }
    }
  }, [queryComponents, onChange, initialQuery, buildQuery, isUpdatingFromQuery]);

  // Update components when external query changes
  useEffect(() => {
    if (initialQuery !== buildQuery(queryComponents)) {
      setIsUpdatingFromQuery(true);
      const parsedComponents = parseQuery(initialQuery);
      setQueryComponents(parsedComponents);
      setTimeout(() => setIsUpdatingFromQuery(false), 0);
    }
  }, [initialQuery, queryComponents, parseQuery, buildQuery]);

  // Update components
  const updateComponents = useCallback((update: Partial<QueryComponents>) => {
    setQueryComponents(prev => {
      if (!prev) {
        return prev;
      }
      return { ...prev, ...update };
    });
  }, []);

  return {
    queryComponents,
    updateComponents,
    validationResult,
  };
} 