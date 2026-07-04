export type GarmentType = "shirt" | "sweater" | "hoodie";
export type Side = "front" | "back";

export const VIEW_W = 1000;
export const VIEW_H = 1080;

export const GARMENT_LABELS: Record<GarmentType, string> = {
  shirt: "Camisa",
  sweater: "Suéter",
  hoodie: "Sudadera",
};

export const SIDE_LABELS: Record<Side, string> = {
  front: "Frente",
  back: "Atrás",
};

export const PRINT_AREAS: Record<
  GarmentType,
  Record<Side, { x: number; y: number; w: number; h: number }>
> = {
  shirt: {
    front: { x: 320, y: 290, w: 360, h: 530 },
    back: { x: 320, y: 250, w: 360, h: 570 },
  },
  sweater: {
    front: { x: 315, y: 280, w: 370, h: 560 },
    back: { x: 315, y: 250, w: 370, h: 590 },
  },
  hoodie: {
    front: { x: 330, y: 300, w: 340, h: 400 },
    back: { x: 315, y: 360, w: 370, h: 480 },
  },
};

const SHIRT_FRONT =
  "M 405 148 Q 500 258 595 148 L 730 168 Q 760 180 785 202 L 940 330 L 885 478 L 728 434 Q 738 650 745 916 Q 500 942 255 916 Q 262 650 272 434 L 115 478 L 60 330 L 215 202 Q 240 180 270 168 Z";
const SHIRT_BACK =
  "M 405 148 Q 500 196 595 148 L 730 168 Q 760 180 785 202 L 940 330 L 885 478 L 728 434 Q 738 650 745 916 Q 500 942 255 916 Q 262 650 272 434 L 115 478 L 60 330 L 215 202 Q 240 180 270 168 Z";
const SHIRT_COLLAR_FRONT =
  "M 405 148 Q 500 258 595 148 L 581 139 Q 500 226 419 139 Z";
const SHIRT_COLLAR_BACK =
  "M 405 148 Q 500 196 595 148 L 585 139 Q 500 181 415 139 Z";

const SW_BODY_FRONT =
  "M 405 150 Q 500 235 595 150 L 725 170 Q 748 195 752 260 L 758 470 L 762 940 Q 500 962 238 940 L 242 470 L 248 260 Q 252 195 275 170 Z";
const SW_BODY_BACK =
  "M 405 150 Q 500 196 595 150 L 725 170 Q 748 195 752 260 L 758 470 L 762 940 Q 500 962 238 940 L 242 470 L 248 260 Q 252 195 275 170 Z";
const SW_HEM = "M 238 940 L 762 940 L 762 1008 Q 500 1028 238 1008 Z";
const SW_SLEEVE_L =
  "M 275 170 L 200 208 Q 155 250 143 330 L 112 830 L 116 892 L 232 898 L 244 470 L 252 300 Q 258 225 275 170 Z";
const SW_SLEEVE_R =
  "M 725 170 L 800 208 Q 845 250 857 330 L 888 830 L 884 892 L 768 898 L 756 470 L 748 300 Q 742 225 725 170 Z";
const SW_CUFF_L = "M 114 892 L 232 898 L 228 972 L 106 964 Z";
const SW_CUFF_R = "M 886 892 L 768 898 L 772 972 L 894 964 Z";
const SW_COLLAR_FRONT =
  "M 405 150 Q 500 235 595 150 L 580 140 Q 500 208 420 140 Z";
const SW_COLLAR_BACK =
  "M 405 150 Q 500 196 595 150 L 587 140 Q 500 180 413 140 Z";

const HOODIE_POCKET = "M 350 720 L 650 720 L 690 940 L 310 940 Z";
const HOOD_RING =
  "M 370 180 Q 392 84 500 76 Q 608 84 630 180 Q 638 214 614 232 L 588 202 Q 580 120 500 116 Q 420 120 412 202 L 386 232 Q 362 214 370 180 Z";
const HOOD_CAVITY =
  "M 412 202 Q 420 120 500 116 Q 580 120 588 202 Q 500 262 412 202 Z";
const HOOD_BACK =
  "M 375 175 Q 398 66 500 58 Q 602 66 625 175 Q 630 252 500 328 Q 370 252 375 175 Z";
const HOOD_STRINGS = "M 460 234 Q 450 300 446 362 M 540 234 Q 550 300 554 362";

