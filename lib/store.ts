// Data layer backed by the Olmeware Go backend.
//
// Reads (products, collections, session) are fetched from the API and held in a
// module-level cache so the existing useSyncExternalStore hooks can read them
// synchronously; each async update dispatches `olmeware:store` to refresh open
// components. The cart and mockup design-draft stay in localStorage: the cart
// carries per-line print customization (icon display/position) that isn't a
// backend variant dimension, and the draft is a client-only editor handoff.

import { api, setGuestToken } from "./api";
import type { ApiError, BackendCollection, BackendProduct, Session } from "./api";
import type {
  CartItem,
  Collection,
  Customization,
  DesignDraft,
  Product,
  Size,
} from "./types";

const KEYS = {
  cart: "olmeware.cart",
  draft: "olmeware.designDraft",
};

const STORE_EVENT = "olmeware:store";

const emit = () => {
  if (typeof window !== "undefined") window.dispatchEvent(new Event(STORE_EVENT));
};

export const subscribe = (callback: () => void) => {
  window.addEventListener(STORE_EVENT, callback);
  window.addEventListener("storage", callback);
  // Kick off background loads when the first component subscribes.
  void ensureProducts();
  void ensureCollections();
  void hydrateSession();
  return () => {
    window.removeEventListener(STORE_EVENT, callback);
    window.removeEventListener("storage", callback);
  };
};

// ---- caches ----

let productsCache: Product[] = [];
let adminProductsCache: Product[] = [];
let collectionsCache: Collection[] = [];
let sessionCache: Session | null = null;
const slugById: Record<string, string> = {};

let productsLoaded = false;
let adminProductsLoaded = false;
let collectionsLoaded = false;
let sessionLoaded = false;
let productsInflight: Promise<void> | null = null;
let adminProductsInflight: Promise<void> | null = null;
let collectionsInflight: Promise<void> | null = null;
let sessionInflight: Promise<void> | null = null;

