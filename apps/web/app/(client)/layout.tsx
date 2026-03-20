import { ClientHeader } from '@/components/client/header';

export default function ClientLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-background">
      <ClientHeader />
      <main className="container mx-auto px-4 py-6 lg:px-8">{children}</main>
    </div>
  );
}
