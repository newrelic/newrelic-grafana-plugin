// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
process.env.TZ = 'UTC';

module.exports = {
  moduleNameMapper: {
    // Mock monaco-editor to avoid issues with Jest and Monaco Editor
    'monaco-editor': '<rootDir>/node_modules/monaco-editor/esm/vs/editor/editor.api.d.ts',
  },
  // Jest configuration provided by Grafana scaffolding
  ...require('./.config/jest.config'),
};
