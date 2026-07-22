// Typed client for the Olmeware Go backend. All storefront/admin data flows
// through here; the backend composes responses and this layer only shapes them
// for the UI. Base URL is configurable via NEXT_PUBLIC_API_URL.

const BASE =
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "") ??
  "http://localhost:8000/api/v1";

const TOKENS = {
  access: "olmeware.accessToken",
  refresh: "olmeware.refreshToken",
  guest: "olmeware.guestToken",
};

export type ApiError = { code: string; message: string };

const ls = (): Storage | null =>
  typeof window === "undefined" ? null : window.localStorage;

export const getAccessToken = () => ls()?.getItem(TOKENS.access) ?? null;
export const getRefreshToken = () => ls()?.getItem(TOKENS.refresh) ?? null;
export const getGuestToken = () => ls()?.getItem(TOKENS.guest) ?? null;

export const setTokens = (access: string, refresh: string) => {
  ls()?.setItem(TOKENS.access, access);
  ls()?.setItem(TOKENS.refresh, refresh);
};

export const clearTokens = () => {
  ls()?.removeItem(TOKENS.access);
  ls()?.removeItem(TOKENS.refresh);
};

export const setGuestToken = (token: string) => {
  if (token) ls()?.setItem(TOKENS.guest, token);
};

type FetchOpts = {
  method?: string;
  body?: unknown;
  auth?: boolean; // attach the access token
  guest?: boolean; // attach the guest cart token
  _retried?: boolean;
};

async function raw<T>(path: string, opts: FetchOpts = {}): Promise<T> {
  const headers: Record<string, string> = {};
  if (opts.body !== undefined) headers["Content-Type"] = "application/json";
  if (opts.auth) {
    const token = getAccessToken();
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }
  if (opts.guest) {
    const gt = getGuestToken();
    if (gt) headers["X-Guest-Token"] = gt;
  }

  const res = await fetch(`${BASE}${path}`, {
    method: opts.method ?? "GET",
    headers,
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
  });

  // Transparently refresh an expired access token once.
  if (res.status === 401 && opts.auth && !opts._retried && getRefreshToken()) {
    if (await tryRefresh()) {
      return raw<T>(path, { ...opts, _retried: true });
    }
  }

  if (res.status === 204) return undefined as T;

  const data = await res.json().catch(() => null);
  if (!res.ok) {
    const err: ApiError = data?.error ?? {
      code: "network_error",
      message: "Request failed.",
    };
    throw err;
  }
  return data as T;
}

