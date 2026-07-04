import { SEED_COLLECTIONS, SEED_PRODUCTS, SEED_USERS } from "./seed";
import type {
  CartItem,
  Collection,
  DesignDraft,
  Product,
  Session,
  Size,
  User,
} from "./types";

const KEYS = {
  products: "olmeware.products",
  collections: "olmeware.collections",
  users: "olmeware.users",
  cart: "olmeware.cart",
  session: "olmeware.session",
  draft: "olmeware.designDraft",
};

const STORE_EVENT = "olmeware:store";

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
  window.dispatchEvent(new Event(STORE_EVENT));
};

const remove = (key: string) => {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(key);
  window.dispatchEvent(new Event(STORE_EVENT));
};

const readSeeded = <T>(key: string, seed: T): T => {
  if (typeof window === "undefined") return seed;
  const raw = window.localStorage.getItem(key);
  if (raw === null) {
    window.localStorage.setItem(key, JSON.stringify(seed));
    return seed;
  }
  try {
    return JSON.parse(raw) as T;
  } catch {
    return seed;
  }
};

export const subscribe = (callback: () => void) => {
  window.addEventListener(STORE_EVENT, callback);
  window.addEventListener("storage", callback);
  return () => {
    window.removeEventListener(STORE_EVENT, callback);
    window.removeEventListener("storage", callback);
  };
};

export const getProducts = (): Product[] =>
  readSeeded(KEYS.products, SEED_PRODUCTS);

export const getProduct = (id: string): Product | undefined =>
  getProducts().find((p) => p.id === id);

export const saveProduct = (product: Product) => {
  const products = getProducts();
  const index = products.findIndex((p) => p.id === product.id);
  if (index >= 0) products[index] = product;
  else products.unshift(product);
  write(KEYS.products, products);
};

export const deleteProduct = (id: string) => {
  write(
    KEYS.products,
    getProducts().filter((p) => p.id !== id),
  );
};

export const getCollections = (): Collection[] =>
  readSeeded(KEYS.collections, SEED_COLLECTIONS);

export const saveCollection = (collection: Collection) => {
  const collections = getCollections();
  const index = collections.findIndex((c) => c.id === collection.id);
  if (index >= 0) collections[index] = collection;
  else collections.unshift(collection);
  write(KEYS.collections, collections);
};

export const deleteCollection = (id: string) => {
  write(
    KEYS.collections,
    getCollections().filter((c) => c.id !== id),
  );
  write(
    KEYS.products,
    getProducts().map((p) =>
      p.collectionId === id ? { ...p, collectionId: undefined } : p,
    ),
  );
};

const getUsers = (): User[] => readSeeded(KEYS.users, SEED_USERS);

export const registerUser = (
  name: string,
  email: string,
  password: string,
): { ok: true } | { ok: false; error: string } => {
  const users = getUsers();
  const normalized = email.trim().toLowerCase();
  if (users.some((u) => u.email === normalized)) {
    return { ok: false, error: "An account with this email already exists." };
  }
  users.push({
    id: crypto.randomUUID(),
    name: name.trim(),
    email: normalized,
    password,
    role: "customer",
    createdAt: new Date().toISOString(),
  });
  write(KEYS.users, users);
  return { ok: true };
};

export const login = (
  email: string,
  password: string,
): { ok: true; session: Session } | { ok: false; error: string } => {
  const normalized = email.trim().toLowerCase();
  const user = getUsers().find(
    (u) => u.email === normalized && u.password === password,
  );
  if (!user) return { ok: false, error: "Invalid email or password." };
  const session: Session = {
    userId: user.id,
    name: user.name,
    email: user.email,
    role: user.role,
  };
  write(KEYS.session, session);
  return { ok: true, session };
};

export const logout = () => remove(KEYS.session);

export const getSession = (): Session | null => read(KEYS.session, null);

export const getCart = (): CartItem[] => read(KEYS.cart, []);

export const addToCart = (productId: string, size: Size, qty: number) => {
  const cart = getCart();
  const item = cart.find((i) => i.productId === productId && i.size === size);
  if (item) item.qty += qty;
  else cart.push({ productId, size, qty });
  write(KEYS.cart, cart);
};

export const setCartQty = (productId: string, size: Size, qty: number) => {
  const cart = getCart()
    .map((i) =>
      i.productId === productId && i.size === size ? { ...i, qty } : i,
    )
    .filter((i) => i.qty > 0);
  write(KEYS.cart, cart);
};

export const removeFromCart = (productId: string, size: Size) => {
  write(
    KEYS.cart,
    getCart().filter((i) => !(i.productId === productId && i.size === size)),
  );
};

export const clearCart = () => write(KEYS.cart, []);

export const saveDesignDraft = (draft: DesignDraft) =>
  write(KEYS.draft, draft);

export const getDesignDraft = (): DesignDraft | null => read(KEYS.draft, null);

export const clearDesignDraft = () => remove(KEYS.draft);
