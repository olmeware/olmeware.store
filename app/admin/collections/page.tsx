"use client";

import { useState } from "react";
import Link from "next/link";
import { useCollections, useProducts } from "@/lib/hooks";
import { deleteCollection, saveCollection } from "@/lib/store";

const slugify = (value: string) =>
  value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");

const AdminCollectionsPage = () => {
  const collections = useCollections();
  const products = useProducts();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      setError("Give the collection a name.");
      return;
    }
    if (collections.some((c) => c.slug === slugify(trimmed))) {
      setError("A collection with this name already exists.");
      return;
    }
    saveCollection({
      id: crypto.randomUUID(),
      name: trimmed,
      slug: slugify(trimmed),
      description: description.trim(),
      createdAt: new Date().toISOString(),
    });
    setName("");
    setDescription("");
    setError("");
  };

  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Collections</h1>
        <p className="text-sm text-neutral-500">
          Group merch into drops and themes shown on the storefront.
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <form
          onSubmit={handleCreate}
          className="flex h-fit flex-col gap-4 rounded-xl border border-neutral-200 bg-white p-6"
        >
          <h2 className="font-semibold">New collection</h2>
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            Name
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Summer Drop"
              className="rounded-lg border border-neutral-300 bg-white px-3 py-2 text-sm font-normal outline-none focus:border-neutral-500"
            />
          </label>
          <label className="flex flex-col gap-1.5 text-sm font-medium">
            Description
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              placeholder="What makes this drop special?"
              className="rounded-lg border border-neutral-300 bg-white px-3 py-2 text-sm font-normal outline-none focus:border-neutral-500"
            />
          </label>
          {error && (
            <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">
              {error}
            </p>
          )}
          <button
            type="submit"
            className="rounded-lg bg-neutral-900 px-5 py-2.5 text-sm font-semibold text-white hover:bg-neutral-700"
          >
            Create collection
          </button>
        </form>

        <div className="flex flex-col gap-4 lg:col-span-2">
          {collections.map((c) => {
            const count = products.filter((p) => p.collectionId === c.id).length;
            return (
              <div
                key={c.id}
                className="flex items-start justify-between gap-4 rounded-xl border border-neutral-200 bg-white p-5"
              >
                <div>
                  <h3 className="font-semibold">{c.name}</h3>
                  <p className="mt-0.5 text-sm text-neutral-500">
                    {c.description || "No description."}
                  </p>
                  <p className="mt-2 text-xs uppercase tracking-wide text-neutral-400">
                    {count} products · /{c.slug}
                  </p>
                </div>
                <div className="flex shrink-0 gap-3 text-sm">
                  <Link
                    href={`/shop?collection=${c.id}`}
                    className="font-medium text-neutral-600 hover:text-neutral-900"
                  >
                    View
                  </Link>
                  <button
                    onClick={() => {
                      if (
                        confirm(
                          `Delete "${c.name}"? Products in it are kept but ungrouped.`,
                        )
                      )
                        deleteCollection(c.id);
                    }}
                    className="font-medium text-neutral-400 hover:text-red-600"
                  >
                    Delete
                  </button>
                </div>
              </div>
            );
          })}
          {collections.length === 0 && (
            <div className="flex h-40 items-center justify-center rounded-xl border border-dashed border-neutral-300 text-sm text-neutral-500">
              No collections yet — create the first one.
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default AdminCollectionsPage;
