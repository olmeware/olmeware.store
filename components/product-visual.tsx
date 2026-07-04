"use client";

import { useId } from "react";
import type { Product, Side } from "@/lib/types";
import {
  GarmentBase,
  GarmentOverlay,
  PRINT_AREAS,
  VIEW_H,
  VIEW_W,
  clipPathsFor,
  isDarkColor,
} from "./garments";

type Props = {
  product: Product;
  side?: Side;
  image?: string;
  className?: string;
};

const ProductVisual = ({ product, side = "front", image, className }: Props) => {
  const clipId = useId();
  const src = image ?? product.images[0];

  if (src) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img src={src} alt={product.name} className={className} />
    );
  }

  const dark = isDarkColor(product.color);
  const area = PRINT_AREAS[product.garment][side];
  const pad = 0.12;

  return (
    <svg
      viewBox={`0 0 ${VIEW_W} ${VIEW_H}`}
      className={className}
      role="img"
      aria-label={product.name}
    >
      <defs>
        <clipPath id={clipId}>
          {clipPathsFor(product.garment, side).map((d, i) => (
            <path key={i} d={d} />
          ))}
        </clipPath>
      </defs>
      <GarmentBase
        garment={product.garment}
        side={side}
        color={product.color}
        dark={dark}
      />
      {product.logo && side === "front" && (
        <g clipPath={`url(#${clipId})`}>
          <image
            href={product.logo}
            x={area.x + area.w * pad}
            y={area.y + area.h * pad}
            width={area.w * (1 - pad * 2)}
            height={area.h * (1 - pad * 2)}
            preserveAspectRatio="xMidYMid meet"
          />
        </g>
      )}
      <GarmentOverlay
        garment={product.garment}
        side={side}
        color={product.color}
        dark={dark}
      />
    </svg>
  );
};

export default ProductVisual;
