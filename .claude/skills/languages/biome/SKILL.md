---
name: biome
description: Biome 2.x linting and formatting patterns. Use when configuring code quality tools, setting up linting rules, formatting code, or integrating with CI/CD. Covers migration from ESLint/Prettier.
---

# Biome 2.x

## Overview

Fast, all-in-one toolchain for linting and formatting JavaScript, TypeScript, JSX, and JSON. Biome 2.x replaces ESLint and Prettier with a single, performant tool written in Rust.

**Install**: `pnpm add -D @biomejs/biome`

> **API Reference:** Use Context7 MCP for full rule reference and CLI flags (`mcp__context7__resolve-library-id` → `@biomejs/biome`).

## Workflows

**Initial setup:**
1. [ ] Install Biome: `pnpm add -D @biomejs/biome`
2. [ ] Initialize config: `pnpm biome init`
3. [ ] Configure biome.json with project standards
4. [ ] Install VS Code extension: `biomejs.biome`
5. [ ] Add npm scripts to package.json
6. [ ] Test: `pnpm biome check .`

**Migrating from ESLint/Prettier:**
1. [ ] Run migration helper: `pnpm biome migrate eslint --write`
2. [ ] Review generated biome.json
3. [ ] Remove ESLint/Prettier configs and dependencies
4. [ ] Update pre-commit hooks and CI scripts
5. [ ] Run full check: `pnpm biome check --write .`

**Daily usage:**
1. [ ] Format on save (VS Code integration)
2. [ ] Run `pnpm biome check .` before commits
3. [ ] Fix auto-fixable issues: `pnpm biome check --write .`
4. [ ] Review manual fixes for remaining issues

## Configuration

### biome.json Structure

```json
{
  "$schema": "https://biomejs.dev/schemas/2.0.0/schema.json",
  "organizeImports": { "enabled": true },
  "linter": {
    "enabled": true,
    "rules": { "recommended": true }
  },
  "formatter": {
    "enabled": true,
    "indentStyle": "space",
    "indentWidth": 2,
    "lineWidth": 100
  },
  "javascript": {
    "formatter": {
      "quoteStyle": "single",
      "semicolons": "always",
      "trailingCommas": "es5",
      "arrowParentheses": "asNeeded"
    }
  },
  "files": {
    "ignore": ["dist", "build", "node_modules", "*.min.js", "coverage"]
  }
}
```

### TypeScript / React Project Config

```json
{
  "linter": {
    "rules": {
      "recommended": true,
      "a11y": { "noBlankTarget": "error", "useAltText": "error", "useButtonType": "error" },
      "complexity": { "noExcessiveCognitiveComplexity": "warn", "noUselessFragments": "error" },
      "correctness": { "noUnusedVariables": "error", "useExhaustiveDependencies": "warn", "useHookAtTopLevel": "error" },
      "performance": { "noAccumulatingSpread": "warn" },
      "security": { "noDangerouslySetInnerHtml": "warn" },
      "style": { "noNonNullAssertion": "warn", "useConst": "error", "useImportType": "error" },
      "suspicious": { "noExplicitAny": "error", "noDebugger": "error", "noConsoleLog": "warn" }
    }
  },
  "javascript": {
    "formatter": { "jsxQuoteStyle": "double", "quoteStyle": "single" }
  }
}
```

### Per-File Overrides

```json
{
  "overrides": [
    {
      "include": ["tests/**/*.ts"],
      "linter": { "rules": { "suspicious": { "noExplicitAny": "off" } } }
    },
    {
      "include": ["scripts/**/*.js"],
      "formatter": { "lineWidth": 120 }
    }
  ]
}
```

## CLI Commands

```bash
# Check (lint + format) — use this for most tasks
pnpm biome check .
pnpm biome check --write .          # auto-fix
pnpm biome check --write --dry-run . # preview changes

# Lint or format only
pnpm biome lint --write .
pnpm biome format --write .

# Diagnostics
pnpm biome rage                     # effective config + diagnostics
pnpm biome explain src/App.tsx      # explain failures for a file
pnpm biome migrate eslint --write   # migrate from ESLint
pnpm biome migrate prettier --write # migrate from Prettier
```

