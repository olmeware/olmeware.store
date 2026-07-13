"use client";

import { useId } from "react";
import type { IconDisplay, IconPosition, Product, Side } from "@/lib/types";
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
  display?: IconDisplay;
  position?: IconPosition;
  color?: string;
  className?: string;
};

const AVG_CHAR_WIDTH = 0.62;

const ProductVisual = ({
  product,
  side = "front",
  image,
  display = "icon",
  position = "center",
  color,
  className,
}: Props) => {
  const clipId = useId();
  const src = image ?? product.images[0];

  if (src) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img src={src} alt={product.name} className={className} />
    );
  }

  const fill = color ?? product.color;
  const dark = isDarkColor(fill);
  const area = PRINT_AREAS[product.garment][side];
  const label = product.tech;

  // Caps have no neck: designs stay centered in the crown's print area.
  const isCap = product.garment === "cap";
  const pad = 0.12;
  const inner = {
    x: area.x + area.w * pad,
    y: area.y + area.h * pad,
    w: area.w * (1 - pad * 2),
    h: area.h * (1 - pad * 2),
  };

  // Chest row: top edge sits ~4-8cm below the collar (per /public/clothing refs).
  const chestY = area.y + area.h * 0.04;

  const iconOnlySize = isCap
    ? Math.min(inner.w, inner.h)
    : area.w * (position === "center" ? 0.5 : 0.28);
  // "left"/"right" follow apparel convention (wearer-relative): the wearer's
  // left chest appears on the viewer's right in these front-facing mockups.
  const iconOnlyX = isCap
    ? inner.x + (inner.w - iconOnlySize) / 2
    : position === "left"
      ? area.x + area.w - iconOnlySize
      : position === "right"
        ? area.x
        : area.x + (area.w - iconOnlySize) / 2;
  const iconOnlyY = isCap ? inner.y + (inner.h - iconOnlySize) / 2 : chestY;

  const iconSize = isCap
    ? Math.min(inner.h * 0.8, inner.w * 0.32)
    : area.w * (position === "center" ? 0.32 : 0.24);
  const gap = iconSize * 0.2;
  const maxTextW = (isCap ? inner.w : area.w) - iconSize - gap;
  const fontSize = Math.min(
    iconSize * 0.55,
    maxTextW / (AVG_CHAR_WIDTH * label.length),
  );
  const rowWidth = iconSize + gap + fontSize * AVG_CHAR_WIDTH * label.length;
  const rowX = isCap
    ? inner.x + (inner.w - rowWidth) / 2
    : position === "left"
      ? area.x + area.w - rowWidth
      : position === "right"
        ? area.x
        : area.x + (area.w - rowWidth) / 2;
  const rowCy = isCap ? inner.y + inner.h / 2 : chestY + iconSize / 2;

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
        color={fill}
        dark={dark}
      />
      {product.logo && side === "front" && (
        <g clipPath={`url(#${clipId})`}>
          {display === "icon-name" ? (
            <>
              <image
                href={product.logo}
                x={rowX}
                y={rowCy - iconSize / 2}
                width={iconSize}
                height={iconSize}
                preserveAspectRatio="xMidYMid meet"
              />
              <text
                x={rowX + iconSize + gap}
                y={rowCy}
                dominantBaseline="central"
                fontFamily="var(--font-roboto), Roboto, Arial, sans-serif"
                fontSize={fontSize}
                fontWeight={700}
                fill={dark ? "#fafafa" : "#171717"}
              >
                {label}
              </text>
            </>
          ) : (
            <image
              href={product.logo}
              x={iconOnlyX}
              y={iconOnlyY}
              width={iconOnlySize}
              height={iconOnlySize}
              preserveAspectRatio="xMidYMid meet"
            />
          )}
        </g>
      )}
      <GarmentOverlay
        garment={product.garment}
        side={side}
        color={fill}
        dark={dark}
      />
    </svg>
  );
};

export default ProductVisual;
