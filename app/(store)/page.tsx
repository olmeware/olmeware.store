"use client";

import Link from "next/link";
import ProductCard from "@/components/product-card";
import { GARMENT_LABELS } from "@/lib/constants";
import { useCollections, useProducts } from "@/lib/hooks";
import type { GarmentType } from "@/lib/types";

const HomePage = () => {
  const products = useProducts().filter((p) => p.status === "active");
  const collections = useCollections();
  const featured = products.filter((p) => p.featured).slice(0, 4);
  const newest = [...products]
    .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
    .slice(0, 4);

  return (
    <div>
      <section className="bg-neutral-950 text-white">
        <div className="mx-auto flex max-w-7xl flex-col items-start gap-6 px-4 py-24 sm:px-6">
          <p className="rounded-full border border-neutral-700 px-3 py-1 text-xs uppercase tracking-widest text-neutral-400">
            Tech clothing & more
          </p>
          <h1 className="max-w-2xl text-4xl font-black tracking-tight sm:text-6xl">
            Wear your stack.
          </h1>
          <p className="max-w-xl text-lg text-neutral-400">
            Shirts, sweaters, hoodies, and caps for the languages, frameworks,
            and tools you actually ship with.
          </p>
          <div className="flex gap-3">
            <Link
              href="/shop"
              className="rounded-lg bg-white px-6 py-3 text-sm font-semibold text-neutral-950 transition hover:bg-neutral-200"
            >
              Shop the catalog
            </Link>
            <Link
              href="/shop?type=hoodie"
              className="rounded-lg border border-neutral-700 px-6 py-3 text-sm font-semibold text-white transition hover:border-neutral-400"
            >
              Hoodies
            </Link>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-14 sm:px-6">
        <h2 className="mb-6 text-xl font-bold tracking-tight">Shop by garment</h2>
        <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
          {(Object.keys(GARMENT_LABELS) as GarmentType[]).map((garment) => (
            <Link
              key={garment}
              href={`/shop?type=${garment}`}
              className="group rounded-xl border border-neutral-200 bg-white p-6 text-center transition hover:border-neutral-400 hover:shadow-md"
            >
              <p className="text-lg font-semibold group-hover:underline">
                {GARMENT_LABELS[garment]}s
              </p>
              <p className="mt-1 text-sm text-neutral-500">
                {products.filter((p) => p.garment === garment).length} products
              </p>
            </Link>
          ))}
        </div>
      </section>

      {featured.length > 0 && (
        <section className="mx-auto max-w-7xl px-4 py-6 sm:px-6">
          <div className="mb-6 flex items-end justify-between">
            <h2 className="text-xl font-bold tracking-tight">Featured</h2>
            <Link href="/shop" className="text-sm text-neutral-500 hover:text-neutral-900">
              View all →
            </Link>
          </div>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            {featured.map((product) => (
              <ProductCard key={product.id} product={product} />
            ))}
          </div>
        </section>
      )}

      <section className="mx-auto max-w-7xl px-4 py-14 sm:px-6">
        <h2 className="mb-6 text-xl font-bold tracking-tight">Collections</h2>
        <div className="grid gap-4 md:grid-cols-3">
          {collections.map((collection) => {
            const count = products.filter(
              (p) => p.collectionId === collection.id,
            ).length;
            return (
              <Link
                key={collection.id}
                href={`/shop?collection=${collection.id}`}
                className="group rounded-xl border border-neutral-200 bg-white p-6 transition hover:border-neutral-400 hover:shadow-md"
              >
                <h3 className="text-lg font-semibold group-hover:underline">
                  {collection.name}
                </h3>
                <p className="mt-1 text-sm text-neutral-500">
                  {collection.description}
                </p>
                <p className="mt-4 text-xs font-medium uppercase tracking-wide text-neutral-400">
                  {count} products
                </p>
              </Link>
            );
          })}
        </div>
      </section>

      {newest.length > 0 && (
        <section className="mx-auto max-w-7xl px-4 pb-10 sm:px-6">
          <div className="mb-6 flex items-end justify-between">
            <h2 className="text-xl font-bold tracking-tight">Just dropped</h2>
            <Link
              href="/shop?sort=newest"
              className="text-sm text-neutral-500 hover:text-neutral-900"
            >
              View all →
            </Link>
          </div>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            {newest.map((product) => (
              <ProductCard key={product.id} product={product} />
            ))}
          </div>
        </section>
      )}
    </div>
  );
};

export default HomePage;
