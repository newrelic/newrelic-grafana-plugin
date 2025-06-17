import { renderHook, act } from '@testing-library/react';
import { useQueryBuilder } from '../useQueryBuilder';
import { DEFAULT_QUERY_COMPONENTS } from '../../types/query/constants';

describe('useQueryBuilder', () => {
  const mockOnChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('initialization', () => {
    it('should initialize with default components for empty query', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: '',
          onChange: mockOnChange,
        })
      );

      expect(result.current.queryComponents).toEqual({
        ...DEFAULT_QUERY_COMPONENTS,
        limit: 0,
      });
      expect(result.current.validationResult.isValid).toBe(true);
    });

    it('should initialize with default components for invalid query', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: 'invalid query',
          onChange: mockOnChange,
        })
      );

      expect(result.current.queryComponents).toEqual({
        ...DEFAULT_QUERY_COMPONENTS,
        limit: 0, // No limit when parsing invalid query
      });
    });

    it('should parse a simple count query correctly', () => {
      const query = 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: query,
          onChange: mockOnChange,
        })
      );

      expect(result.current.queryComponents).toEqual({
        ...DEFAULT_QUERY_COMPONENTS,
        aggregation: 'count',
        field: '',
        from: 'Transaction',
        since: '1 hour',
        limit: 0, // No limit when no LIMIT clause in query
      });
    });

    it('should parse a complex query with all clauses', () => {
      const query = 'SELECT average(duration) FROM Transaction WHERE appName = "MyApp" FACET host, name SINCE 2 hours ago UNTIL 1 hour ago TIMESERIES AUTO LIMIT 50';
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: query,
          onChange: mockOnChange,
        })
      );

      expect(result.current.queryComponents).toEqual({
        aggregation: 'average',
        field: 'duration',
        from: 'Transaction',
        where: 'appName = "MyApp"',
        facet: ['host', 'name'],
        since: '2 hours',
        until: '1 hour',
        timeseries: true,
        limit: 50,
      });
    });

    it('should parse SELECT * query as raw aggregation', () => {
      const query = 'SELECT * FROM Transaction SINCE 1 hour ago';
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: query,
          onChange: mockOnChange,
        })
      );

      expect(result.current.queryComponents.aggregation).toBe('raw');
      expect(result.current.queryComponents.field).toBe('');
    });

    it('should parse percentile function correctly', () => {
      const query = 'SELECT percentile(duration, 95) FROM Transaction SINCE 1 hour ago';
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: query,
          onChange: mockOnChange,
        })
      );

      expect(result.current.queryComponents.aggregation).toBe('percentile');
      expect(result.current.queryComponents.field).toBe('duration');
    });
  });

  describe('query building', () => {
    it('should build a basic count query', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: '',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({
          aggregation: 'count',
          from: 'Transaction',
          since: '1 hour',
        });
      });

      expect(mockOnChange).toHaveBeenCalledWith('SELECT count(*) FROM Transaction SINCE 1 hour ago');
    });

    it('should build query with aggregation function', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: '',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({
          aggregation: 'average',
          field: 'duration',
          from: 'Transaction',
          since: '30 minutes',
        });
      });

      expect(mockOnChange).toHaveBeenCalledWith('SELECT average(duration) FROM Transaction SINCE 30 minutes ago');
    });

    it('should build query with WHERE clause', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: '',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({
          aggregation: 'count',
          from: 'Transaction',
          where: 'appName = "MyApp"',
          since: '1 hour',
        });
      });

      expect(mockOnChange).toHaveBeenCalledWith('SELECT count(*) FROM Transaction WHERE appName = "MyApp" SINCE 1 hour ago');
    });

    it('should build query with FACET clause', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: '',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({
          aggregation: 'count',
          from: 'Transaction',
          facet: ['host', 'appName'],
          since: '1 hour',
        });
      });

      expect(mockOnChange).toHaveBeenCalledWith('SELECT count(*) FROM Transaction FACET host, appName SINCE 1 hour ago');
    });

    it('should build query with TIMESERIES', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: '',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({
          aggregation: 'count',
          from: 'Transaction',
          since: '1 hour',
          timeseries: true,
        });
      });

      expect(mockOnChange).toHaveBeenCalledWith('SELECT count(*) FROM Transaction SINCE 1 hour ago TIMESERIES AUTO');
    });

    it('should build query with LIMIT', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: '',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({
          aggregation: 'count',
          from: 'Transaction',
          since: '1 hour',
          limit: 25,
        });
      });

      expect(mockOnChange).toHaveBeenCalledWith('SELECT count(*) FROM Transaction SINCE 1 hour ago LIMIT 25');
    });

    it('should build complete query with all clauses', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: '',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({
          aggregation: 'average',
          field: 'duration',
          from: 'Transaction',
          where: 'appName = "MyApp"',
          facet: ['host'],
          since: '2 hours',
          until: '1 hour',
          timeseries: true,
          limit: 100,
        });
      });

      expect(mockOnChange).toHaveBeenCalledWith(
        'SELECT average(duration) FROM Transaction WHERE appName = "MyApp" FACET host SINCE 2 hours ago UNTIL 1 hour ago TIMESERIES AUTO LIMIT 100'
      );
    });

    it('should handle percentile function correctly', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: '',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({
          aggregation: 'percentile',
          field: 'duration',
          from: 'Transaction',
          since: '1 hour',
        });
      });

      expect(mockOnChange).toHaveBeenCalledWith('SELECT percentile(duration, 95) FROM Transaction SINCE 1 hour ago');
    });

    it('should handle raw aggregation (SELECT *)', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: '',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({
          aggregation: 'raw',
          from: 'Transaction',
          since: '1 hour',
        });
      });

      expect(mockOnChange).toHaveBeenCalledWith('SELECT * FROM Transaction SINCE 1 hour ago');
    });
  });

  describe('component updates', () => {
    it('should update single component', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: 'SELECT count(*) FROM Transaction SINCE 1 hour ago',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({ from: 'Span' });
      });

      expect(result.current.queryComponents.from).toBe('Span');
      expect(result.current.queryComponents.aggregation).toBe('count'); // Other components unchanged
    });

    it('should update multiple components at once', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: 'SELECT count(*) FROM Transaction SINCE 1 hour ago',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({
          aggregation: 'average',
          field: 'duration',
          from: 'Span',
        });
      });

      expect(result.current.queryComponents.aggregation).toBe('average');
      expect(result.current.queryComponents.field).toBe('duration');
      expect(result.current.queryComponents.from).toBe('Span');
    });

    it('should handle empty facet array correctly', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: 'SELECT count(*) FROM Transaction FACET host SINCE 1 hour ago',
          onChange: mockOnChange,
        })
      );

      act(() => {
        result.current.updateComponents({ facet: [] });
      });

      expect(result.current.queryComponents.facet).toEqual([]);
    });
  });

  describe('external query updates', () => {
    it('should update components when external query changes', () => {
      const { result, rerender } = renderHook(
        ({ query }) =>
          useQueryBuilder({
            initialQuery: query,
            onChange: mockOnChange,
          }),
        {
          initialProps: { query: 'SELECT count(*) FROM Transaction SINCE 1 hour ago' },
        }
      );

      expect(result.current.queryComponents.from).toBe('Transaction');

      rerender({ query: 'SELECT count(*) FROM Span SINCE 2 hours ago' });

      expect(result.current.queryComponents.from).toBe('Span');
      expect(result.current.queryComponents.since).toBe('2 hours');
    });
  });

  describe('edge cases', () => {
    it('should handle null/undefined initialQuery gracefully', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: null as any,
          onChange: mockOnChange,
        })
      );

      expect(result.current.queryComponents).toEqual({
        ...DEFAULT_QUERY_COMPONENTS,
        limit: 0, // Empty queries start with no limit
      });
    });

    it('should handle malformed NRQL gracefully', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: 'SELECTROM',
          onChange: mockOnChange,
        })
      );

      expect(result.current.queryComponents).toEqual({
        ...DEFAULT_QUERY_COMPONENTS,
        limit: 0, // No limit when parsing malformed query
      });
    });

    it('should handle queries with missing required clauses', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: 'SELECT count(*)',
          onChange: mockOnChange,
        })
      );

      expect(result.current.queryComponents.aggregation).toBe('count');
      expect(result.current.queryComponents.from).toBe('Transaction'); // Should fall back to default
    });

    it('should handle invalid limit values', () => {
      const { result } = renderHook(() =>
        useQueryBuilder({
          initialQuery: 'SELECT count(*) FROM Transaction LIMIT abc',
          onChange: mockOnChange,
        })
      );

      expect(result.current.queryComponents.limit).toBe(0); // Should not fall back to default when invalid
    });
  });

  describe('NRQL parsing and building', () => {
    it('should parse NRQL queries with aggregations correctly', () => {
      const { result } = renderHook(() => useQueryBuilder({
        initialQuery: 'SELECT average(duration) FROM Transaction WHERE appName = "test" SINCE 1 hour ago',
        onChange: jest.fn(),
        useGrafanaTime: false
      }));

      expect(result.current.queryComponents.aggregation).toBe('average');
      expect(result.current.queryComponents.field).toBe('duration');
      expect(result.current.queryComponents.from).toBe('Transaction');
      expect(result.current.queryComponents.where).toBe('appName = "test"');
      expect(result.current.queryComponents.since).toBe('1 hour');
    });

    it('should handle Grafana time variables without duplicating WHERE clauses', () => {
      const { result } = renderHook(() => useQueryBuilder({
        initialQuery: 'SELECT average(duration) FROM Transaction WHERE timestamp >= $__from AND timestamp <= $__to LIMIT 100',
        onChange: jest.fn(),
        useGrafanaTime: true
      }));

      expect(result.current.queryComponents.aggregation).toBe('average');
      expect(result.current.queryComponents.field).toBe('duration');
      expect(result.current.queryComponents.from).toBe('Transaction');
      expect(result.current.queryComponents.where).toBe(''); // Should be cleaned
      expect(result.current.queryComponents.limit).toBe(100);
    });

    it('should handle mixed WHERE clauses with Grafana time variables', () => {
      const { result } = renderHook(() => useQueryBuilder({
        initialQuery: 'SELECT sum(duration) FROM Transaction WHERE appName = "test" AND timestamp >= $__from AND timestamp <= $__to',
        onChange: jest.fn(),
        useGrafanaTime: true
      }));

      expect(result.current.queryComponents.aggregation).toBe('sum');
      expect(result.current.queryComponents.field).toBe('duration');
      expect(result.current.queryComponents.where).toBe('appName = "test"'); // Grafana time vars should be removed
    });

    it('should build query correctly with useGrafanaTime', () => {
      const onChange = jest.fn();
      const { result } = renderHook(() => useQueryBuilder({
        initialQuery: 'SELECT count(*) FROM Transaction',
        onChange,
        useGrafanaTime: true
      }));

      act(() => {
        result.current.updateComponents({ where: 'appName = "test"' });
      });

      // Should build query with Grafana time variables
      expect(onChange).toHaveBeenCalledWith(
        'SELECT count(*) FROM Transaction WHERE appName = "test" AND timestamp >= $__from AND timestamp <= $__to'
      );
    });

    it('should validate invalid aggregations', () => {
      const { result } = renderHook(() => useQueryBuilder({
        initialQuery: 'SELECT xyz(duration) FROM Transaction',
        onChange: jest.fn(),
        useGrafanaTime: false
      }));

      expect(result.current.queryComponents.aggregation).toBe('xyz'); // Should parse even if invalid
      expect(result.current.queryComponents.field).toBe('duration');
    });

    it('should handle percentile aggregation correctly', () => {
      const { result } = renderHook(() => useQueryBuilder({
        initialQuery: 'SELECT percentile(duration, 95) FROM Transaction',
        onChange: jest.fn(),
        useGrafanaTime: false
      }));

      expect(result.current.queryComponents.aggregation).toBe('percentile');
      expect(result.current.queryComponents.field).toBe('duration');
    });

    it('should handle count(*) correctly', () => {
      const { result } = renderHook(() => useQueryBuilder({
        initialQuery: 'SELECT count(*) FROM Transaction',
        onChange: jest.fn(),
        useGrafanaTime: false
      }));

      expect(result.current.queryComponents.aggregation).toBe('count');
      expect(result.current.queryComponents.field).toBe('');
    });
  });
}); 