"use client";

import { useState } from "react";
import Link from "next/link";
import ProductVisual from "@/components/product-visual";
import { COLOR_LABELS, GARMENT_LABELS, formatPrice } from "@/lib/constants";
import { useCart, useHydrated, useProducts, useSession } from "@/lib/hooks";
import {
  checkout,
  removeFromCart,
  setCartQty,
  startStripePayment,
} from "@/lib/store";
import type { StripePayment as StripePaymentInfo } from "@/lib/store";
import StripePayment from "@/components/stripe-payment";

const CartPage = () => {
  const cart = useCart();
  const products = useProducts();
  const session = useSession();
  const hydrated = useHydrated();

  const [showForm, setShowForm] = useState(false);
  const [placing, setPlacing] = useState(false);
  const [error, setError] = useState("");
  const [placed, setPlaced] = useState<
    { orderNumber: number; total: string; paid: boolean } | null
  >(null);
  const [payment, setPayment] = useState<
    (StripePaymentInfo & { orderNumber: number; total: string }) | null
  >(null);

  const [form, setForm] = useState({
    name: "",
    email: "",
    phone: "",
    line1: "",
    line2: "",
    city: "",
    state: "",
    postalCode: "",
  });
  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  if (!hydrated) return <div className="min-h-[60vh]" />;

  const lines = cart
    .map((item) => ({ item, product: products.find((p) => p.id === item.productId) }))
    .filter((line) => line.product !== undefined);

  const subtotal = lines.reduce(
    (sum, { item, product }) => sum + item.qty * product!.price,
    0,
  );

  const handleCheckout = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setPlacing(true);
    try {
      const result = await checkout({
        name: form.name || session?.name || "",
        email: form.email || session?.email || "",
        phone: form.phone,
        shippingAddress: {
          recipientName: form.name || session?.name || "",
          line1: form.line1,
          line2: form.line2,
          city: form.city,
          state: form.state,
          postalCode: form.postalCode,
        },
      });
      const pay = await startStripePayment(result.orderId);
      if (pay) {
        setPayment({ ...pay, orderNumber: result.orderNumber, total: result.total });
      } else {
        setPlaced({ orderNumber: result.orderNumber, total: result.total, paid: false });
      }
    } catch (err) {
      setError((err as { message?: string })?.message ?? "Could not place your order.");
    } finally {
      setPlacing(false);
    }
  };

  if (placed) {
    return (
      <div className="mx-auto flex min-h-[60vh] max-w-xl flex-col items-center justify-center gap-4 px-4 text-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-full bg-green-100 text-3xl">
          ✓
        </div>
        <h1 className="text-2xl font-bold">
          Order #{placed.orderNumber} {placed.paid ? "paid" : "placed"}!
        </h1>
        <p className="text-neutral-500">
          {placed.paid
            ? `Payment of ${placed.total} received — thanks for your order!`
            : `Your order total is ${placed.total}. It's awaiting payment — inventory has been reserved for you.`}
        </p>
        <Link
          href="/shop"
          className="rounded-lg bg-neutral-900 px-6 py-3 text-sm font-semibold text-white hover:bg-neutral-700"
        >
          Keep shopping
        </Link>
      </div>
    );
  }

  if (payment) {
    return (
      <div className="mx-auto flex min-h-[60vh] max-w-md flex-col justify-center px-4 py-12">
        <h1 className="text-2xl font-bold tracking-tight">Complete payment</h1>
        <p className="mt-1 text-sm text-neutral-500">
          Order #{payment.orderNumber} · {payment.total}
        </p>
        <StripePayment
          publishableKey={payment.publishableKey}
          clientSecret={payment.clientSecret}
          total={payment.total}
          onSuccess={() =>
            setPlaced({
              orderNumber: payment.orderNumber,
              total: payment.total,
              paid: true,
            })
          }
        />
      </div>
    );
  }

  if (lines.length === 0) {
    return (
      <div className="mx-auto flex min-h-[60vh] max-w-xl flex-col items-center justify-center gap-4 px-4 text-center">
        <h1 className="text-2xl font-bold">Your cart is empty</h1>
        <p className="text-neutral-500">Go find something that matches your stack.</p>
        <Link
          href="/shop"
          className="rounded-lg bg-neutral-900 px-6 py-3 text-sm font-semibold text-white hover:bg-neutral-700"
        >
          Browse the catalog
        </Link>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
      <h1 className="mb-8 text-2xl font-bold tracking-tight">
        Cart ({lines.length} {lines.length === 1 ? "item" : "items"})
      </h1>
      <div className="grid gap-8 lg:grid-cols-3">
        <div className="flex flex-col gap-4 lg:col-span-2">
          {lines.map(({ item, product }) => {
            const custom = {
              display: item.display,
              color: item.color,
              position: item.position,
            };
            return (
              <div
                key={`${item.productId}-${item.size}-${item.display ?? "icon"}-${item.color ?? ""}-${item.position ?? "center"}`}
                className="flex gap-4 rounded-xl border border-neutral-200 bg-white p-4"
              >
                <Link
                  href={`/product/${product!.id}`}
                  className="w-24 shrink-0 rounded-lg bg-neutral-100 p-2"
                >
                  <ProductVisual
                    product={product!}
                    display={item.display}
                    position={item.position}
                    color={item.color}
                    className="aspect-square w-full object-contain"
                  />
                </Link>
                <div className="flex flex-1 flex-col gap-1">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <Link
                        href={`/product/${product!.id}`}
                        className="font-medium hover:underline"
                      >
                        {product!.name}
                      </Link>
                      <p className="text-sm text-neutral-500">
                        {GARMENT_LABELS[product!.garment]} · Size {item.size}
                        {item.color && ` · ${COLOR_LABELS[item.color] ?? item.color}`}
                        {item.display === "icon-name" && " · Icon + name"}
                        {item.position && item.position !== "center" &&
                          ` · ${item.position === "left" ? "Left" : "Right"}`}
                      </p>
                    </div>
                    <button
                      onClick={() => removeFromCart(item.productId, item.size, custom)}
                      className="text-sm text-neutral-400 hover:text-red-600"
                    >
                      Remove
                    </button>
                  </div>
                  <div className="mt-auto flex items-center justify-between">
                    <div className="flex items-center rounded-lg border border-neutral-300">
                      <button
                        onClick={() =>
                          setCartQty(item.productId, item.size, item.qty - 1, custom)
                        }
                        className="px-3 py-1.5 hover:bg-neutral-100"
                      >
                        −
                      </button>
                      <span className="w-8 text-center text-sm">{item.qty}</span>
                      <button
                        onClick={() =>
                          setCartQty(item.productId, item.size, item.qty + 1, custom)
                        }
                        className="px-3 py-1.5 hover:bg-neutral-100"
                      >
                        +
                      </button>
                    </div>
                    <p className="font-semibold">{formatPrice(product!.price * item.qty)}</p>
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        <aside className="h-fit rounded-xl border border-neutral-200 bg-white p-6">
          <h2 className="mb-4 text-lg font-semibold">Summary</h2>
          <div className="flex flex-col gap-2 text-sm">
            <div className="flex justify-between">
              <span className="text-neutral-500">Subtotal</span>
              <span className="font-medium">{formatPrice(subtotal)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">Shipping</span>
              <span className="font-medium">Free</span>
            </div>
            <div className="mt-3 flex justify-between border-t border-neutral-200 pt-3 text-base font-bold">
              <span>Total</span>
              <span>{formatPrice(subtotal)}</span>
            </div>
          </div>

          {!showForm ? (
            <button
              onClick={() => {
                setShowForm(true);
                setForm((f) => ({
                  ...f,
                  name: f.name || session?.name || "",
                  email: f.email || session?.email || "",
                }));
              }}
              className="mt-6 w-full rounded-lg bg-neutral-900 px-6 py-3 text-sm font-semibold text-white hover:bg-neutral-700"
            >
              Checkout
            </button>
          ) : (
            <form onSubmit={handleCheckout} className="mt-6 flex flex-col gap-3">
              <Input placeholder="Full name" value={form.name} onChange={set("name")} required />
              <Input
                placeholder="Email"
                type="email"
                value={form.email}
                onChange={set("email")}
                required
              />
              <Input placeholder="Phone (optional)" value={form.phone} onChange={set("phone")} />
              <Input placeholder="Address line 1" value={form.line1} onChange={set("line1")} required />
              <Input
                placeholder="Address line 2 (optional)"
                value={form.line2}
                onChange={set("line2")}
              />
              <div className="grid grid-cols-2 gap-3">
                <Input placeholder="City" value={form.city} onChange={set("city")} required />
                <Input placeholder="State" value={form.state} onChange={set("state")} required />
              </div>
              <Input
                placeholder="Postal code"
                value={form.postalCode}
                onChange={set("postalCode")}
                required
              />
              {error && (
                <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">{error}</p>
              )}
              <button
                type="submit"
                disabled={placing}
                className="w-full rounded-lg bg-neutral-900 px-6 py-3 text-sm font-semibold text-white hover:bg-neutral-700 disabled:opacity-60"
              >
                {placing ? "Placing order…" : `Place order · ${formatPrice(subtotal)}`}
              </button>
            </form>
          )}

          <Link
            href="/shop"
            className="mt-3 block text-center text-sm text-neutral-500 hover:text-neutral-900"
          >
            Continue shopping
          </Link>
        </aside>
      </div>
    </div>
  );
};

const Input = (props: React.InputHTMLAttributes<HTMLInputElement>) => (
  <input
    {...props}
    className="rounded-lg border border-neutral-300 bg-white px-3 py-2 text-sm outline-none focus:border-neutral-500"
  />
);

export default CartPage;
