"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import ProductCard from "@/components/product-card";
import ProductVisual from "@/components/product-visual";
import {
  COLOR_LABELS,
  GARMENT_COLORS,
  GARMENT_LABELS,
  SIDE_LABELS,
  STACK_LABELS,
  formatPrice,
} from "@/lib/constants";
import { useHydrated, useProducts } from "@/lib/hooks";
import { addToCart } from "@/lib/store";
import type { IconDisplay, IconPosition, Side, Size } from "@/lib/types";

const DISPLAY_OPTIONS: { value: IconDisplay; label: string }[] = [
  { value: "icon", label: "Icon only" },
  { value: "icon-name", label: "Icon + name" },
];

const POSITION_OPTIONS: { value: IconPosition; label: string }[] = [
  { value: "left", label: "Left" },
  { value: "center", label: "Center" },
  { value: "right", label: "Right" },
];

const OptionButton = ({
  active,
  onClick,
  className = "",
  children,
}: {
  active: boolean;
  onClick: () => void;
  className?: string;
  children: React.ReactNode;
}) => (
  <button
    onClick={onClick}
    className={`rounded-lg border px-3 py-2 text-sm font-medium ${
      active
        ? "border-neutral-900 bg-neutral-900 text-white"
        : "border-neutral-300 hover:border-neutral-500"
    } ${className}`}
  >
    {children}
  </button>
);

type View = { kind: "svg"; side: Side } | { kind: "image"; src: string };

