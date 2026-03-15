---
name: hono
description: Hono 4.x web framework patterns. Use when building APIs, middleware, routing, or server-side applications. Covers multi-runtime support (Node, Bun, Cloudflare Workers), validation, CORS, and error handling.
---

# Hono

> Use Context7 MCP (`resolve-library-id` then `query-docs`) for full API reference, all built-in middleware, and additional runtime setup examples.

## Overview

Lightweight, fast web framework for APIs and server-side applications. Hono 4.x works across Node.js, Bun, Deno, Cloudflare Workers with a consistent API.

**Install**: `pnpm add hono`

## Workflows

**Creating a basic API:**
1. [ ] Create Hono app instance: `const app = new Hono()`
2. [ ] Define routes with HTTP methods
3. [ ] Add middleware (CORS, logger, error handling)
4. [ ] Export app for runtime adapter
5. [ ] Test endpoints with curl or Postman

**Adding validation:**
1. [ ] Install Zod: `pnpm add zod @hono/zod-validator`
2. [ ] Define schemas with Zod
3. [ ] Apply `zValidator` middleware to routes
4. [ ] Handle validation errors
5. [ ] Access type-safe request data via `c.req.valid()`

## Runtime Setup

```typescript
// Node.js
import { Hono } from 'hono';
import { serve } from '@hono/node-server';
const app = new Hono();
serve({ fetch: app.fetch, port: 3000 });

// Bun
export default { port: 3000, fetch: app.fetch };

// Cloudflare Workers
export default app;
```

## Routing

```typescript
// HTTP methods
app.get('/users', (c) => c.json({ users: [] }));
app.post('/users', (c) => c.json({ created: true }, 201));
app.put('/users/:id', (c) => c.json({ updated: true }));
app.delete('/users/:id', (c) => c.json({ deleted: true }));
app.on(['GET', 'POST'], '/multi', (c) => c.text(c.req.method));

// Path parameters
app.get('/users/:id', (c) => {
  const id = c.req.param('id');
  return c.json({ userId: id });
});

// Multiple params
app.get('/posts/:postId/comments/:commentId', (c) => {
  const { postId, commentId } = c.req.param();
  return c.json({ postId, commentId });
});

// Wildcard
app.get('/files/*', (c) => c.text('File handler'));

// Regex constraint (only numeric IDs)
app.get('/posts/:id{[0-9]+}', (c) => c.json({ id: c.req.param('id') }));
```

### Route Groups

```typescript
const v1 = new Hono();
v1.get('/users', (c) => c.json({ version: 1, users: [] }));

const v2 = new Hono();
v2.get('/users', (c) => c.json({ version: 2, users: [] }));

app.route('/api/v1', v1);
app.route('/api/v2', v2);
```

## Request Handling

```typescript
// Query params
app.get('/search', (c) => {
  const q = c.req.query('q');
  const page = c.req.query('page') ?? '1';
  return c.json({ q, page });
});

// JSON body
app.post('/users', async (c) => {
  const body = await c.req.json();
  return c.json({ received: body });
});

// Headers
app.get('/me', (c) => {
  const auth = c.req.header('Authorization');
  c.header('X-Custom-Header', 'value');
  return c.json({ auth });
});
```

## Response Types

```typescript
c.json({ data: 'value' })          // JSON (default 200)
c.json({ created: true }, 201)     // JSON with status
c.text('OK')                       // Plain text
c.html('<h1>Hello</h1>')           // HTML
c.redirect('/new')                 // 302 redirect
c.redirect('https://example.com', 301)
c.notFound()                       // 404
```

## Middleware

```typescript
import { logger } from 'hono/logger';
import { cors } from 'hono/cors';
import { secureHeaders } from 'hono/secure-headers';
import { prettyJSON } from 'hono/pretty-json';

app.use('*', logger());
app.use('*', secureHeaders());
app.use('*', cors({
  origin: ['http://localhost:3000', 'https://example.com'],
  allowMethods: ['GET', 'POST', 'PUT', 'DELETE'],
  allowHeaders: ['Content-Type', 'Authorization'],
  credentials: true,
}));
if (process.env.NODE_ENV === 'development') {
  app.use('*', prettyJSON());
}
```

### Custom Middleware

```typescript
// Auth middleware — store data in context
const authMiddleware = async (c, next) => {
  const token = c.req.header('Authorization');
  if (!token) return c.json({ error: 'Unauthorized' }, 401);
  c.set('user', { id: 1, name: 'Alice' }); // stored in context
  await next();
};

app.use('/api/*', authMiddleware);

app.get('/api/profile', (c) => {
  const user = c.get('user');
  return c.json({ user });
});
```

## Validation with Zod

