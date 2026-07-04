"use client";

import { Suspense, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import ProductCard from "@/components/product-card";
import {
  ALL_SIZES,
  GARMENT_LABELS,
  STACK_LABELS,
  formatPrice,
} from "@/lib/constants";
import { useCollections, useProducts } from "@/lib/hooks";
import type { GarmentType, Size, Stack } from "@/lib/types";

type SortKey = "newest" | "price-asc" | "price-desc" | "name";

const SORT_LABELS: Record<SortKey, string> = {
  newest: "Newest",
  "price-asc": "Price: low to high",
  "price-desc": "Price: high to low",
  name: "Name",
};

const toggle = <T,>(list: T[], value: T): T[] =>
  list.includes(value) ? list.filter((v) => v !== value) : [...list, value];

const FilterGroup = ({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) => (
  <div className="border-b border-neutral-200 pb-5">
    <p className="mb-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">
      {title}
    </p>
    {children}
  </div>
);

const ShopContent = () => {
  const searchParams = useSearchParams();
  const products = useProducts().filter((p) => p.status === "active");
  const collections = useCollections();

  const initialType = searchParams.get("type") as GarmentType | null;
  const [types, setTypes] = useState<GarmentType[]>(
    initialType && initialType in GARMENT_LABELS ? [initialType] : [],
  );
  const [stacks, setStacks] = useState<Stack[]>([]);
  const [sizes, setSizes] = useState<Size[]>([]);
  const [collectionId, setCollectionId] = useState(
    searchParams.get("collection") ?? "",
  );
  const [maxPrice, setMaxPrice] = useState(1000);
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<SortKey>(
    searchParams.get("sort") === "newest" ? "newest" : "newest",
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const result = products.filter(
      (p) =>
        (types.length === 0 || types.includes(p.garment)) &&
        (stacks.length === 0 || stacks.includes(p.stack)) &&
        (sizes.length === 0 || p.sizes.some((s) => sizes.includes(s))) &&
        (collectionId === "" || p.collectionId === collectionId) &&
        p.price <= maxPrice &&
        (q === "" ||
          p.name.toLowerCase().includes(q) ||
          p.tech.toLowerCase().includes(q)),
    );
    const sorters: Record<SortKey, (a: typeof result[number], b: typeof result[number]) => number> = {
      newest: (a, b) => b.createdAt.localeCompare(a.createdAt),
      "price-asc": (a, b) => a.price - b.price,
      "price-desc": (a, b) => b.price - a.price,
      name: (a, b) => a.name.localeCompare(b.name),
    };
    return result.sort(sorters[sort]);
  }, [products, types, stacks, sizes, collectionId, maxPrice, query, sort]);

  const clearFilters = () => {
    setTypes([]);
    setStacks([]);
    setSizes([]);
    setCollectionId("");
    setMaxPrice(1000);
    setQuery("");
  };

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Catalog</h1>
          <p className="text-sm text-neutral-500">
            {filtered.length} of {products.length} products
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search tech or product…"
            className="w-56 rounded-lg border border-neutral-300 bg-white px-3 py-2 text-sm outline-none focus:border-neutral-500"
          />
          <select
            value={sort}
            onChange={(e) => setSort(e.target.value as SortKey)}
            className="rounded-lg border border-neutral-300 bg-white px-3 py-2 text-sm outline-none focus:border-neutral-500"
          >
            {(Object.keys(SORT_LABELS) as SortKey[]).map((key) => (
              <option key={key} value={key}>
                {SORT_LABELS[key]}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="flex flex-col gap-8 lg:flex-row">
        <aside className="w-full shrink-0 lg:w-60">
          <div className="flex flex-col gap-5 rounded-xl border border-neutral-200 bg-white p-5">
            <FilterGroup title="Garment">
              <div className="flex flex-col gap-2">
                {(Object.keys(GARMENT_LABELS) as GarmentType[]).map((g) => (
                  <label key={g} className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={types.includes(g)}
                      onChange={() => setTypes((t) => toggle(t, g))}
                    />
                    {GARMENT_LABELS[g]}
                  </label>
                ))}
              </div>
            </FilterGroup>

            <FilterGroup title="Stack">
              <div className="flex flex-col gap-2">
                {(Object.keys(STACK_LABELS) as Stack[]).map((s) => (
                  <label key={s} className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={stacks.includes(s)}
                      onChange={() => setStacks((st) => toggle(st, s))}
                    />
                    {STACK_LABELS[s]}
                  </label>
                ))}
              </div>
            </FilterGroup>

            <FilterGroup title="Size">
              <div className="flex flex-wrap gap-2">
                {ALL_SIZES.map((s) => (
                  <button
                    key={s}
                    onClick={() => setSizes((sz) => toggle(sz, s))}
                    className={`rounded-lg border px-2.5 py-1 text-xs font-medium ${
                      sizes.includes(s)
                        ? "border-neutral-900 bg-neutral-900 text-white"
                        : "border-neutral-300 hover:border-neutral-500"
                    }`}
                  >
                    {s}
                  </button>
                ))}
              </div>
            </FilterGroup>

            <FilterGroup title="Collection">
              <select
                value={collectionId}
                onChange={(e) => setCollectionId(e.target.value)}
                className="w-full rounded-lg border border-neutral-300 bg-white px-2 py-1.5 text-sm outline-none focus:border-neutral-500"
              >
                <option value="">All collections</option>
                {collections.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </FilterGroup>

            <FilterGroup title={`Max price · ${formatPrice(maxPrice)}`}>
              <input
                type="range"
                min={200}
                max={1000}
                step={50}
                value={maxPrice}
                onChange={(e) => setMaxPrice(Number(e.target.value))}
                className="w-full"
              />
            </FilterGroup>

            <button
              onClick={clearFilters}
              className="rounded-lg border border-neutral-300 px-3 py-2 text-sm font-medium hover:border-neutral-500"
            >
              Clear filters
            </button>
          </div>
        </aside>

        <div className="flex-1">
          {filtered.length === 0 ? (
            <div className="flex h-64 flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-neutral-300 text-neutral-500">
              <p className="font-medium">No products match your filters.</p>
              <button onClick={clearFilters} className="text-sm underline">
                Clear filters
              </button>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-4 xl:grid-cols-3">
              {filtered.map((product) => (
                <ProductCard key={product.id} product={product} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

const ShopPage = () => (
  <Suspense>
    <ShopContent />
  </Suspense>
);

export default ShopPage;
