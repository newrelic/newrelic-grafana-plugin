import React, { useState } from 'react';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { Input, Button } from '@grafana/ui';
import { MyDataSourceOptions, MyQuery } from '../types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

export const QueryEditor: React.FC<Props> = ({ query, onChange, onRunQuery }) => {
  const [queryText, setQueryText] = useState(query.queryText || '');
  const [responseData] = useState<string | null>(null);

  const handleInputChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    setQueryText(value);
    onChange({ ...query, queryText: value });
  };

  const handleSubmit = () => {
    // Trigger query execution handled by Grafana QueryData infrastructure
    onRunQuery();
  };

  return (
    <div>
      <div className="gf-form">
        <Input
          value={queryText}
          onChange={handleInputChange}
          width={40}
          placeholder="Enter query"
        />
        <Button onClick={handleSubmit} variant="primary">
          Submit
        </Button>
      </div>
      {responseData && (
        <div style={{ marginTop: '10px', whiteSpace: 'pre-wrap' }}>
          <strong>Response Data:</strong>
          <pre>{responseData}</pre> {/* Display the response data */}
        </div>
      )}
    </div>
  );
};