## Package.json Scripts

```json
{
  "scripts": {
    "lint": "biome lint .",
    "format": "biome format --write .",
    "check": "biome check .",
    "fix": "biome check --write .",
    "typecheck": "tsc --noEmit",
    "quality": "pnpm lint && pnpm typecheck && pnpm build"
  }
}
```

## Editor Integration

### VS Code settings.json

```json
{
  "editor.defaultFormatter": "biomejs.biome",
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "quickfix.biome": "explicit",
    "source.organizeImports.biome": "explicit"
  },
  "eslint.enable": false,
  "prettier.enable": false,
  "[javascript]": { "editor.defaultFormatter": "biomejs.biome" },
  "[typescript]": { "editor.defaultFormatter": "biomejs.biome" },
  "[typescriptreact]": { "editor.defaultFormatter": "biomejs.biome" },
  "[json]": { "editor.defaultFormatter": "biomejs.biome" },
  "biome.lspBin": "./node_modules/@biomejs/biome/bin/biome"
}
```

## Ignoring Code

```typescript
// biome-ignore lint/suspicious/noExplicitAny: legacy code
function legacy(param: any) { return param; }

// biome-ignore format: preserve matrix formatting
const matrix = [
  1, 0, 0,
  0, 1, 0,
  0, 0, 1,
];
```

Prefer `files.ignore` in biome.json for ignoring entire directories over inline comments.

## Git Hooks Integration

**Using lint-staged + Husky:**

```json
// package.json
{
  "lint-staged": {
    "*.{js,jsx,ts,tsx,json}": ["biome check --write --no-errors-on-unmatched"]
  }
}
```

**Using Lefthook (`lefthook.yml`):**
```yaml
pre-commit:
  parallel: true
  commands:
    biome:
      glob: "*.{js,ts,jsx,tsx,json}"
      run: biome check --write --no-errors-on-unmatched {staged_files}
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Code Quality
on: [push, pull_request]
jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v4
        with: { version: 10 }
      - uses: actions/setup-node@v4
        with: { node-version: '24', cache: 'pnpm' }
      - run: pnpm install --frozen-lockfile
      - run: pnpm biome check .
      - run: pnpm typecheck
```

## ESLint Rule Equivalents

| ESLint Rule | Biome Rule |
|-------------|------------|
| `no-unused-vars` | `correctness/noUnusedVariables` |
| `@typescript-eslint/no-explicit-any` | `suspicious/noExplicitAny` |
| `react-hooks/exhaustive-deps` | `correctness/useExhaustiveDependencies` |
| `no-console` | `suspicious/noConsoleLog` |
| `prefer-const` | `style/useConst` |
| `jsx-a11y/alt-text` | `a11y/useAltText` |

## Best Practices

- **Use recommended ruleset** as baseline, then customize specific rules
- **Enable format-on-save** in VS Code for seamless workflow
- **Run check before commits** using git hooks (Husky/Lefthook)
- **Use biome check** (not lint + format separately) for unified workflow
- **Ignore generated files** in biome.json, not inline comments
- **Use overrides** for different rules in tests vs source
- **Commit biome.json** to version control for team consistency
- **Document custom rules** with comments explaining why they're needed

## Anti-Patterns

- Running lint and format separately (use `check` instead)
- Disabling recommended rules without justification
- Using biome-ignore excessively (fix the underlying issue)
- Not committing biome.json to version control
- Mixing ESLint and Biome in the same project
- Ignoring files via inline comments instead of configuration
- Not testing migration thoroughly before removing ESLint/Prettier
- Skipping pre-commit hooks for "quick fixes"
- Using outdated schema version in biome.json

## Feedback Loops

```bash
# Preview formatting changes without writing
pnpm biome format --write --dry-run .

# Diagnose config issues
pnpm biome rage

# CI test locally (exit code 0 = success)
pnpm biome check . --error-on-warnings && echo "OK"
```

Verify VS Code integration: Command Palette → "Biome: Show Output Channel" → should show LSP server logs.
