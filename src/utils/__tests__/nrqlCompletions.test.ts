import { getContextFromLine, registerNrqlCompletionProvider, _resetRegistration } from '../nrqlCompletions';

// Helper to build a mock Monaco instance and extract the registered provider
function createMockMonaco() {
  const registerFn = jest.fn();
  const mockMonaco = {
    languages: {
      registerCompletionItemProvider: registerFn,
      CompletionItemKind: {
        Function: 1,
        Class: 5,
        Field: 4,
        Keyword: 17,
        Unit: 12,
        Variable: 6,
        Value: 11,
        Constant: 14,
      },
      CompletionItemInsertTextRule: {
        InsertAsSnippet: 4,
      },
    },
  };
  return { mockMonaco, registerFn };
}

function getProvider(registerFn: jest.Mock) {
  return registerFn.mock.calls[0][1];
}

function getSuggestions(provider: any, lineText: string, column?: number) {
  const col = column ?? lineText.length + 1;
  const model = {
    getLineContent: () => lineText,
    getWordUntilPosition: () => ({ startColumn: col, endColumn: col }),
  };
  const position = { lineNumber: 1, column: col };
  return provider.provideCompletionItems(model, position).suggestions;
}

describe('getContextFromLine', () => {
  it('returns "select" after SELECT keyword', () => {
    expect(getContextFromLine('SELECT ')).toBe('select');
  });

  it('returns "select" after comma in SELECT clause', () => {
    expect(getContextFromLine('SELECT count(*), ')).toBe('select');
  });

  it('returns "from" after FROM keyword', () => {
    expect(getContextFromLine('SELECT count(*) FROM ')).toBe('from');
  });

  it('returns "where" after WHERE keyword', () => {
    expect(getContextFromLine('SELECT count(*) FROM Transaction WHERE ')).toBe('where');
  });

  it('returns "where" after AND keyword', () => {
    expect(getContextFromLine("SELECT count(*) FROM Transaction WHERE appName = 'test' AND ")).toBe('where');
  });

  it('returns "where" after OR keyword', () => {
    expect(getContextFromLine('SELECT count(*) FROM Transaction WHERE x = 1 OR ')).toBe('where');
  });

  it('returns "facet" after FACET keyword', () => {
    expect(getContextFromLine('SELECT count(*) FROM Transaction FACET ')).toBe('facet');
  });

  it('returns "since" after SINCE keyword', () => {
    expect(getContextFromLine('SELECT count(*) FROM Transaction SINCE ')).toBe('since');
  });

  it('returns "since" after UNTIL keyword', () => {
    expect(getContextFromLine('SELECT count(*) FROM Transaction SINCE 1 hour ago UNTIL ')).toBe('since');
  });

  it('returns "limit" after LIMIT keyword', () => {
    expect(getContextFromLine('SELECT count(*) FROM Transaction LIMIT ')).toBe('limit');
  });

  it('returns "join" after JOIN( keyword', () => {
    expect(getContextFromLine('FROM Transaction JOIN (')).toBe('join');
  });

  it('returns "join" after INNER JOIN( keyword', () => {
    expect(getContextFromLine('FROM Transaction INNER JOIN (')).toBe('join');
  });

  it('returns "join" after LEFT JOIN( keyword', () => {
    expect(getContextFromLine('FROM Transaction LEFT JOIN (')).toBe('join');
  });

  it('returns "default" for empty string', () => {
    expect(getContextFromLine('')).toBe('default');
  });

  it('returns "default" for partial keyword typing', () => {
    expect(getContextFromLine('SEL')).toBe('default');
  });

  it('is case-insensitive', () => {
    expect(getContextFromLine('select ')).toBe('select');
    expect(getContextFromLine('from ')).toBe('from');
    expect(getContextFromLine('Select count(*) From ')).toBe('from');
  });
});