const HEM_RIBS = Array.from(
  { length: 17 },
  (_, i) => `M ${260 + i * 30} 948 L ${260 + i * 30} 1000`,
).join(" ");

export function isDarkColor(hex: string) {
  const n = parseInt(hex.slice(1), 16);
  const r = (n >> 16) & 255;
  const g = (n >> 8) & 255;
  const b = n & 255;
  return 0.299 * r + 0.587 * g + 0.114 * b < 140;
}

export function clipPathsFor(garment: GarmentType, side: Side): string[] {
  if (garment === "shirt") return [side === "front" ? SHIRT_FRONT : SHIRT_BACK];
  const body = side === "front" ? SW_BODY_FRONT : SW_BODY_BACK;
  return [body, SW_HEM, SW_SLEEVE_L, SW_SLEEVE_R, SW_CUFF_L, SW_CUFF_R];
}

type GarmentProps = {
  garment: GarmentType;
  side: Side;
  color: string;
  dark: boolean;
};

export function GarmentBase({ garment, side, color, dark }: GarmentProps) {
  const outline = dark ? "rgba(255,255,255,0.30)" : "rgba(0,0,0,0.32)";
  const seam = dark ? "rgba(255,255,255,0.18)" : "rgba(0,0,0,0.18)";
  const fillProps = { fill: color, stroke: outline, strokeWidth: 4 };

  if (garment === "shirt") {
    return (
      <g>
        <path d={side === "front" ? SHIRT_FRONT : SHIRT_BACK} {...fillProps} />
        <path
          d="M 745 416 L 868 450 M 255 416 L 132 450"
          stroke={seam}
          strokeWidth={3}
          fill="none"
        />
        <path
          d="M 265 890 Q 500 914 735 890"
          stroke={seam}
          strokeWidth={3}
          fill="none"
        />
      </g>
    );
  }

  return (
    <g>
      <path d={side === "front" ? SW_BODY_FRONT : SW_BODY_BACK} {...fillProps} />
      <path d={SW_HEM} {...fillProps} />
      <path d={HEM_RIBS} stroke={seam} strokeWidth={2} fill="none" />
      <path d={SW_SLEEVE_L} {...fillProps} />
      <path d={SW_SLEEVE_R} {...fillProps} />
      <path d={SW_CUFF_L} {...fillProps} />
      <path d={SW_CUFF_R} {...fillProps} />
      {garment === "hoodie" && side === "front" && (
        <path
          d={HOODIE_POCKET}
          fill={dark ? "rgba(255,255,255,0.06)" : "rgba(0,0,0,0.08)"}
          stroke={seam}
          strokeWidth={3}
        />
      )}
    </g>
  );
}

export function GarmentOverlay({ garment, side, color, dark }: GarmentProps) {
  const outline = dark ? "rgba(255,255,255,0.30)" : "rgba(0,0,0,0.32)";
  const rib = dark ? "rgba(255,255,255,0.14)" : "rgba(0,0,0,0.16)";
  const strings = dark ? "rgba(255,255,255,0.70)" : "rgba(0,0,0,0.38)";

  if (garment === "shirt") {
    return (
      <path
        d={side === "front" ? SHIRT_COLLAR_FRONT : SHIRT_COLLAR_BACK}
        fill={rib}
        stroke={outline}
        strokeWidth={3}
      />
    );
  }

  if (garment === "sweater") {
    return (
      <path
        d={side === "front" ? SW_COLLAR_FRONT : SW_COLLAR_BACK}
        fill={rib}
        stroke={outline}
        strokeWidth={3}
      />
    );
  }

  if (side === "front") {
    return (
      <g>
        <path d={HOOD_RING} fill={color} stroke={outline} strokeWidth={4} />
        <path d={HOOD_CAVITY} fill="rgba(0,0,0,0.42)" />
        <path
          d={HOOD_STRINGS}
          stroke={strings}
          strokeWidth={7}
          fill="none"
          strokeLinecap="round"
        />
        <circle cx={446} cy={366} r={7} fill={strings} />
        <circle cx={554} cy={366} r={7} fill={strings} />
      </g>
    );
  }

  return (
    <g>
      <path d={HOOD_BACK} fill={color} stroke={outline} strokeWidth={4} />
      <path
        d="M 500 58 L 500 328"
        stroke={dark ? "rgba(255,255,255,0.18)" : "rgba(0,0,0,0.18)"}
        strokeWidth={3}
      />
    </g>
  );
}
