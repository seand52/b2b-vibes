# Tech Strategy - Golden Paths (Customize for Your Project)

This is the **SINGLE SOURCE OF TRUTH** for technology choices.

## Customization Required

**IMPORTANT**: This file contains example technology choices. Customize it for your project.

Replace the Golden Paths below with your actual tech stack. The framework enforces whatever you put here.

## Compliance

1. **Follow This File**: Use the technologies listed in the Golden Paths below
2. **No Deviations**: Do not suggest alternatives unless explicitly instructed
3. **Latest Stable**: Always use the latest stable version unless pinned

## Language Golden Paths

### Go (Systems Standard)

| Component | Choice |
|-----------|--------|
| Runtime | Go 1.25+ (PGO) |
| Framework | Gin or Chi |
| Data | sqlc + pgx v5 |
| Linting | golangci-lint |
| Images | Wolfi base |

