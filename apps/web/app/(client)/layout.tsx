import { ClientHeader } from '@/components/client/header';

export default function ClientLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-background">
      <ClientHeader />
      <main className="container py-6">{children}</main>
    </div>
  );
}
