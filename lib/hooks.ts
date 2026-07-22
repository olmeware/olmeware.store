"use client";

import { useEffect, useMemo, useSyncExternalStore } from "react";
import {
  ensureAdminProducts,
  getAdminProducts,
  getCart,
  getCollections,
  getProducts,
  getSession,
  getSessionReady,
  subscribe,
} from "./store";
import type { Session } from "./api";
import type { CartItem, Collection, Product } from "./types";

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

// useAdminProducts returns every product (all statuses) for the admin panel.
export const useAdminProducts = (): Product[] => {
  useEffect(() => {
    void ensureAdminProducts();
  }, []);
  return useStoreValue(getAdminProducts, []);
};

export const useCollections = (): Collection[] =>
  useStoreValue(getCollections, []);

export const useCart = (): CartItem[] => useStoreValue(getCart, []);

export const useSession = (): Session | null =>
  useStoreValue(getSession, null);

// useSessionReady is true once the initial session hydration has finished,
// letting guards avoid redirecting before the session is known.
export const useSessionReady = (): boolean =>
  useSyncExternalStore(
    subscribe,
    () => getSessionReady(),
    () => false,
  );

export const useHydrated = (): boolean =>
  useSyncExternalStore(
    noopSubscribe,
    () => true,
    () => false,
  );
