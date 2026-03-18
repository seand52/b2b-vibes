'use client';

import { Plus, Minus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useUpdateCartQuantity, useRemoveFromCart } from '@/lib/api/hooks';
import type { CartItemResponse } from '@/lib/types';

interface CartItemProps {
  item: CartItemResponse;
}

export function CartItem({ item }: CartItemProps) {
  const updateQuantity = useUpdateCartQuantity();
  const removeItem = useRemoveFromCart();

  const handleIncrement = () => {
    if (item.quantity >= item.stock_available) {
      toast.error('Cannot add more - not enough stock');
      return;
    }

    updateQuantity.mutate(
      {
        productId: item.product_id,
        quantity: item.quantity + 1,
      },
      {
        onError: () => {
          toast.error('Failed to update quantity');
        },
      }
    );
  };

  const handleDecrement = () => {
    if (item.quantity <= item.min_order_quantity) {
      // Remove item if below minimum
      removeItem.mutate(item.product_id, {
        onSuccess: () => {
          toast.success('Item removed from cart');
        },
        onError: () => {
          toast.error('Failed to remove item');
        },
      });
      return;
    }

    updateQuantity.mutate(
      {
        productId: item.product_id,
        quantity: item.quantity - 1,
      },
      {
        onError: () => {
          toast.error('Failed to update quantity');
        },
      }
    );
  };

  const handleRemove = () => {
    removeItem.mutate(item.product_id, {
      onSuccess: () => {
        toast.success('Item removed from cart');
      },
      onError: () => {
        toast.error('Failed to remove item');
      },
    });
  };

  const isPending = updateQuantity.isPending || removeItem.isPending;

  return (
    <Card>
      <CardContent className="flex items-center gap-4 p-4">
        <div className="flex-1 min-w-0">
          <h3 className="font-semibold truncate">{item.product_name}</h3>
          <p className="text-sm text-muted-foreground">{item.product_sku}</p>
          <div className="mt-1 flex items-center gap-2">
            <span className="text-sm">€{item.unit_price.toFixed(2)} each</span>
            {!item.in_stock && (
              <Badge variant="destructive" className="text-xs">
                Low Stock
              </Badge>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8"
            onClick={handleDecrement}
            disabled={isPending}
          >
            <Minus className="h-3 w-3" />
          </Button>
          <span className="w-12 text-center font-medium">{item.quantity}</span>
          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8"
            onClick={handleIncrement}
            disabled={isPending || item.quantity >= item.stock_available}
          >
            <Plus className="h-3 w-3" />
          </Button>
        </div>

        <div className="w-24 text-right">
          <p className="font-semibold">€{item.line_total.toFixed(2)}</p>
        </div>

        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-muted-foreground hover:text-destructive"
          onClick={handleRemove}
          disabled={isPending}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </CardContent>
    </Card>
  );
}