```typescript
import { z } from 'zod';
import { zValidator } from '@hono/zod-validator';

const userSchema = z.object({
  name: z.string().min(1).max(100),
  email: z.string().email(),
  age: z.number().int().min(0).max(120).optional(),
});

// Validate body
app.post('/users', zValidator('json', userSchema), async (c) => {
  const user = c.req.valid('json'); // fully typed
  return c.json({ created: true, user }, 201);
});

// Validate path params
app.get('/users/:id', zValidator('param', z.object({ id: z.string().regex(/^\d+$/) })), (c) => {
  const { id } = c.req.valid('param');
  return c.json({ userId: id });
});

// Custom validation error response
app.post('/users', zValidator('json', userSchema, (result, c) => {
  if (!result.success) {
    return c.json({ error: 'Validation failed', details: result.error.flatten() }, 400);
  }
}), handler);
```

## Error Handling

```typescript
import { HTTPException } from 'hono/http-exception';

// Global error handler
app.onError((err, c) => {
  if (err instanceof HTTPException) {
    return c.json({ error: err.message, status: err.status }, err.status);
  }
  return c.json({
    error: 'Internal Server Error',
    message: process.env.NODE_ENV === 'development' ? err.message : undefined,
  }, 500);
});

// 404 handler
app.notFound((c) => c.json({ error: 'Not Found', path: c.req.path }, 404));

// Throw HTTP exceptions from route handlers
app.get('/protected', (c) => {
  throw new HTTPException(403, { message: 'Forbidden' });
});
```

## Type Safety

### Typed Context Variables

```typescript
type Env = {
  Variables: { user: { id: number; name: string } };
};

const app = new Hono<Env>();

app.use('/api/*', async (c, next) => {
  c.set('user', { id: 1, name: 'Alice' }); // type-checked
  await next();
});

app.get('/api/profile', (c) => {
  const user = c.get('user'); // fully typed
  return c.json({ user });
});
```

### RPC Type Safety (Hono Client)

```typescript
// server.ts — export the app type
const app = new Hono()
  .get('/posts', (c) => c.json({ posts: [] }))
  .post('/posts', async (c) => c.json({ created: true }, 201));

export type AppType = typeof app;

// client.ts — fully typed calls, no separate OpenAPI spec needed
import { hc } from 'hono/client';
import type { AppType } from './server';

const client = hc<AppType>('http://localhost:3000');
const res = await client.posts.$get();
const data = await res.json(); // { posts: [] }
```

## Testing

```typescript
import { describe, it, expect } from 'vitest';

describe('API', () => {
  const app = new Hono();
  app.get('/hello', (c) => c.json({ message: 'Hello' }));

  it('returns hello', async () => {
    const res = await app.request('/hello');
    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ message: 'Hello' });
  });

  it('handles POST', async () => {
    app.post('/users', async (c) => {
      const body = await c.req.json();
      return c.json({ created: true, user: body }, 201);
    });

    const res = await app.request('/users', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'Alice' }),
    });

    expect(res.status).toBe(201);
  });
});
```

## Best Practices

- **Use route groups** to organize related endpoints into modular routers
- **Validate all inputs** with Zod for type safety and runtime validation
- **Apply middleware sparingly** - only use what you need per route group
- **Set explicit CORS policies** for production — never use permissive CORS in prod
- **Use typed contexts** (`Hono<Env>`) for variables set in middleware
- **Handle errors globally** with `app.onError()` for consistent error responses
- **Use `HTTPException`** instead of manually constructing error responses
- **Test with `app.request()`** — Hono's built-in test utility (no server needed)
- **Leverage RPC types** for type-safe client-server communication

## Anti-Patterns

- ❌ Applying logger middleware after routes (won't log those routes)
- ❌ Forgetting to `await next()` in middleware (breaks middleware chain)
- ❌ Using `cors()` only on specific routes (preflight requests need global CORS)
- ❌ Parsing request body multiple times (cache after first parse)
- ❌ Not validating path parameters (always validate user input)
- ❌ Using `any` type instead of proper Hono generics
- ❌ Hardcoding origins in CORS config (use environment variables)
- ❌ Missing error handlers (leads to unhandled promise rejections)
- ❌ Forgetting to export app for runtime adapters

## Feedback Loops

**Testing endpoints:**
```bash
curl -X GET http://localhost:3000/api/users
curl -X POST http://localhost:3000/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'
```

**Validation testing:**
```bash
# Should return 400 with validation details
curl -X POST http://localhost:3000/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"","email":"invalid"}'
```

**Performance testing:**
```bash
pnpm add -D autocannon
npx autocannon -c 100 -d 10 http://localhost:3000/api/users
# Target: <10ms p99 latency for simple endpoints
```
