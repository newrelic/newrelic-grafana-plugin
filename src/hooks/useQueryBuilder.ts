import { useState, useEffect, useCallback, useRef } from 'react';
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

// Parse NRQL query into components (standalone function)
function parseNRQLQuery(query: string): QueryComponents {
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
}

// Build NRQL query from components (standalone function)
function buildNRQLQuery(components: QueryComponents): string {
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
}

export function useQueryBuilder({ initialQuery, onChange }: UseQueryBuilderProps): UseQueryBuilderResult {
  // Start with defaults that don't include limit unless parsing a real query
  const [queryComponents, setQueryComponents] = useState<QueryComponents>(() => {
    if (initialQuery) {
      return parseNRQLQuery(initialQuery);
    }
    // For empty initial query, start with defaults but no limit
    return {
      ...DEFAULT_QUERY_COMPONENTS,
      limit: 0, // No limit initially
    };
  });
  const [validationResult, setValidationResult] = useState<QueryValidationResult>({ isValid: true });
  
  const lastQueryRef = useRef<string>(initialQuery || '');
  const isUpdatingRef = useRef<boolean>(false);

  // Update components when external query changes
  useEffect(() => {
    const currentQuery = initialQuery || '';
    if (currentQuery !== lastQueryRef.current && !isUpdatingRef.current) {
      isUpdatingRef.current = true;
      lastQueryRef.current = currentQuery;
      
      const parsedComponents = currentQuery ? parseNRQLQuery(currentQuery) : {
        ...DEFAULT_QUERY_COMPONENTS,
        limit: 0,
      };
      
      setQueryComponents(parsedComponents);
      
      // Reset the flag asynchronously
      setTimeout(() => {
        isUpdatingRef.current = false;
      }, 0);
    }
  }, [initialQuery]);

  // Update components
  const updateComponents = useCallback((update: Partial<QueryComponents>) => {
    setQueryComponents(prev => {
      const newComponents = { ...prev, ...update };
      // Trigger query rebuild immediately
      const newQuery = buildNRQLQuery(newComponents);
      if (newQuery !== lastQueryRef.current) {
        lastQueryRef.current = newQuery;
        onChange(newQuery);
      }
      return newComponents;
    });
  }, [onChange]);

  return {
    queryComponents,
    updateComponents,
    validationResult,
  };
} 