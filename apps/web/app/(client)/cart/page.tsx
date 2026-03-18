'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Trash2, ShoppingBag } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  useCart,
  useUpdateCartNotes,
  useSubmitCart,
  useDiscardCart,
} from '@/lib/api/hooks';
import { ApiClientError } from '@/lib/api/client';
import { CartItem } from '@/components/client/cart-item';

export default function CartPage() {
  const router = useRouter();
  const { data: cart, isLoading, error } = useCart();
  const updateNotes = useUpdateCartNotes();
  const submitCart = useSubmitCart();
  const discardCart = useDiscardCart();

  const [notes, setNotes] = useState('');
  const [isNotesChanged, setIsNotesChanged] = useState(false);

  // Update local notes when cart loads
  if (cart && !isNotesChanged && cart.notes !== notes) {
    setNotes(cart.notes || '');
  }

  const handleNotesChange = (value: string) => {
    setNotes(value);
    setIsNotesChanged(true);
  };

  const handleSaveNotes = async () => {
    try {
      await updateNotes.mutateAsync(notes);
      setIsNotesChanged(false);
      toast.success('Notes saved');
    } catch {
      toast.error('Failed to save notes');
    }
  };

  const handleSubmitOrder = async () => {
    // Save notes first if changed
    if (isNotesChanged) {
      try {
        await updateNotes.mutateAsync(notes);
      } catch {
        toast.error('Failed to save notes');
        return;
      }
    }

    try {
      const order = await submitCart.mutateAsync();
      toast.success('Order submitted successfully!');
      router.push(`/orders/${order.id}`);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to submit order';
      toast.error(message);
    }
  };

  const handleDiscardCart = async () => {
    try {
      await discardCart.mutateAsync();
      toast.success('Cart discarded');
    } catch {
      toast.error('Failed to discard cart');
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-32" />
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2 space-y-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-24 w-full" />
            ))}
          </div>
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    );
  }

  // Handle 404 (no cart exists)
  if ((error instanceof ApiClientError && error.status === 404) || !cart) {
    return (
      <div className="space-y-6">
        <Link
          href="/products"
          className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Browse Products
        </Link>

        <div className="flex flex-col items-center justify-center py-16 text-center">
          <ShoppingBag className="h-16 w-16 text-muted-foreground" />
          <h2 className="mt-4 text-xl font-semibold">Your cart is empty</h2>
          <p className="mt-2 text-muted-foreground">
            Start browsing products to add items to your cart.
          </p>
          <Link href="/products">
            <Button className="mt-6">Browse Products</Button>
          </Link>
        </div>
      </div>
    );
  }

  const hasItems = cart.items.length > 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Link
            href="/products"
            className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            Continue Shopping
          </Link>
          <h1 className="mt-2 text-3xl font-bold tracking-tight">Your Cart</h1>
        </div>
        {hasItems && (
          <Button
            variant="ghost"
            size="sm"
            className="text-destructive hover:text-destructive"
            onClick={handleDiscardCart}
            disabled={discardCart.isPending}
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Clear Cart
          </Button>
        )}
      </div>

      {!hasItems ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <ShoppingBag className="h-16 w-16 text-muted-foreground" />
          <h2 className="mt-4 text-xl font-semibold">Your cart is empty</h2>
          <p className="mt-2 text-muted-foreground">
            Start browsing products to add items to your cart.
          </p>
          <Link href="/products">
            <Button className="mt-6">Browse Products</Button>
          </Link>
        </div>
      ) : (
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2 space-y-4">
            {cart.items.map((item) => (
              <CartItem key={item.product_id} item={item} />
            ))}

            <Card>
              <CardHeader>
                <CardTitle className="text-base">Order Notes</CardTitle>
              </CardHeader>
              <CardContent>
                <Textarea
                  placeholder="Add any special instructions or notes for your order..."
                  value={notes}
                  onChange={(e) => handleNotesChange(e.target.value)}
                  rows={3}
                />
              </CardContent>
              {isNotesChanged && (
                <CardFooter>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleSaveNotes}
                    disabled={updateNotes.isPending}
                  >
                    {updateNotes.isPending ? 'Saving...' : 'Save Notes'}
                  </Button>
                </CardFooter>
              )}
            </Card>
          </div>

          <div>
            <Card className="sticky top-24">
              <CardHeader>
                <CardTitle>Order Summary</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Items</span>
                  <span>{cart.summary.item_count}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Total Units</span>
                  <span>{cart.summary.total_units}</span>
                </div>
                <Separator />
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Subtotal</span>
                  <span>€{cart.summary.subtotal.toFixed(2)}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">
                    VAT ({cart.summary.tax_rate}%)
                  </span>
                  <span>€{cart.summary.tax_amount.toFixed(2)}</span>
                </div>
                <Separator />
                <div className="flex justify-between font-semibold">
                  <span>Total</span>
                  <span>€{cart.summary.total.toFixed(2)}</span>
                </div>
              </CardContent>
              <CardFooter>
                <Button
                  className="w-full"
                  size="lg"
                  onClick={handleSubmitOrder}
                  disabled={submitCart.isPending}
                >
                  {submitCart.isPending ? 'Submitting...' : 'Submit Order'}
                </Button>
              </CardFooter>
            </Card>
          </div>
        </div>
      )}
    </div>
  );
}
