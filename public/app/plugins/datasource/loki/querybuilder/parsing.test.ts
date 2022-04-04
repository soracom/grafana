import { buildVisualQueryFromString } from './parsing';
import { LokiVisualQuery } from './types';

describe('buildVisualQueryFromString', () => {
  it('parses simple query with label-values', () => {
    expect(buildVisualQueryFromString('{app="frontend"}')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [],
      })
    );
  });

  it('parses query with multiple label-values pairs', () => {
    expect(buildVisualQueryFromString('{app="frontend", instance!="1"}')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
          {
            op: '!=',
            value: '1',
            label: 'instance',
          },
        ],
        operations: [],
      })
    );
  });

  it('parses query with line filter', () => {
    expect(buildVisualQueryFromString('{app="frontend"} |= "line"')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: '__line_contains', params: ['line'] }],
      })
    );
  });

  it('parses query with line filters and escaped characters', () => {
    expect(buildVisualQueryFromString('{app="frontend"} |= "\\\\line"')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: '__line_contains', params: ['\\line'] }],
      })
    );
  });

  it('parses query with matcher label filter', () => {
    expect(buildVisualQueryFromString('{app="frontend"} | bar="baz"')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: '__label_filter', params: ['bar', '=', 'baz'] }],
      })
    );
  });

  it('parses query with number label filter', () => {
    expect(buildVisualQueryFromString('{app="frontend"} | bar >= 8')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: '__label_filter', params: ['bar', '>=', '8'] }],
      })
    );
  });

  it('parses query with no pipe errors filter', () => {
    expect(buildVisualQueryFromString('{app="frontend"} | __error__=""')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: '__label_filter_no_errors', params: [] }],
      })
    );
  });

  it('parses query with with unit label filter', () => {
    expect(buildVisualQueryFromString('{app="frontend"} | bar < 8mb')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: '__label_filter', params: ['bar', '<', '8mb'] }],
      })
    );
  });

  it('parses query with with parser', () => {
    expect(buildVisualQueryFromString('{app="frontend"} | json')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: 'json', params: [] }],
      })
    );
  });

  it('parses metrics query with function', () => {
    expect(buildVisualQueryFromString('rate({app="frontend"} | json [5m])')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [
          { id: 'json', params: [] },
          { id: 'rate', params: ['5m'] },
        ],
      })
    );
  });

  it('parses metrics query with function and aggregation', () => {
    expect(buildVisualQueryFromString('sum(rate({app="frontend"} | json [5m]))')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [
          { id: 'json', params: [] },
          { id: 'rate', params: ['5m'] },
          { id: 'sum', params: [] },
        ],
      })
    );
  });

  it('parses metrics query with function and aggregation and filters', () => {
    expect(buildVisualQueryFromString('sum(rate({app="frontend"} |~ `abc` | json | bar="baz" [5m]))')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [
          { id: '__line_matches_regex', params: ['abc'] },
          { id: 'json', params: [] },
          { id: '__label_filter', params: ['bar', '=', 'baz'] },
          { id: 'rate', params: ['5m'] },
          { id: 'sum', params: [] },
        ],
      })
    );
  });

  it('parses template variables in strings', () => {
    expect(buildVisualQueryFromString('{instance="$label_variable"}')).toEqual(
      noErrors({
        labels: [{ label: 'instance', op: '=', value: '$label_variable' }],
        operations: [],
      })
    );
  });

  it('parses metrics query with interval variables', () => {
    expect(buildVisualQueryFromString('rate({app="frontend"} [$__interval])')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: 'rate', params: ['$__interval'] }],
      })
    );
  });

  it('parses quantile queries', () => {
    expect(buildVisualQueryFromString(`quantile_over_time(0.99, {app="frontend"} [1m])`)).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: 'quantile_over_time', params: ['0.99', '1m'] }],
      })
    );
  });

  it('parses query with line format', () => {
    expect(buildVisualQueryFromString('{app="frontend"} | line_format "abc"')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: 'line_format', params: ['abc'] }],
      })
    );
  });

  it('parses query with label format', () => {
    expect(buildVisualQueryFromString('{app="frontend"} | label_format newLabel=oldLabel')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [{ id: 'label_format', params: ['newLabel', '=', 'oldLabel'] }],
      })
    );
  });

  it('parses query with multiple label format', () => {
    expect(buildVisualQueryFromString('{app="frontend"} | label_format newLabel=oldLabel, bar="baz"')).toEqual(
      noErrors({
        labels: [
          {
            op: '=',
            value: 'frontend',
            label: 'app',
          },
        ],
        operations: [
          { id: 'label_format', params: ['newLabel', '=', 'oldLabel'] },
          { id: 'label_format', params: ['bar', '=', 'baz'] },
        ],
      })
    );
  });

  it('parses binary query', () => {
    expect(buildVisualQueryFromString('rate({project="bar"}[5m]) / rate({project="foo"}[5m])')).toEqual(
      noErrors({
        labels: [{ op: '=', value: 'bar', label: 'project' }],
        operations: [{ id: 'rate', params: ['5m'] }],
        binaryQueries: [
          {
            operator: '/',
            query: {
              labels: [{ op: '=', value: 'foo', label: 'project' }],
              operations: [{ id: 'rate', params: ['5m'] }],
            },
          },
        ],
      })
    );
  });

  it('parses binary scalar query', () => {
    expect(buildVisualQueryFromString('rate({project="bar"}[5m]) / 2')).toEqual(
      noErrors({
        labels: [{ op: '=', value: 'bar', label: 'project' }],
        operations: [
          { id: 'rate', params: ['5m'] },
          { id: '__divide_by', params: [2] },
        ],
      })
    );
  });

  it('parses chained binary query', () => {
    expect(
      buildVisualQueryFromString('rate({project="bar"}[5m]) * 2 / rate({project="foo"}[5m]) + rate({app="test"}[1m])')
    ).toEqual(
      noErrors({
        labels: [{ op: '=', value: 'bar', label: 'project' }],
        operations: [
          { id: 'rate', params: ['5m'] },
          { id: '__multiply_by', params: [2] },
        ],
        binaryQueries: [
          {
            operator: '/',
            query: {
              labels: [{ op: '=', value: 'foo', label: 'project' }],
              operations: [{ id: 'rate', params: ['5m'] }],
              binaryQueries: [
                {
                  operator: '+',
                  query: {
                    labels: [{ op: '=', value: 'test', label: 'app' }],
                    operations: [{ id: 'rate', params: ['1m'] }],
                  },
                },
              ],
            },
          },
        ],
      })
    );
  });
});

function noErrors(query: LokiVisualQuery) {
  return {
    errors: [],
    query,
  };
}
