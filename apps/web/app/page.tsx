import { redirect } from 'next/navigation';
import Link from 'next/link';
import { auth0, extractRolesFromAccessToken, hasAdminRole } from '@/lib/auth/config';
import { Button } from '@/components/ui/button';

export default async function HomePage() {
  const session = await auth0.getSession();

  // If already authenticated, redirect based on role
  if (session) {
    const accessToken = await auth0.getAccessToken();
    const roles = extractRolesFromAccessToken(accessToken?.token || '');

    if (hasAdminRole(roles)) {
      redirect('/admin');
    }
    redirect('/products');
  }

  // Show landing page for unauthenticated users
  return (
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <header className="border-b">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-xl font-bold">B2B Orders</h1>
          <Link href="/auth/login">
            <Button>Sign In</Button>
          </Link>
        </div>
      </header>

      {/* Hero */}
      <main className="flex-1 flex items-center justify-center">
        <div className="container mx-auto px-4 text-center">
          <h2 className="text-4xl font-bold mb-4">
            Welcome to B2B Orders
          </h2>
          <p className="text-lg text-muted-foreground mb-8 max-w-md mx-auto">
            The ordering platform for business clients. Browse products, manage your cart, and track your orders.
          </p>
          <Link href="/auth/login">
            <Button size="lg">
              Get Started
            </Button>
          </Link>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t py-6">
        <div className="container mx-auto px-4 text-center text-sm text-muted-foreground">
          © {new Date().getFullYear()} B2B Orders. All rights reserved.
        </div>
      </footer>
    </div>
  );
}
