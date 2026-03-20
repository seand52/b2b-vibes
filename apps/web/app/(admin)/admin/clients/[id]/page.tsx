'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Mail, Phone, Building2, ChevronRight } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Separator } from '@/components/ui/separator';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { useAdminClient, useAdminOrders } from '@/lib/api/hooks';
import { OrderStatusBadge } from '@/components/client/order-status-badge';

export default function AdminClientDetailPage() {
  const params = useParams();
  const clientId = params.id as string;

  const { data: client, isLoading, error } = useAdminClient(clientId);
  const { data: clientOrders } = useAdminOrders({ client_id: clientId });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-32" />
        <div className="grid gap-6 lg:grid-cols-3">
          <Skeleton className="h-64 w-full lg:col-span-1" />
          <Skeleton className="h-64 w-full lg:col-span-2" />
        </div>
      </div>
    );
  }

  if (error || !client) {
    return (
      <div className="space-y-6">
        <Link
          href="/admin/clients"
          className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Clients
        </Link>
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-destructive">
            Client not found or failed to load.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link
        href="/admin/clients"
        className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Clients
      </Link>

      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            {client.company_name}
          </h1>
          {client.contact_name && (
            <p className="text-muted-foreground">{client.contact_name}</p>
          )}
        </div>
        <div className="flex gap-2">
          <Badge variant={client.is_active ? 'default' : 'secondary'}>
            {client.is_active ? 'Active' : 'Inactive'}
          </Badge>
          <Badge variant={client.is_linked ? 'outline' : 'secondary'}>
            {client.is_linked ? 'Linked' : 'Not Linked'}
          </Badge>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Contact Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-3">
                <Mail className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm">{client.email}</span>
              </div>
              {client.phone && (
                <div className="flex items-center gap-3">
                  <Phone className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm">{client.phone}</span>
                </div>
              )}
              {client.vat_number && (
                <div className="flex items-center gap-3">
                  <Building2 className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm">{client.vat_number}</span>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Account Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <p className="text-sm text-muted-foreground">Client ID</p>
                <p className="font-mono text-sm">{client.id}</p>
              </div>
              <Separator />
              <div>
                <p className="text-sm text-muted-foreground">Holded ID</p>
                <p className="font-mono text-sm">{client.holded_id}</p>
              </div>
              <Separator />
              <div>
                <p className="text-sm text-muted-foreground">Created</p>
                <p className="text-sm">
                  {new Date(client.created_at).toLocaleDateString('en-US', {
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric',
                  })}
                </p>
              </div>
            </CardContent>
          </Card>

          {(client.billing_address || client.shipping_address) && (
            <Card>
              <CardHeader>
                <CardTitle>Addresses</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {client.billing_address && (
                  <div>
                    <p className="text-sm font-medium">Billing Address</p>
                    <p className="text-sm text-muted-foreground">
                      {client.billing_address.street && (
                        <>{client.billing_address.street}<br /></>
                      )}
                      {client.billing_address.city && client.billing_address.postal_code && (
                        <>{client.billing_address.postal_code} {client.billing_address.city}<br /></>
                      )}
                      {client.billing_address.country}
                    </p>
                  </div>
                )}
                {client.shipping_address && (
                  <>
                    {client.billing_address && <Separator />}
                    <div>
                      <p className="text-sm font-medium">Shipping Address</p>
                      <p className="text-sm text-muted-foreground">
                        {client.shipping_address.street && (
                          <>{client.shipping_address.street}<br /></>
                        )}
                        {client.shipping_address.city && client.shipping_address.postal_code && (
                          <>{client.shipping_address.postal_code} {client.shipping_address.city}<br /></>
                        )}
                        {client.shipping_address.country}
                      </p>
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          )}
        </div>

        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Order History</CardTitle>
            <CardDescription>
              Recent orders from this client
            </CardDescription>
          </CardHeader>
          <CardContent>
            {clientOrders && clientOrders.length > 0 ? (
              <div className="space-y-4">
                {clientOrders.slice(0, 10).map((order) => (
                  <Link
                    key={order.id}
                    href={`/admin/orders/${order.id}`}
                    className="flex items-center justify-between rounded-lg border p-4 transition-colors hover:bg-muted/50"
                  >
                    <div className="space-y-1">
                      <div className="flex items-center gap-3">
                        <span className="font-medium">
                          Order #{order.id.slice(0, 8)}
                        </span>
                        <OrderStatusBadge status={order.status} />
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {order.item_count} item{order.item_count !== 1 ? 's' : ''} •{' '}
                        {new Date(order.created_at).toLocaleDateString()}
                      </p>
                    </div>
                    <ChevronRight className="h-5 w-5 text-muted-foreground" />
                  </Link>
                ))}
              </div>
            ) : (
              <p className="text-center text-muted-foreground py-8">
                No orders from this client yet.
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