const ProductPage = () => {
  const { id } = useParams<{ id: string }>();
  const products = useProducts();
  const hydrated = useHydrated();
  const product = products.find((p) => p.id === id);

  const views = useMemo<View[]>(() => {
    if (!product) return [];
    if (product.images.length > 0) {
      return product.images.map((src) => ({ kind: "image", src }));
    }
    return [
      { kind: "svg", side: "front" },
      { kind: "svg", side: "back" },
    ];
  }, [product]);

  const [viewIndex, setViewIndex] = useState(0);
  const [size, setSize] = useState<Size | null>(null);
  const [qty, setQty] = useState(1);
  const [display, setDisplay] = useState<IconDisplay>("icon");
  const [position, setPosition] = useState<IconPosition>("center");
  const [pickedColor, setPickedColor] = useState<string | null>(null);
  const [added, setAdded] = useState(false);

  if (!hydrated) return <div className="min-h-[60vh]" />;

  if (!product) {
    return (
      <div className="mx-auto flex min-h-[60vh] max-w-7xl flex-col items-center justify-center gap-3 px-4">
        <h1 className="text-2xl font-bold">Product not found</h1>
        <Link href="/shop" className="text-sm underline">
          Back to the catalog
        </Link>
      </div>
    );
  }

  const related = products
    .filter(
      (p) => p.status === "active" && p.id !== product.id && p.stack === product.stack,
    )
    .slice(0, 4);
  const view = views[Math.min(viewIndex, views.length - 1)];

  const customizable = product.images.length === 0 && Boolean(product.logo);
  const color =
    pickedColor ??
    (GARMENT_COLORS.includes(product.color) ? product.color : GARMENT_COLORS[0]);
  const isCap = product.garment === "cap";

  const handleAdd = () => {
    if (!size) return;
    addToCart(
      product.id,
      size,
      qty,
      customizable ? { display, color, position } : {},
    );
    setAdded(true);
    setTimeout(() => setAdded(false), 2000);
  };

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
      <nav className="mb-6 text-sm text-neutral-500">
        <Link href="/" className="hover:text-neutral-900">Home</Link>
        {" / "}
        <Link href="/shop" className="hover:text-neutral-900">Catalog</Link>
        {" / "}
        <span className="text-neutral-900">{product.name}</span>
      </nav>

      <div className="grid gap-10 lg:grid-cols-2">
        <div>
          <div className="rounded-xl border border-neutral-200 bg-neutral-100 p-6">
            {view.kind === "svg" ? (
              <ProductVisual
                product={product}
                side={view.side}
                display={display}
                position={position}
                color={color}
                className="aspect-square w-full object-contain"
              />
            ) : (
              <ProductVisual
                product={product}
                image={view.src}
                className="aspect-square w-full object-contain"
              />
            )}
          </div>
          <div className="mt-4 flex gap-3">
            {views.map((v, i) => (
              <button
                key={i}
                onClick={() => setViewIndex(i)}
                className={`w-20 rounded-lg border bg-neutral-100 p-2 ${
                  i === viewIndex
                    ? "border-neutral-900"
                    : "border-neutral-200 hover:border-neutral-400"
                }`}
              >
                {v.kind === "svg" ? (
                  <ProductVisual
                    product={product}
                    side={v.side}
                    display={display}
                    position={position}
                    color={color}
                    className="aspect-square w-full"
                  />
                ) : (
                  <ProductVisual
                    product={product}
                    image={v.src}
                    className="aspect-square w-full object-contain"
                  />
                )}
                <span className="mt-1 block text-center text-xs text-neutral-500">
                  {v.kind === "svg" ? SIDE_LABELS[v.side] : `View ${i + 1}`}
                </span>
              </button>
            ))}
          </div>
        </div>

        <div className="flex flex-col gap-5">
          <div className="flex flex-wrap gap-2 text-xs font-medium uppercase tracking-wide">
            <span className="rounded-full bg-neutral-900 px-3 py-1 text-white">
              {GARMENT_LABELS[product.garment]}
            </span>
            <span className="rounded-full bg-neutral-200 px-3 py-1 text-neutral-700">
              {STACK_LABELS[product.stack]}
            </span>
            <span className="rounded-full bg-sky-100 px-3 py-1 text-sky-700">
              {product.tech}
            </span>
          </div>

          <div>
            <h1 className="text-3xl font-bold tracking-tight">{product.name}</h1>
            <p className="mt-2 text-2xl font-semibold">
              {formatPrice(product.price)}
            </p>
          </div>

          <p className="text-neutral-600">{product.description}</p>

          <div>
            <p className="mb-2 text-sm font-semibold">Size</p>
            <div className="flex flex-wrap gap-2">
              {product.sizes.map((s) => (
                <OptionButton
                  key={s}
                  active={size === s}
                  onClick={() => setSize(s)}
                  className="min-w-12"
                >
                  {s}
                </OptionButton>
              ))}
            </div>
          </div>

          {customizable && (
            <>
              <div>
                <p className="mb-2 text-sm font-semibold">Color</p>
                <div className="flex flex-wrap gap-2">
                  {GARMENT_COLORS.map((c) => (
                    <OptionButton
                      key={c}
                      active={color === c}
                      onClick={() => setPickedColor(c)}
                      className="flex items-center gap-2"
                    >
                      <span
                        className="h-4 w-4 rounded-full border border-neutral-400"
                        style={{ backgroundColor: c }}
                      />
                      {COLOR_LABELS[c]}
                    </OptionButton>
                  ))}
                </div>
              </div>

              <div>
                <p className="mb-2 text-sm font-semibold">Icon display</p>
                <div className="flex flex-wrap gap-2">
                  {DISPLAY_OPTIONS.map((opt) => (
                    <OptionButton
                      key={opt.value}
                      active={display === opt.value}
                      onClick={() => setDisplay(opt.value)}
                    >
                      {opt.label}
                    </OptionButton>
                  ))}
                </div>
              </div>

              {!isCap && (
                <div>
                  <p className="mb-2 text-sm font-semibold">Icon position</p>
                  <div className="flex flex-wrap gap-2">
                    {POSITION_OPTIONS.map((opt) => (
                      <OptionButton
                        key={opt.value}
                        active={position === opt.value}
                        onClick={() => setPosition(opt.value)}
                      >
                        {opt.label}
                      </OptionButton>
                    ))}
                  </div>
                  <p className="mt-2 text-xs text-neutral-500">
                    Printed on the front, just below the neckline.
                  </p>
                </div>
              )}
            </>
          )}

          <div>
            <p className="mb-2 text-sm font-semibold">Quantity</p>
            <div className="flex w-fit items-center rounded-lg border border-neutral-300">
              <button
                onClick={() => setQty((q) => Math.max(1, q - 1))}
                className="px-4 py-2 text-lg hover:bg-neutral-100"
              >
                −
              </button>
              <span className="w-10 text-center text-sm font-medium">{qty}</span>
              <button
                onClick={() => setQty((q) => Math.min(99, q + 1))}
                className="px-4 py-2 text-lg hover:bg-neutral-100"
              >
                +
              </button>
            </div>
          </div>

          <button
            onClick={handleAdd}
            disabled={!size}
            className="rounded-lg bg-neutral-900 px-8 py-3.5 text-sm font-semibold text-white transition hover:bg-neutral-700 disabled:cursor-not-allowed disabled:bg-neutral-300"
          >
            {added
              ? "Added to cart ✓"
              : size
                ? "Add to cart"
                : "Select a size"}
          </button>

          <p className="text-sm text-neutral-500">
            Made to order. Production begins after your order is confirmed.
          </p>
        </div>
      </div>

      {related.length > 0 && (
        <section className="mt-16">
          <h2 className="mb-6 text-xl font-bold tracking-tight">
            More from {STACK_LABELS[product.stack]}
          </h2>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            {related.map((p) => (
              <ProductCard key={p.id} product={p} />
            ))}
          </div>
        </section>
      )}
    </div>
  );
};

export default ProductPage;
