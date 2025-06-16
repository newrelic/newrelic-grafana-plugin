import React, { useState, useEffect } from 'react';
import { QueryEditorProps } from '@grafana/data';
import { InlineField, InlineFieldRow, TextArea, Button } from '@grafana/ui';
import { DataSource } from '../datasource';
import { NewRelicQuery, NewRelicDataSourceOptions } from '../types';
import { NRQLQueryBuilder } from './NRQLQueryBuilder';

type Props = QueryEditorProps<DataSource, NewRelicQuery, NewRelicDataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const [useQueryBuilder, setUseQueryBuilder] = useState(false);
  
  // Migration logic: handle old queries that might have 'nrql' field
  const migrateQuery = (q: any): NewRelicQuery => {
    if (q.nrql && !q.queryText) {
      console.log('QueryEditor: Migrating old nrql field to queryText:', q.nrql);
      const migratedQuery = { ...q, queryText: q.nrql };
      delete migratedQuery.nrql; // Remove old field
      onChange(migratedQuery); // Update the query immediately
      return migratedQuery;
    }
    
    if (q.nrql && q.queryText) {
      console.log('QueryEditor: Removing duplicate nrql field, keeping queryText:', q.queryText);
      const cleanedQuery = { ...q };
      delete cleanedQuery.nrql; // Remove old field
      onChange(cleanedQuery); // Update the query immediately
      return cleanedQuery;
    }
    
    return q;
  };

  // Apply migration on component mount and when query changes
  const currentQuery = migrateQuery(query);
  const [rawNRQL, setRawNRQL] = useState(currentQuery.queryText || '');

  // Update rawNRQL when query changes externally
  useEffect(() => {
    const migratedQuery = migrateQuery(query);
    if (migratedQuery.queryText !== rawNRQL) {
      setRawNRQL(migratedQuery.queryText || '');
    }
  }, [query.queryText, query]);

  const handleNRQLChange = (queryText: string) => {
    setRawNRQL(queryText);
    onChange({ ...currentQuery, queryText });
  };

  const handleBuilderQueryChange = (queryText: string) => {
    setRawNRQL(queryText); // Update rawNRQL to keep in sync
    onChange({ ...currentQuery, queryText });
  };

  const toggleQueryBuilder = () => {
    setUseQueryBuilder(!useQueryBuilder);
    if (!useQueryBuilder) {
      // When switching to query builder, ensure the query is in a valid format
      if (!currentQuery.queryText || currentQuery.queryText.trim() === '') {
        const defaultQuery = 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
        setRawNRQL(defaultQuery);
        onChange({ ...currentQuery, queryText: defaultQuery });
      }
    }
  };

  return (
    <div>
      <InlineFieldRow>
        <InlineField>
          <Button
            variant={useQueryBuilder ? 'primary' : 'secondary'}
            onClick={toggleQueryBuilder}
          >
            {useQueryBuilder ? 'Switch to Text Editor' : 'Use Query Builder'}
          </Button>
        </InlineField>
        <InlineField>
          <Button
            variant="primary"
            onClick={onRunQuery}
          >
            Run Query
          </Button>
        </InlineField>
      </InlineFieldRow>

      {useQueryBuilder ? (
        <NRQLQueryBuilder
          value={currentQuery.queryText || ''}
          onChange={handleBuilderQueryChange}
          onRunQuery={onRunQuery}
        />
      ) : (
        <div style={{ position: 'relative' }}>
          <TextArea
            value={rawNRQL}
            onChange={e => handleNRQLChange(e.currentTarget.value)}
            placeholder="Enter NRQL query..."
            rows={5}
          />
          <Button onClick={onRunQuery} icon="play" variant="primary" style={{ marginTop: 8 }}>
            Run Query
          </Button>
        </div>
      )}
    </div>
  );
}