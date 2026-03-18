import { NextResponse, type NextRequest } from 'next/server';
import { auth0, extractRoles, hasAdminRole } from '@/lib/auth/config';

// Routes that require authentication
const protectedRoutes = ['/products', '/cart', '/orders', '/admin'];
// Routes that require admin role
const adminRoutes = ['/admin'];

export async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Let auth0 handle /auth/* routes
  const authResponse = await auth0.middleware(request);

  // For auth routes, return the auth0 response directly
  if (pathname.startsWith('/auth/')) {
    return authResponse;
  }

  // For API routes, let them handle their own auth
  if (pathname.startsWith('/api/')) {
    return authResponse;
  }

  // Check if this is a protected route
  const isProtectedRoute = protectedRoutes.some((route) =>
    pathname.startsWith(route)
  );
  const isAdminRoute = adminRoutes.some((route) => pathname.startsWith(route));

  // For protected routes, check authentication
  if (isProtectedRoute) {
    try {
      const session = await auth0.getSession();

      if (!session) {
        // Redirect to login
        const loginUrl = new URL('/auth/login', request.url);
        loginUrl.searchParams.set('returnTo', pathname);
        return NextResponse.redirect(loginUrl);
      }

      // For admin routes, check admin role
      if (isAdminRoute) {
        // In Auth0 SDK v4, user claims are available directly on session.user
        const claims = session.user as Record<string, unknown>;
        const roles = extractRoles(claims);

        if (!hasAdminRole(roles)) {
          // Non-admin trying to access admin route, redirect to products
          return NextResponse.redirect(new URL('/products', request.url));
        }
      }
    } catch {
      // Session error, redirect to login
      const loginUrl = new URL('/auth/login', request.url);
      loginUrl.searchParams.set('returnTo', pathname);
      return NextResponse.redirect(loginUrl);
    }
  }

  return authResponse;
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public folder
     */
    '/((?!_next/static|_next/image|favicon.ico|.*\\..*|public).*)',
  ],
};
