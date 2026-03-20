'use client';

import { ClipboardList, Users, Clock, CheckCircle } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useAdminOrders, useAdminClients } from '@/lib/api/hooks';

export default function AdminDashboardPage() {
  const { data: allOrders, isLoading: ordersLoading } = useAdminOrders();
  const { data: pendingOrders, isLoading: pendingLoading } = useAdminOrders({
    status: 'pending',
  });
  const { data: clients, isLoading: clientsLoading } = useAdminClients();

  const stats = [
    {
      name: 'Total Orders',
      value: allOrders?.length ?? 0,
      icon: ClipboardList,
      isLoading: ordersLoading,
    },
    {
      name: 'Pending Orders',
      value: pendingOrders?.length ?? 0,
      icon: Clock,
      isLoading: pendingLoading,
      highlight: (pendingOrders?.length ?? 0) > 0,
    },
    {
      name: 'Approved Today',
      value:
        allOrders?.filter(
          (o) =>
            o.status === 'approved' &&
            o.approved_at &&
            new Date(o.approved_at).toDateString() === new Date().toDateString()
        ).length ?? 0,
      icon: CheckCircle,
      isLoading: ordersLoading,
    },
    {
      name: 'Total Clients',
      value: clients?.length ?? 0,
      icon: Users,
      isLoading: clientsLoading,
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          Overview of your B2B orders platform
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon;
          return (
            <Card key={stat.name} className={stat.highlight ? 'border-primary' : ''}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  {stat.name}
                </CardTitle>
                <Icon className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                {stat.isLoading ? (
                  <Skeleton className="h-8 w-16" />
                ) : (
                  <div className="text-2xl font-bold">{stat.value}</div>
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>

      {pendingOrders && pendingOrders.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Recent Pending Orders</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {pendingOrders.slice(0, 5).map((order) => (
                <div
                  key={order.id}
                  className="flex items-center justify-between rounded-lg border p-4"
                >
                  <div>
                    <p className="font-medium">Order #{order.id.slice(0, 8)}</p>
                    <p className="text-sm text-muted-foreground">
                      {order.item_count} items •{' '}
                      {new Date(order.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  <a
                    href={`/admin/orders/${order.id}`}
                    className="text-sm font-medium text-primary hover:underline"
                  >
                    Review
                  </a>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
