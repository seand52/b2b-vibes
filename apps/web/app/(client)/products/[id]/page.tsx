'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import Image from 'next/image';
import Link from 'next/link';
import { ArrowLeft, Plus, Minus, ShoppingCart } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Separator } from '@/components/ui/separator';
import { useProduct, useAddToCart } from '@/lib/api/hooks';

export default function ProductDetailPage() {
  const params = useParams();
  const productId = params.id as string;

  const { data: product, isLoading, error } = useProduct(productId);
  const addToCart = useAddToCart();

  const [quantity, setQuantity] = useState(1);
  const [selectedImageIndex, setSelectedImageIndex] = useState(0);

  // Reset quantity when product loads
  if (product && quantity < (product.min_order_quantity || 1)) {
    setQuantity(product.min_order_quantity || 1);
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-32" />
        <div className="grid gap-8 lg:grid-cols-2">
          <Skeleton className="aspect-square w-full rounded-lg" />
          <div className="space-y-4">
            <Skeleton className="h-10 w-3/4" />
            <Skeleton className="h-6 w-1/4" />
            <Skeleton className="h-24 w-full" />
          </div>
        </div>
      </div>
    );
  }

  if (error || !product) {
    return (
      <div className="space-y-6">
        <Link
          href="/products"
          className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Products
        </Link>
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
          <p className="text-destructive">
            Product not found or failed to load.
          </p>
        </div>
      </div>
    );
  }

  const inStock = product.stock_quantity > 0;
  const images = product.images || [];
  const selectedImage = images[selectedImageIndex];

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
    <div className="space-y-6">
      <Link
        href="/products"
        className="inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Products
      </Link>

      <div className="grid gap-8 lg:grid-cols-2">
        {/* Image Gallery */}
        <div className="space-y-4">
          <div className="relative aspect-square overflow-hidden rounded-lg bg-muted">
            {selectedImage ? (
              <Image
                src={selectedImage.url}
                alt={product.name}
                fill
                className="object-cover"
                sizes="(max-width: 1024px) 100vw, 50vw"
                priority
              />
            ) : (
              <div className="flex h-full items-center justify-center text-muted-foreground">
                No image available
              </div>
            )}
            {!inStock && (
              <div className="absolute inset-0 flex items-center justify-center bg-background/80">
                <Badge variant="secondary" className="text-lg">
                  Out of Stock
                </Badge>
              </div>
            )}
          </div>

          {images.length > 1 && (
            <div className="flex gap-2 overflow-x-auto pb-2">
              {images.map((image, index) => (
                <button
                  key={index}
                  onClick={() => setSelectedImageIndex(index)}
                  className={`relative h-20 w-20 flex-shrink-0 overflow-hidden rounded-md border-2 ${
                    index === selectedImageIndex
                      ? 'border-primary'
                      : 'border-transparent'
                  }`}
                >
                  <Image
                    src={image.url}
                    alt={`${product.name} - Image ${index + 1}`}
                    fill
                    className="object-cover"
                    sizes="80px"
                  />
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Product Info */}
        <div className="space-y-6">
          <div>
            <p className="text-sm text-muted-foreground">{product.sku}</p>
            <h1 className="mt-1 text-3xl font-bold">{product.name}</h1>
            {product.category && (
              <Badge variant="secondary" className="mt-2">
                {product.category}
              </Badge>
            )}
          </div>

          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-bold">€{product.price.toFixed(2)}</span>
            <span className="text-muted-foreground">+{product.tax_rate}% VAT</span>
          </div>

          {product.description && (
            <>
              <Separator />
              <div>
                <h2 className="mb-2 font-semibold">Description</h2>
                <p className="text-muted-foreground">{product.description}</p>
              </div>
            </>
          )}

          <Separator />

          <div className="space-y-4">
            <div className="flex items-center gap-4">
              <span className="text-sm font-medium">Stock:</span>
              {inStock ? (
                <Badge variant="outline" className="text-green-600">
                  {product.stock_quantity} available
                </Badge>
              ) : (
                <Badge variant="destructive">Out of Stock</Badge>
              )}
            </div>

            {product.min_order_quantity > 1 && (
              <div className="flex items-center gap-4">
                <span className="text-sm font-medium">Minimum Order:</span>
                <span className="text-muted-foreground">
                  {product.min_order_quantity} units
                </span>
              </div>
            )}
          </div>

          {inStock && (
            <>
              <Separator />

              <div className="space-y-4">
                <div className="flex items-center gap-4">
                  <span className="text-sm font-medium">Quantity:</span>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={decrementQuantity}
                      disabled={quantity <= (product.min_order_quantity || 1)}
                    >
                      <Minus className="h-4 w-4" />
                    </Button>
                    <span className="w-16 text-center font-medium text-lg">
                      {quantity}
                    </span>
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={incrementQuantity}
                      disabled={quantity >= product.stock_quantity}
                    >
                      <Plus className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                <div className="flex items-center gap-4">
                  <span className="text-sm font-medium">Subtotal:</span>
                  <span className="text-lg font-bold">
                    €{(product.price * quantity).toFixed(2)}
                  </span>
                </div>

                <Button
                  size="lg"
                  className="w-full"
                  onClick={handleAddToCart}
                  disabled={addToCart.isPending}
                >
                  <ShoppingCart className="mr-2 h-5 w-5" />
                  {addToCart.isPending ? 'Adding...' : 'Add to Cart'}
                </Button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
