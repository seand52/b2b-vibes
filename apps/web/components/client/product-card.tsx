'use client';

import { useState } from 'react';
import Image from 'next/image';
import Link from 'next/link';
import { Plus, Minus, ShoppingCart } from 'lucide-react';
import { toast } from 'sonner';
import { Card, CardContent, CardFooter } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useAddToCart } from '@/lib/api/hooks';
import type { Product } from '@/lib/types';

interface ProductCardProps {
  product: Product;
}

export function ProductCard({ product }: ProductCardProps) {
  const [quantity, setQuantity] = useState(product.min_order_quantity || 1);
  const addToCart = useAddToCart();

  const primaryImage = product.images?.find((img) => img.is_primary) || product.images?.[0];
  const inStock = product.stock_quantity > 0;

  const handleAddToCart = async () => {
    try {
      await addToCart.mutateAsync({ productId: product.id, quantity });
      toast.success(`Added ${quantity}x ${product.name} to cart`);
    } catch {
      toast.error('Failed to add item to cart');
    }
  };

  const incrementQuantity = () => {
    if (quantity < product.stock_quantity) {
      setQuantity((q) => q + 1);
    }
  };

  const decrementQuantity = () => {
    if (quantity > (product.min_order_quantity || 1)) {
      setQuantity((q) => q - 1);
    }
  };

  return (
    <Card className="overflow-hidden" data-testid="product-card">
      <Link href={`/products/${product.id}`}>
        <div className="relative aspect-square bg-muted">
          {primaryImage ? (
            <Image
              src={primaryImage.url}
              alt={product.name}
              fill
              className="object-cover transition-transform hover:scale-105"
              sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 25vw"
            />
          ) : (
            <div className="flex h-full items-center justify-center text-muted-foreground">
              No image
            </div>
          )}
          {!inStock && (
            <div className="absolute inset-0 flex items-center justify-center bg-background/80">
              <Badge variant="secondary">Out of Stock</Badge>
            </div>
          )}
        </div>
      </Link>

      <CardContent className="p-4">
        <Link href={`/products/${product.id}`}>
          <h3 className="font-semibold hover:underline line-clamp-1">
            {product.name}
          </h3>
        </Link>
        <p className="text-sm text-muted-foreground line-clamp-1">
          {product.sku}
        </p>
        <div className="mt-2 flex items-baseline gap-2">
          <span className="text-lg font-bold">
            €{product.price.toFixed(2)}
          </span>
          <span className="text-xs text-muted-foreground">
            +{product.tax_rate}% VAT
          </span>
        </div>
        {inStock && (
          <p className="mt-1 text-xs text-muted-foreground">
            {product.stock_quantity} in stock
            {product.min_order_quantity > 1 && (
              <span> • Min: {product.min_order_quantity}</span>
            )}
          </p>
        )}
      </CardContent>

      <CardFooter className="flex flex-col gap-2 p-4 pt-0">
        {inStock && (
          <>
            <div className="flex w-full items-center justify-center gap-2">
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8"
                onClick={decrementQuantity}
                disabled={quantity <= (product.min_order_quantity || 1)}
              >
                <Minus className="h-3 w-3" />
              </Button>
              <span className="w-12 text-center font-medium">{quantity}</span>
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8"
                onClick={incrementQuantity}
                disabled={quantity >= product.stock_quantity}
              >
                <Plus className="h-3 w-3" />
              </Button>
            </div>
            <Button
              className="w-full"
              onClick={handleAddToCart}
              disabled={addToCart.isPending}
            >
              <ShoppingCart className="mr-2 h-4 w-4" />
              {addToCart.isPending ? 'Adding...' : 'Add to Cart'}
            </Button>
          </>
        )}
      </CardFooter>
    </Card>
  );
}
