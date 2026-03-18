import { redirect } from 'next/navigation';
import { auth0 } from '@/lib/auth/config';

export default async function HomePage() {
  const session = await auth0.getSession();

  if (session) {
    // Authenticated user -> go to products
    redirect('/products');
  } else {
    // Not authenticated -> go to login
    redirect('/auth/login');
  }
}
