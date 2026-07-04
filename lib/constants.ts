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

export const GARMENT_COLORS = [
  "#f5f5f5",
  "#d4d4d4",
  "#8a8a8a",
  "#1a1a1a",
  "#1e2a44",
  "#3b6ea5",
  "#c0392b",
  "#2e7d4f",
  "#d9c7a7",
  "#5b3e8f",
  "#e6b93c",
  "#e39cc0",
];

export const formatPrice = (value: number) =>
  new Intl.NumberFormat("es-MX", {
    style: "currency",
    currency: "MXN",
    maximumFractionDigits: 0,
  }).format(value) + " MXN";
