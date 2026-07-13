import type { GarmentType, Side, Size, Stack } from "./types";

export const GARMENT_LABELS: Record<GarmentType, string> = {
  shirt: "Shirt",
  sweater: "Sweater",
  hoodie: "Hoodie",
  cap: "Cap",
};

export const SIDE_LABELS: Record<Side, string> = {
  front: "Front",
  back: "Back",
};

export const STACK_LABELS: Record<Stack, string> = {
  languages: "Languages",
  frontend: "Frontend",
  backend: "Backend",
  "ai-ml": "AI / ML",
  devops: "DevOps",
  databases: "Databases",
  cloud: "Cloud",
  tools: "Tools",
};

export const ALL_SIZES: Size[] = ["XS", "S", "M", "L", "XL", "XXL"];

export const GARMENT_COLORS = ["#1a1a1a", "#f5f5f5"];

export const COLOR_LABELS: Record<string, string> = {
  "#1a1a1a": "Black",
  "#f5f5f5": "White",
};

export const formatPrice = (value: number) =>
  new Intl.NumberFormat("es-MX", {
    style: "currency",
    currency: "MXN",
    maximumFractionDigits: 0,
  }).format(value) + " MXN";
