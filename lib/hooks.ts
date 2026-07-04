"use client";

import { useMemo, useSyncExternalStore } from "react";
import {
  getCart,
  getCollections,
  getProducts,
  getSession,
  subscribe,
} from "./store";
import type { CartItem, Collection, Product, Session } from "./types";

const noopSubscribe = () => () => {};

const useStoreValue = <T>(getter: () => T, fallback: T): T => {
  const raw = useSyncExternalStore(
    subscribe,
    () => JSON.stringify(getter()),
    () => JSON.stringify(fallback),
  );
  return useMemo(() => JSON.parse(raw) as T, [raw]);
};

export const useProducts = (): Product[] => useStoreValue(getProducts, []);

export const useCollections = (): Collection[] =>
  useStoreValue(getCollections, []);

export const useCart = (): CartItem[] => useStoreValue(getCart, []);

export const useSession = (): Session | null =>
  useStoreValue(getSession, null);

export const useHydrated = (): boolean =>
  useSyncExternalStore(
    noopSubscribe,
    () => true,
    () => false,
  );