describe('registerNrqlCompletionProvider', () => {
  beforeEach(() => {
    _resetRegistration();
  });

  it('registers a completion provider on the sql language', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);

    expect(registerFn).toHaveBeenCalledTimes(1);
    expect(registerFn).toHaveBeenCalledWith('sql', expect.objectContaining({
      triggerCharacters: [' ', ',', '$', '('],
      provideCompletionItems: expect.any(Function),
    }));
  });

  it('only registers once on the same Monaco instance', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    registerNrqlCompletionProvider(mockMonaco);

    expect(registerFn).toHaveBeenCalledTimes(1);
  });

  it('registers on each different Monaco instance (Dashboard vs Explore)', () => {
    const monaco1 = createMockMonaco();
    const monaco2 = createMockMonaco();

    registerNrqlCompletionProvider(monaco1.mockMonaco);
    registerNrqlCompletionProvider(monaco2.mockMonaco);

    expect(monaco1.registerFn).toHaveBeenCalledTimes(1);
    expect(monaco2.registerFn).toHaveBeenCalledTimes(1);
  });

  // ── Default context ──

  it('returns keyword suggestions for default context', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, '');

    expect(suggestions.length).toBeGreaterThan(0);
    expect(suggestions[0].label).toBe('SELECT');
    const labels = suggestions.map((s: any) => s.label);
    expect(labels).toContain('FROM');
    expect(labels).toContain('WHERE');
    expect(labels).toContain('FACET');
    expect(labels).toContain('JOIN');
    expect(labels).toContain('INNER JOIN');
    expect(labels).toContain('LEFT JOIN');
    expect(labels).toContain('SLIDE BY');
    expect(labels).toContain('PREDICT');
    expect(labels).toContain('SHOW EVENT TYPES');
    expect(labels).toContain('FACET CASES');
    expect(labels).toContain('RLIKE');
    expect(labels).toContain('OFFSET');
  });

  // ── SELECT context ──

  it('returns aggregator functions first after SELECT', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, 'SELECT ');

    // First real function should be count
    const fnSuggestions = suggestions.filter((s: any) => s.kind === 1);
    expect(fnSuggestions[0].label).toBe('count');
    expect(fnSuggestions[0].insertText).toBe('count(${1:*})');
  });

  it('includes non-aggregator functions after SELECT', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, 'SELECT ');
    const labels = suggestions.map((s: any) => s.label);

    expect(labels).toContain('capture');
    expect(labels).toContain('aparse');
    expect(labels).toContain('if');
    expect(labels).toContain('numeric');
    expect(labels).toContain('concat');
    expect(labels).toContain('abs');
    expect(labels).toContain('lower');
    expect(labels).toContain('toDatetime');
    expect(labels).toContain('convert');
    expect(labels).toContain('jparse');
  });

  it('includes new aggregator functions (median, derivative, cdfPercentage, etc.)', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, 'SELECT ');
    const labels = suggestions.map((s: any) => s.label);

    expect(labels).toContain('median');
    expect(labels).toContain('derivative');
    expect(labels).toContain('cdfPercentage');
    expect(labels).toContain('getCdfCount');
    expect(labels).toContain('keyset');
    expect(labels).toContain('percentage');
    expect(labels).toContain('buckets');
    expect(labels).toContain('accountId');
    expect(labels).toContain('eventType');
  });

  it('includes * wildcard in SELECT suggestions', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, 'SELECT ');
    const labels = suggestions.map((s: any) => s.label);

    expect(labels).toContain('*');
  });

  // ── FROM context ──

  it('returns all event types after FROM', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, 'SELECT count(*) FROM ');
    const labels = suggestions.map((s: any) => s.label);

    expect(labels[0]).toBe('Transaction');
    // New event types
    expect(labels).toContain('JavaScriptError');
    expect(labels).toContain('AjaxRequest');
    expect(labels).toContain('MobileRequest');
    expect(labels).toContain('K8sContainerSample');
    expect(labels).toContain('K8sPodSample');
    expect(labels).toContain('K8sNodeSample');
    expect(labels).toContain('SyntheticRequest');
    expect(labels).toContain('AwsLambdaInvocation');
    expect(labels).toContain('NrIntegrationError');
    expect(labels).toContain('StorageSample');
  });

  // ── WHERE context ──

  it('returns fields and non-aggregator functions after WHERE', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, 'SELECT count(*) FROM Transaction WHERE ');
    const labels = suggestions.map((s: any) => s.label);

    // Fields
    expect(labels).toContain('appName');
    expect(labels).toContain('service.name');
    expect(labels).toContain('entity.name');
    expect(labels).toContain('traceId');
    expect(labels).toContain('spanId');
    expect(labels).toContain('error.message');
    expect(labels).toContain('cpuPercent');
    expect(labels).toContain('clusterName');
    expect(labels).toContain('podName');
    expect(labels).toContain('pageUrl');
    expect(labels).toContain('city');
    // Non-aggregator functions are also valid in WHERE
    expect(labels).toContain('capture');
    expect(labels).toContain('numeric');
    expect(labels).toContain('if');
  });

  // ── FACET context ──

  it('returns fields and functions after FACET', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, 'SELECT count(*) FROM Transaction FACET ');
    const labels = suggestions.map((s: any) => s.label);

    expect(labels).toContain('appName');
    expect(labels).toContain('capture');
    expect(labels).toContain('aparse');
  });

  // ── SINCE context ──

  it('returns full time expressions after SINCE', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, 'SELECT count(*) FROM Transaction SINCE ');
    const labels = suggestions.map((s: any) => s.label);

    expect(labels).toContain('1 hour ago');
    expect(labels).toContain('1 day ago');
    expect(labels).toContain('1 week ago');
    expect(labels).toContain('today');
    expect(labels).toContain('yesterday');
    expect(labels).toContain('$__from');
    expect(labels).toContain('$__to');
    expect(labels).toContain('$__interval');
  });

  // ── LIMIT context ──

  it('returns limit values including 5000 and MAX', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, 'SELECT count(*) FROM Transaction LIMIT ');
    const labels = suggestions.map((s: any) => s.label);

    expect(labels).toContain('10');
    expect(labels).toContain('5000');
    expect(labels).toContain('MAX');
  });

  // ── JOIN context ──

  it('returns FROM and event types after JOIN(', () => {
    const { mockMonaco, registerFn } = createMockMonaco();
    registerNrqlCompletionProvider(mockMonaco);
    const provider = getProvider(registerFn);
    const suggestions = getSuggestions(provider, 'FROM Transaction JOIN (');
    const labels = suggestions.map((s: any) => s.label);

    expect(labels[0]).toBe('FROM');
    expect(labels).toContain('Transaction');
    expect(labels).toContain('Span');
  });
});