const logoSlug = (logo?: string) =>
  logo ? logo.replace(/^.*\//, "").replace(/\.svg$/, "") : "";

const mapCollection = (c: BackendCollection): Collection => ({
  id: c.id,
  name: c.name,
  slug: c.slug,
  description: c.description,
  createdAt: "",
});

const mapProduct = (p: BackendProduct): Product => {
  slugById[p.id] = p.slug;
  const collectionSlug = p.collections?.[0];
  const collectionId = collectionSlug
    ? collectionsCache.find((c) => c.slug === collectionSlug)?.id
    : undefined;
  return {
    id: p.id,
    name: p.name,
    description: p.description,
    garment: p.garment as Product["garment"],
    stack: p.stack as Product["stack"],
    tech: p.tech,
    price: Math.round(p.priceMinor / 100),
    sizes: p.sizes as Size[],
    color: p.colorHex,
    logo: p.logo,
    images: p.images ?? [],
    collectionId,
    featured: p.featured,
    status: p.status === "active" ? "active" : "draft",
    createdAt: "",
  };
};

async function ensureCollections(force = false): Promise<void> {
  if (collectionsInflight) return collectionsInflight;
  if (collectionsLoaded && !force) return;
  collectionsInflight = (async () => {
    try {
      const { collections } = await api.listCollections();
      collectionsCache = collections.map(mapCollection);
      collectionsLoaded = true;
      emit();
    } finally {
      collectionsInflight = null;
    }
  })();
  return collectionsInflight;
}

async function ensureProducts(force = false): Promise<void> {
  if (productsInflight) return productsInflight;
  if (productsLoaded && !force) return;
  productsInflight = (async () => {
    try {
      await ensureCollections();
      const { products } = await api.listProducts("limit=100");
      productsCache = products.map(mapProduct);
      productsLoaded = true;
      emit();
    } finally {
      productsInflight = null;
    }
  })();
  return productsInflight;
}

// ensureAdminProducts loads every product (all statuses) for the admin panel.
// Requires an admin session; a non-admin call simply leaves the cache empty.
export async function ensureAdminProducts(force = false): Promise<void> {
  if (adminProductsInflight) return adminProductsInflight;
  if (adminProductsLoaded && !force) return;
  adminProductsInflight = (async () => {
    try {
      await ensureCollections();
      const { products } = await api.listAdminProducts();
      adminProductsCache = products.map(mapProduct);
      adminProductsLoaded = true;
      emit();
    } catch {
      /* not an admin, or offline — leave cache as-is */
    } finally {
      adminProductsInflight = null;
    }
  })();
  return adminProductsInflight;
}

async function hydrateSession(force = false): Promise<void> {
  if (sessionInflight) return sessionInflight;
  if (sessionLoaded && !force) return;
  sessionInflight = (async () => {
    try {
      sessionCache = await api.me();
      sessionLoaded = true;
      emit();
    } finally {
      sessionInflight = null;
    }
  })();
  return sessionInflight;
}

// ---- catalog reads (synchronous cache access for hooks) ----

export const getProducts = (): Product[] => productsCache;

export const getAdminProducts = (): Product[] => adminProductsCache;

export const getProduct = (id: string): Product | undefined =>
  productsCache.find((p) => p.id === id);

export const getCollections = (): Collection[] => collectionsCache;

// ---- auth ----

export const getSession = (): Session | null => sessionCache;

// getSessionReady reports whether the initial /auth/me hydration has completed,
// so guards can distinguish "not logged in" from "still loading".
export const getSessionReady = (): boolean => sessionLoaded;

export const login = async (email: string, password: string): Promise<Session> => {
  const session = await api.login(email, password);
  sessionCache = session;
  sessionLoaded = true;
  emit();
  return session;
};

export const registerUser = async (
  name: string,
  email: string,
  password: string,
): Promise<Session> => {
  const session = await api.register(name, email, password);
  sessionCache = session;
  sessionLoaded = true;
  emit();
  return session;
};

export const logout = async (): Promise<void> => {
  sessionCache = null;
  sessionLoaded = true;
  emit();
  await api.logout();
};

// ---- admin writes ----

const toProductInput = (p: Product) => ({
  name: p.name,
  description: p.description,
  garment: p.garment,
  stack: p.stack,
  tech: p.tech,
  logoSlug: logoSlug(p.logo),
  price: p.price,
  colorHex: p.color,
  sizes: p.sizes,
  collectionSlug: p.collectionId
    ? (collectionsCache.find((c) => c.id === p.collectionId)?.slug ?? "")
    : "",
  featured: p.featured,
  status: p.status,
});

export const saveProduct = async (product: Product): Promise<void> => {
  const input = toProductInput(product);
  const existing = adminProductsCache.some((p) => p.id === product.id);
  if (existing) await api.updateProduct(product.id, input);
  else await api.createProduct(input);
  await Promise.all([ensureProducts(true), ensureAdminProducts(true)]);
};

export const deleteProduct = async (id: string): Promise<void> => {
  await api.deleteProductAdmin(id);
  await Promise.all([ensureProducts(true), ensureAdminProducts(true)]);
};

export const saveCollection = async (collection: Collection): Promise<void> => {
  const input = { name: collection.name, description: collection.description, sortOrder: 0 };
  const existing = collectionsCache.some((c) => c.id === collection.id);
  if (existing) await api.updateCollectionAdmin(collection.id, input);
  else await api.createCollectionAdmin(input);
  await ensureCollections(true);
};

export const deleteCollection = async (id: string): Promise<void> => {
  await api.deleteCollectionAdmin(id);
  await ensureCollections(true);
  await Promise.all([ensureProducts(true), ensureAdminProducts(true)]);
};

// ---- cart (localStorage: preserves per-line print customization) ----

const read = <T>(key: string, fallback: T): T => {
  if (typeof window === "undefined") return fallback;
  const raw = window.localStorage.getItem(key);
  if (raw === null) return fallback;
  try {
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
};

const write = (key: string, value: unknown) => {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(key, JSON.stringify(value));
  emit();
};

const remove = (key: string) => {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(key);
  emit();
};

export const getCart = (): CartItem[] => read(KEYS.cart, []);

const sameLine = (
  item: CartItem,
  productId: string,
  size: Size,
  custom: Customization,
) =>
  item.productId === productId &&
  item.size === size &&
  (item.display ?? "icon") === (custom.display ?? "icon") &&
  (item.color ?? "") === (custom.color ?? "") &&
  (item.position ?? "center") === (custom.position ?? "center");

export const addToCart = (
  productId: string,
  size: Size,
  qty: number,
  custom: Customization = {},
) => {
  const cart = getCart();
  const item = cart.find((i) => sameLine(i, productId, size, custom));
  if (item) item.qty += qty;
  else cart.push({ productId, size, qty, ...custom });
  write(KEYS.cart, cart);
};

export const setCartQty = (
  productId: string,
  size: Size,
  qty: number,
  custom: Customization = {},
) => {
  const cart = getCart()
    .map((i) => (sameLine(i, productId, size, custom) ? { ...i, qty } : i))
    .filter((i) => i.qty > 0);
  write(KEYS.cart, cart);
};

export const removeFromCart = (
  productId: string,
  size: Size,
  custom: Customization = {},
) => {
  write(
    KEYS.cart,
    getCart().filter((i) => !sameLine(i, productId, size, custom)),
  );
};

export const clearCart = () => write(KEYS.cart, []);

// ---- checkout ----

export type CheckoutDetails = {
  email: string;
  name: string;
  phone?: string;
  shippingAddress: {
    recipientName: string;
    line1: string;
    line2?: string;
    city: string;
    state: string;
    postalCode: string;
    countryCode?: string;
  };
};

export type CheckoutResult = {
  orderId: string;
  orderNumber: number;
  total: string;
};

// checkout resolves each local cart line to a backend variant, builds the
// server cart, creates the order (which reserves inventory), and clears the
// local cart. Returns the created order so the UI can start payment.
export const checkout = async (details: CheckoutDetails): Promise<CheckoutResult> => {
  const cart = getCart();
  if (cart.length === 0) throw { code: "empty_cart", message: "Your cart is empty." };

  await ensureProducts();

  // Start from a clean server cart, then add each line by resolved variant.
  try {
    await api.clearCart();
  } catch {
    /* no active cart yet — fine */
  }

  for (const line of cart) {
    const slug = slugById[line.productId];
    if (!slug) continue;
    const detail = await api.getProduct(slug);
    const variant =
      detail.variants?.find((v) => v.size === line.size) ?? detail.variants?.[0];
    if (!variant) {
      throw { code: "unavailable", message: `${detail.name} (${line.size}) is unavailable.` };
    }
    const res = await api.addCartItem(variant.id, line.qty);
    if (res?.guestToken) setGuestToken(res.guestToken);
  }

  const order = await api.createOrder({
    email: details.email,
    name: details.name,
    phone: details.phone,
    shippingAddress: {
      countryCode: "MX",
      ...details.shippingAddress,
    },
  });

  clearCart();
  return { orderId: order.id, orderNumber: order.orderNumber, total: order.total };
};

export type StripePayment = { clientSecret: string; publishableKey: string };

// startStripePayment creates (or reuses) a PaymentIntent for the order and
// returns what the card form needs. Returns null when Stripe is not configured
// so the caller can fall back to an "awaiting payment" confirmation.
export const startStripePayment = async (
  orderId: string,
): Promise<StripePayment | null> => {
  try {
    const res = await api.createStripeIntent(orderId);
    return { clientSecret: res.clientSecret, publishableKey: res.publishableKey };
  } catch (err) {
    if ((err as ApiError)?.code === "payments_unavailable") return null;
    throw err;
  }
};

// ---- design draft (localStorage: mockup editor -> new-product handoff) ----

export const saveDesignDraft = (draft: DesignDraft) => write(KEYS.draft, draft);

export const getDesignDraft = (): DesignDraft | null => read(KEYS.draft, null);

export const clearDesignDraft = () => remove(KEYS.draft);
