'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Check, X, AlertCircle } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Separator } from '@/components/ui/separator';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { useAdminOrder, useApproveOrder, useRejectOrder } from '@/lib/api/hooks';
import { OrderStatusBadge } from '@/components/client/order-status-badge';

export default function AdminOrderDetailPage() {
  const params = useParams();
  const router = useRouter();
  const orderId = params.id as string;

  const { data: order, isLoading, error } = useAdminOrder(orderId);
  const approveOrder = useApproveOrder();
  const rejectOrder = useRejectOrder();

  const [rejectReason, setRejectReason] = useState('');
  const [isRejectDialogOpen, setIsRejectDialogOpen] = useState(false);

  const handleApprove = async () => {
    try {
      // In a real app, we'd get the admin's email from the session
      await approveOrder.mutateAsync({
        orderId,
        approvedBy: 'admin@example.com', // This should come from session
      });
      toast.success('Order approved successfully');
      router.push('/admin/orders');
    } catch {
      toast.error('Failed to approve order');
    }
  };

  const handleReject = async () => {
    if (!rejectReason.trim()) {
      toast.error('Please provide a rejection reason');
      return;
    }

    try {
      await rejectOrder.mutateAsync({
        orderId,
        reason: rejectReason,
      });
      toast.success('Order rejected');
      setIsRejectDialogOpen(false);
      router.push('/admin/orders');
    } catch {
      toast.error('Failed to reject order');
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-32" />
        <div className="space-y-4">
          <Skeleton className="h-32 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    );
  }

  if (error || !order) {
    return (
      <div className="space-y-6">
        <Link
          href="/admin/orders"
          className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Orders
        </Link>
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-destructive">
            Order not found or failed to load.
          </p>
        </div>
      </div>
    );
  }

  const canProcess = order.status === 'pending';

  return (
    <div className="space-y-6">
      <Link
        href="/admin/orders"
        className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Orders
      </Link>

      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">
            Order #{order.id.slice(0, 8)}
          </h1>
          <p className="text-muted-foreground">
            Submitted on{' '}
            {new Date(order.created_at).toLocaleDateString('en-US', {
              year: 'numeric',
              month: 'long',
              day: 'numeric',
              hour: '2-digit',
              minute: '2-digit',
            })}
          </p>
        </div>
        <div className="flex items-center gap-4">
          <OrderStatusBadge status={order.status} className="text-sm" />
          {canProcess && (
            <>
              <Button
                onClick={handleApprove}
                disabled={approveOrder.isPending}
                className="bg-green-600 hover:bg-green-700"
              >
                <Check className="mr-2 h-4 w-4" />
                {approveOrder.isPending ? 'Approving...' : 'Approve'}
              </Button>

              <Dialog open={isRejectDialogOpen} onOpenChange={setIsRejectDialogOpen}>
                <DialogTrigger render={<Button variant="destructive" />}>
                  <X className="mr-2 h-4 w-4" />
                  Reject
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Reject Order</DialogTitle>
                    <DialogDescription>
                      Please provide a reason for rejecting this order. This
                      will be visible to the customer.
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4">
                    <div className="space-y-2">
                      <Label htmlFor="reason">Rejection Reason</Label>
                      <Textarea
                        id="reason"
                        placeholder="Enter the reason for rejection..."
                        value={rejectReason}
                        onChange={(e) => setRejectReason(e.target.value)}
                        rows={4}
                      />
                    </div>
                  </div>
                  <DialogFooter>
                    <Button
                      variant="outline"
                      onClick={() => setIsRejectDialogOpen(false)}
                    >
                      Cancel
                    </Button>
                    <Button
                      variant="destructive"
                      onClick={handleReject}
                      disabled={rejectOrder.isPending || !rejectReason.trim()}
                    >
                      {rejectOrder.isPending ? 'Rejecting...' : 'Reject Order'}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </>
          )}
        </div>
      </div>

      {order.rejection_reason && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <div className="flex gap-3">
            <AlertCircle className="h-5 w-5 text-destructive" />
            <div>
              <h3 className="font-semibold text-destructive">Rejection Reason</h3>
              <p className="text-sm text-destructive/80">
                {order.rejection_reason}
              </p>
            </div>
          </div>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Order Items</CardTitle>
            <CardDescription>
              {order.item_count} item{order.item_count !== 1 ? 's' : ''}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="divide-y">
              {order.items.map((item, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between py-4 first:pt-0 last:pb-0"
                >
                  <div>
                    <p className="font-medium">
                      Product ID: {item.product_id.slice(0, 8)}...
                    </p>
                    <p className="text-sm text-muted-foreground">
                      Quantity: {item.quantity}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Order Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <p className="text-sm text-muted-foreground">Order ID</p>
                <p className="font-mono text-sm">{order.id}</p>
              </div>
              <Separator />
              <div>
                <p className="text-sm text-muted-foreground">Status</p>
                <OrderStatusBadge status={order.status} className="mt-1" />
              </div>
              {order.approved_at && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm text-muted-foreground">Approved</p>
                    <p className="text-sm">
                      {new Date(order.approved_at).toLocaleDateString('en-US', {
                        year: 'numeric',
                        month: 'short',
                        day: 'numeric',
                      })}
                    </p>
                  </div>
                </>
              )}
              {order.holded_invoice_id && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm text-muted-foreground">Holded Invoice</p>
                    <p className="font-mono text-sm">{order.holded_invoice_id}</p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {order.notes && (
            <Card>
              <CardHeader>
                <CardTitle>Customer Notes</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">{order.notes}</p>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
