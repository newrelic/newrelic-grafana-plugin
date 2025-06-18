# Contributing to New Relic Grafana Plugin

Thank you for your interest in contributing to the New Relic Grafana Plugin! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Guidelines](#contributing-guidelines)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting](#issue-reporting)
- [Security](#security)

## Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [conduct@yourorg.com](mailto:conduct@yourorg.com).

## Getting Started

### Prerequisites

Before contributing, ensure you have:

- Node.js 22+ installed
- Go 1.21+ installed (for backend development)
- Git configured with your name and email
- A GitHub account
- Basic knowledge of TypeScript, React, and Go

### Development Setup

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/newrelic-grafana-plugin.git
   cd newrelic-grafana-plugin
   ```

3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/your-org/newrelic-grafana-plugin.git
   ```

4. **Install dependencies**:
   ```bash
   npm install
   ```

5. **Build the project**:
   ```bash
   npm run build
   ```

6. **Start development server**:
   ```bash
   npm run dev
   ```

7. **Run tests** to ensure everything works:
   ```bash
   npm test
   ```

## Contributing Guidelines

### Types of Contributions

We welcome various types of contributions:

- 🐛 **Bug fixes**
- ✨ **New features**
- 📚 **Documentation improvements**
- 🧪 **Test coverage improvements**
- 🎨 **UI/UX enhancements**
- ⚡ **Performance optimizations**
- 🔒 **Security improvements**

### Before You Start

1. **Check existing issues** to avoid duplicate work
2. **Create an issue** for significant changes to discuss the approach
3. **Keep changes focused** - one feature/fix per PR
4. **Follow the coding standards** outlined below

## Code Standards

### TypeScript/React Frontend

#### Code Style

- Use **TypeScript** for all new code
- Follow **React functional components** with hooks
- Use **arrow functions** for component definitions
- Implement **proper error boundaries**
- Include **comprehensive JSDoc comments**

```typescript
/**
 * Validates a New Relic API key format
 * @param apiKey - The API key to validate
 * @returns Validation result with success status and optional error message
 */
export function validateApiKey(apiKey: string): ValidationResult {
  // Implementation
}
```

#### Component Structure

```typescript
import React, { useState, useCallback } from 'react';
import { ComponentProps } from './types';

/**
 * Component description
 */
export function MyComponent({ prop1, prop2 }: ComponentProps) {
  const [state, setState] = useState<StateType>(initialState);

  const handleAction = useCallback(() => {
    // Implementation
  }, [dependencies]);

  return (
    <div>
      {/* JSX */}
    </div>
  );
}
```

#### Accessibility Requirements

- Include **ARIA labels** for all interactive elements
- Provide **keyboard navigation** support
- Ensure **color contrast** meets WCAG 2.1 AA standards
- Add **screen reader** support
- Test with **accessibility tools**

```typescript
<Button
  aria-label="Run NRQL query"
  aria-describedby="query-help"
  onClick={handleRunQuery}
>
  Run Query
</Button>
```

### Go Backend

#### Code Style

- Follow **Go conventions** and **gofmt** formatting
- Use **meaningful variable names**
- Include **comprehensive comments**
- Implement **proper error handling**
- Follow **Go project layout** standards

```go
// ValidateNRQLQuery validates an NRQL query string
func ValidateNRQLQuery(query string) error {
    if strings.TrimSpace(query) == "" {
        return errors.New("query cannot be empty")
    }
    // Additional validation
    return nil
}
```

### General Standards

#### File Organization

```
src/
├── components/          # React components
│   ├── __tests__/      # Component tests
│   └── ComponentName.tsx
├── utils/              # Utility functions
│   ├── __tests__/      # Utility tests
│   └── utilityName.ts
├── types.ts            # Type definitions
└── datasource.ts       # Main datasource
```

#### Naming Conventions

- **Files**: PascalCase for components, camelCase for utilities
- **Variables**: camelCase
- **Constants**: UPPER_SNAKE_CASE
- **Types/Interfaces**: PascalCase
- **Functions**: camelCase with descriptive names

#### Import Organization

```typescript
// External libraries
import React from 'react';
import { Button } from '@grafana/ui';

// Internal utilities
import { validateApiKey } from '../utils/validation';
import { logger } from '../utils/logger';

// Types
import { NewRelicQuery } from '../types';

// Relative imports
import { ComponentName } from './ComponentName';
```

## Testing

### Test Requirements

- **95%+ code coverage** for all new code
- **Unit tests** for all functions and components
- **Integration tests** for complex workflows
- **E2E tests** for critical user paths

### Testing Standards

#### Unit Tests

```typescript
describe('validateApiKey', () => {
  it('should validate correct API key format', () => {
    const result = validateApiKey('NRAK1234567890abcdef1234567890abcdef1234');
    expect(result.isValid).toBe(true);
  });

  it('should reject invalid API key format', () => {
    const result = validateApiKey('invalid-key');
    expect(result.isValid).toBe(false);
    expect(result.message).toBe('API key must be 40 characters long and contain only alphanumeric characters');
  });
});
```

#### Component Tests

```typescript
describe('ConfigEditor', () => {
  it('should render all configuration fields', () => {
    render(<ConfigEditor {...defaultProps} />);
    
    expect(screen.getByLabelText('New Relic API Key')).toBeInTheDocument();
    expect(screen.getByLabelText('New Relic Account ID')).toBeInTheDocument();
  });

  it('should validate input on change', async () => {
    const user = userEvent.setup();
    render(<ConfigEditor {...defaultProps} />);
    
    const input = screen.getByTestId('api-key-input');
    await user.type(input, 'invalid-key');
    
    expect(screen.getByText('Invalid API key format')).toBeInTheDocument();
  });
});
```

### Running Tests

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage

# Run E2E tests
npm run e2e
```

## Pull Request Process

### Before Submitting

1. **Sync with upstream**:
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes** following the code standards

4. **Add tests** for your changes

5. **Run the test suite**:
   ```bash
   npm run test:ci
   npm run lint
   npm run typecheck
   ```

6. **Update documentation** if needed

### Commit Guidelines

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test additions/changes
- `chore`: Build process or auxiliary tool changes

**Examples:**
```
feat(query-builder): add support for SELECT * queries

Add new aggregation option for SELECT * queries in the visual query builder.
Includes proper parsing and validation for raw data queries.

Closes #123
```

### Pull Request Template

When creating a PR, include:

- **Description** of changes
- **Type of change** (bug fix, feature, etc.)
- **Testing** performed
- **Screenshots** for UI changes
- **Breaking changes** if any
- **Related issues** (closes #123)

### Review Process

1. **Automated checks** must pass (CI, tests, linting)
2. **Code review** by at least one maintainer
3. **Manual testing** for significant changes
4. **Documentation review** if applicable
5. **Security review** for security-related changes

## Issue Reporting

### Bug Reports

Include:

- **Clear description** of the issue
- **Steps to reproduce** the problem
- **Expected vs actual behavior**
- **Environment details** (Grafana version, browser, OS)
- **Screenshots/logs** if applicable
- **Minimal reproduction** case

### Feature Requests

Include:

- **Clear description** of the feature
- **Use case** and motivation
- **Proposed solution** (if any)
- **Alternative solutions** considered
- **Additional context** or examples

### Issue Labels

- `bug`: Something isn't working
- `enhancement`: New feature or request
- `documentation`: Improvements or additions to docs
- `good first issue`: Good for newcomers
- `help wanted`: Extra attention is needed
- `priority/high`: High priority issue
- `security`: Security-related issue

## Security

### Reporting Security Issues

**DO NOT** create public issues for security vulnerabilities. Instead:

1. Email [security@yourorg.com](mailto:security@yourorg.com)
2. Include detailed description of the vulnerability
3. Provide steps to reproduce if possible
4. Allow time for investigation and fix

### Security Guidelines

- **Never commit** API keys or sensitive data
- **Validate all inputs** on both frontend and backend
- **Use secure logging** to prevent data exposure
- **Follow OWASP** security guidelines
- **Keep dependencies** up to date

## Development Workflow

### Branch Strategy

- `main`: Production-ready code
- `develop`: Integration branch for features
- `feature/*`: Feature development branches
- `hotfix/*`: Critical bug fixes
- `release/*`: Release preparation branches

### Release Process

1. **Create release branch** from `develop`
2. **Update version** numbers and changelog
3. **Run full test suite** and manual testing
4. **Create pull request** to `main`
5. **Tag release** after merge
6. **Deploy** to production

## Getting Help

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and ideas
- **Email**: [dev@yourorg.com](mailto:dev@yourorg.com) for development questions
- **Slack**: [Join our workspace](https://yourorg.slack.com) for real-time chat

### Documentation

- **API Documentation**: Generated from code comments
- **User Guide**: In the main README
- **Developer Guide**: This document
- **Architecture Guide**: In the `/docs` folder

## Recognition

Contributors will be recognized in:

- **CONTRIBUTORS.md** file
- **Release notes** for significant contributions
- **Annual contributor** appreciation posts
- **Grafana community** showcases

Thank you for contributing to the New Relic Grafana Plugin! 🎉 