let refreshing: Promise<boolean> | null = null;
async function tryRefresh(): Promise<boolean> {
  if (refreshing) return refreshing;
  refreshing = (async () => {
    const refreshToken = getRefreshToken();
    if (!refreshToken) return false;
    try {
      const res = await fetch(`${BASE}/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refreshToken }),
      });
      if (!res.ok) {
        clearTokens();
        return false;
      }
      const data = await res.json();
      setTokens(data.accessToken, data.refreshToken);
      return true;
    } catch {
      return false;
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

// ---- Backend response shapes (only the fields the UI needs) ----

export type BackendProduct = {
  id: string;
  slug: string;
  name: string;
  description: string;
  garment: string;
  stack: string;
  tech: string;
  logo?: string;
  colorHex: string;
  priceMinor: number;
  featured: boolean;
  status: string;
  sizes: string[];
  images: string[];
  collections: string[];
  variants?: BackendVariant[];
};

export type BackendVariant = {
  id: string;
  sku: string;
  size: string;
  colorHex: string;
  priceMinor: number;
  inStock: boolean;
  available: number;
};

export type BackendCollection = {
  id: string;
  slug: string;
  name: string;
  description: string;
  productCount: number;
};

export type Session = {
  userId: string;
  name: string;
  email: string;
  role: "admin" | "customer";
};

type AuthResponse = {
  user: { id: string; name: string; email: string; role: "admin" | "customer" };
  accessToken: string;
  refreshToken: string;
};

const asSession = (a: AuthResponse): Session => ({
  userId: a.user.id,
  name: a.user.name,
  email: a.user.email,
  role: a.user.role,
});

export const api = {
  // Auth
  async register(name: string, email: string, password: string): Promise<Session> {
    const a = await raw<AuthResponse>("/auth/register", {
      method: "POST",
      body: { name, email, password },
    });
    setTokens(a.accessToken, a.refreshToken);
    return asSession(a);
  },
  async login(email: string, password: string): Promise<Session> {
    const a = await raw<AuthResponse>("/auth/login", {
      method: "POST",
      body: { email, password },
    });
    setTokens(a.accessToken, a.refreshToken);
    return asSession(a);
  },
  async logout(): Promise<void> {
    const refreshToken = getRefreshToken();
    clearTokens();
    if (refreshToken) {
      try {
        await raw("/auth/logout", { method: "POST", body: { refreshToken } });
      } catch {
        /* best effort */
      }
    }
  },
  async me(): Promise<Session | null> {
    if (!getAccessToken() && !getRefreshToken()) return null;
    try {
      const u = await raw<AuthResponse["user"]>("/auth/me", { auth: true });
      return { userId: u.id, name: u.name, email: u.email, role: u.role };
    } catch {
      // Stale/invalid session — drop the tokens so we act as a clean guest.
      clearTokens();
      return null;
    }
  },

  // Catalog
  listProducts(query = ""): Promise<{ products: BackendProduct[] }> {
    return raw(`/catalog/products${query ? `?${query}` : ""}`);
  },
  getProduct(slug: string): Promise<BackendProduct> {
    return raw(`/catalog/products/${slug}`);
  },
  listCollections(): Promise<{ collections: BackendCollection[] }> {
    return raw("/catalog/collections");
  },

  // Cart (server-side; used at checkout)
  addCartItem(variantId: string, quantity: number) {
    return raw<{ guestToken?: string }>("/cart/items", {
      method: "POST",
      body: { variantId, quantity },
      auth: true,
      guest: true,
    });
  },
  clearCart() {
    return raw("/cart", { method: "DELETE", auth: true, guest: true });
  },

  // Orders
  createOrder(payload: unknown): Promise<{ id: string; orderNumber: number; total: string; status: string }> {
    return raw("/orders", { method: "POST", body: payload, auth: true, guest: true });
  },
  listOrders() {
    return raw<{ orders: unknown[] }>("/orders", { auth: true });
  },

  // Payments
  paymentConfig(): Promise<{
    stripe: { enabled: boolean; publishableKey: string };
    crypto: { enabled: boolean; coins: string[] };
    currency: string;
  }> {
    return raw("/payments/config");
  },
  createStripeIntent(orderId: string) {
    return raw<{ clientSecret: string; publishableKey: string; status: string; reused: boolean }>(
      "/payments/stripe/intent",
      { method: "POST", body: { orderId }, auth: true, guest: true },
    );
  },
  createCryptoCharge(orderId: string) {
    return raw<{ hostedUrl: string; chargeCode: string; coins: string[] }>(
      "/payments/crypto/charge",
      { method: "POST", body: { orderId }, auth: true, guest: true },
    );
  },

  // Admin
  listAdminProducts(): Promise<{ products: BackendProduct[] }> {
    return raw("/admin/products", { auth: true });
  },
  createProduct(input: unknown) {
    return raw<{ id: string }>("/admin/products", { method: "POST", body: input, auth: true });
  },
  updateProduct(id: string, input: unknown) {
    return raw(`/admin/products/${id}`, { method: "PUT", body: input, auth: true });
  },
  deleteProductAdmin(id: string) {
    return raw(`/admin/products/${id}`, { method: "DELETE", auth: true });
  },
  createCollectionAdmin(input: unknown) {
    return raw<{ id: string }>("/admin/collections", { method: "POST", body: input, auth: true });
  },
  updateCollectionAdmin(id: string, input: unknown) {
    return raw(`/admin/collections/${id}`, { method: "PUT", body: input, auth: true });
  },
  deleteCollectionAdmin(id: string) {
    return raw(`/admin/collections/${id}`, { method: "DELETE", auth: true });
  },
};
