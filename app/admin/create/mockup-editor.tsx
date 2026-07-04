"use client";

import { useEffect, useRef, useState } from "react";
import {
  GARMENT_LABELS,
  GarmentBase,
  GarmentOverlay,
  type GarmentType,
  PRINT_AREAS,
  SIDE_LABELS,
  type Side,
  VIEW_H,
  VIEW_W,
  clipPathsFor,
  isDarkColor,
} from "./garments";

type Design = {
  id: string;
  src: string;
  name: string;
  side: Side;
  cx: number;
  cy: number;
  w: number;
  aspect: number;
};

type DragState = {
  id: string;
  mode: "move" | "resize";
  dx: number;
  dy: number;
};

const COLORS = [
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

const clamp = (v: number, min: number, max: number) =>
  Math.min(max, Math.max(min, v));

export default function MockupEditor() {
  const [garment, setGarment] = useState<GarmentType>("shirt");
  const [color, setColor] = useState("#1a1a1a");
  const [side, setSide] = useState<Side>("front");
  const [designs, setDesigns] = useState<Design[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [showGuide, setShowGuide] = useState(true);
  const [dragOver, setDragOver] = useState(false);

  const svgRef = useRef<SVGSVGElement>(null);
  const dragRef = useRef<DragState | null>(null);

  const dark = isDarkColor(color);
  const sideDesigns = designs.filter((d) => d.side === side);
  const selected = designs.find((d) => d.id === selectedId) ?? null;
  const printArea = PRINT_AREAS[garment][side];

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Delete" && e.key !== "Backspace") return;
      const tag = (e.target as HTMLElement).tagName;
      if (tag === "INPUT" || tag === "TEXTAREA") return;
      setDesigns((ds) => ds.filter((d) => d.id !== selectedId));
      setSelectedId(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [selectedId]);

  const svgPoint = (e: { clientX: number; clientY: number }) => {
    const rect = svgRef.current!.getBoundingClientRect();
    return {
      x: ((e.clientX - rect.left) / rect.width) * VIEW_W,
      y: ((e.clientY - rect.top) / rect.height) * VIEW_H,
    };
  };

  const addFiles = (files: FileList | File[], at?: { x: number; y: number }) => {
    Array.from(files)
      .filter((f) => f.type.startsWith("image/"))
      .forEach((file) => {
        const reader = new FileReader();
        reader.onload = () => {
          const src = reader.result as string;
          const img = new Image();
          img.onload = () => {
            const aspect = img.naturalHeight / img.naturalWidth || 1;
            const area = PRINT_AREAS[garment][side];
            const id = crypto.randomUUID();
            setDesigns((ds) => [
              ...ds,
              {
                id,
                src,
                name: file.name,
                side,
                cx: at?.x ?? area.x + area.w / 2,
                cy: at?.y ?? area.y + area.h / 2,
                w: 240,
                aspect,
              },
            ]);
            setSelectedId(id);
          };
          img.src = src;
        };
        reader.readAsDataURL(file);
      });
  };

  const startDrag = (
    e: React.PointerEvent,
    design: Design,
    mode: DragState["mode"],
  ) => {
    e.preventDefault();
    e.stopPropagation();
    setSelectedId(design.id);
    const p = svgPoint(e);
    dragRef.current = {
      id: design.id,
      mode,
      dx: design.cx - p.x,
      dy: design.cy - p.y,
    };
    (e.currentTarget as Element).setPointerCapture(e.pointerId);
  };

  const onPointerMove = (e: React.PointerEvent) => {
    const drag = dragRef.current;
    if (!drag) return;
    const p = svgPoint(e);
    setDesigns((ds) =>
      ds.map((d) => {
        if (d.id !== drag.id) return d;
        if (drag.mode === "move") {
          return {
            ...d,
            cx: clamp(p.x + drag.dx, 20, VIEW_W - 20),
            cy: clamp(p.y + drag.dy, 20, VIEW_H - 20),
          };
        }
        const w = clamp(
          2 * Math.max(p.x - d.cx, (p.y - d.cy) / d.aspect),
          40,
          950,
        );
        return { ...d, w };
      }),
    );
  };

  const updateSelected = (patch: Partial<Design>) => {
    if (!selectedId) return;
    setDesigns((ds) =>
      ds.map((d) => (d.id === selectedId ? { ...d, ...patch } : d)),
    );
  };

  const moveLayer = (id: string, dir: -1 | 1) => {
    setDesigns((ds) => {
      const i = ds.findIndex((d) => d.id === id);
      const j = i + dir;
      if (i < 0 || j < 0 || j >= ds.length) return ds;
      const next = [...ds];
      [next[i], next[j]] = [next[j], next[i]];
      return next;
    });
  };

  const downloadPng = async () => {
    const svg = svgRef.current;
    if (!svg) return;
    const clone = svg.cloneNode(true) as SVGSVGElement;
    clone.querySelectorAll("[data-export-ignore]").forEach((n) => n.remove());
    clone.setAttribute("width", String(VIEW_W * 2));
    clone.setAttribute("height", String(VIEW_H * 2));
    const xml = new XMLSerializer().serializeToString(clone);
    const url = URL.createObjectURL(
      new Blob([xml], { type: "image/svg+xml" }),
    );
    const img = new Image();
    await new Promise((resolve, reject) => {
      img.onload = resolve;
      img.onerror = reject;
      img.src = url;
    });
    const canvas = document.createElement("canvas");
    canvas.width = VIEW_W * 2;
    canvas.height = VIEW_H * 2;
    canvas.getContext("2d")!.drawImage(img, 0, 0, canvas.width, canvas.height);
    URL.revokeObjectURL(url);
    canvas.toBlob((blob) => {
      if (!blob) return;
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = `olmeware-${garment}-${side}-${color.replace("#", "")}.png`;
      a.click();
      URL.revokeObjectURL(a.href);
    }, "image/png");
  };

  return (
    <div className="min-h-screen bg-neutral-100 text-neutral-900">
      <header className="flex items-center justify-between border-b border-neutral-200 bg-white px-6 py-4">
        <div>
          <h1 className="text-lg font-bold tracking-tight">
            OLMEWARE · Editor de mockups
          </h1>
          <p className="text-sm text-neutral-500">
            Arrastra un logo sobre la prenda, acomódalo y descarga el PNG.
          </p>
        </div>
        <button
          onClick={downloadPng}
          className="rounded-lg bg-neutral-900 px-5 py-2.5 text-sm font-semibold text-white hover:bg-neutral-700"
        >
          Descargar PNG
        </button>
      </header>

      <div className="mx-auto flex max-w-6xl flex-col gap-6 p-6 lg:flex-row">
        <aside className="flex w-full shrink-0 flex-col gap-6 lg:w-72">
          <section className="rounded-xl bg-white p-4 shadow-sm">
            <h2 className="mb-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">
              Prenda
            </h2>
            <div className="grid grid-cols-3 gap-2">
              {(Object.keys(GARMENT_LABELS) as GarmentType[]).map((g) => (
                <button
                  key={g}
                  onClick={() => setGarment(g)}
                  className={`rounded-lg border px-2 py-2 text-sm ${
                    garment === g
                      ? "border-neutral-900 bg-neutral-900 text-white"
                      : "border-neutral-200 hover:border-neutral-400"
                  }`}
                >
                  {GARMENT_LABELS[g]}
                </button>
              ))}
            </div>
            <div className="mt-3 grid grid-cols-2 gap-2">
              {(Object.keys(SIDE_LABELS) as Side[]).map((s) => (
                <button
                  key={s}
                  onClick={() => {
                    setSide(s);
                    setSelectedId(null);
                  }}
                  className={`rounded-lg border px-2 py-2 text-sm ${
                    side === s
                      ? "border-neutral-900 bg-neutral-900 text-white"
                      : "border-neutral-200 hover:border-neutral-400"
                  }`}
                >
                  {SIDE_LABELS[s]}
                </button>
              ))}
            </div>
          </section>

          <section className="rounded-xl bg-white p-4 shadow-sm">
            <h2 className="mb-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">
              Color
            </h2>
            <div className="grid grid-cols-6 gap-2">
              {COLORS.map((c) => (
                <button
                  key={c}
                  onClick={() => setColor(c)}
                  aria-label={`Color ${c}`}
                  className={`h-9 rounded-lg border-2 ${
                    color === c ? "border-sky-500" : "border-neutral-200"
                  }`}
                  style={{ backgroundColor: c }}
                />
              ))}
            </div>
            <label className="mt-3 flex items-center gap-2 text-sm text-neutral-600">
              <input
                type="color"
                value={color}
                onChange={(e) => setColor(e.target.value)}
                className="h-8 w-12 cursor-pointer rounded border border-neutral-200"
              />
              Color personalizado
            </label>
          </section>

          <section className="rounded-xl bg-white p-4 shadow-sm">
            <h2 className="mb-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">
              Diseños ({SIDE_LABELS[side].toLowerCase()})
            </h2>
            <label className="block cursor-pointer rounded-lg border-2 border-dashed border-neutral-300 px-3 py-4 text-center text-sm text-neutral-500 hover:border-neutral-400">
              Subir imágenes
              <input
                type="file"
                accept="image/*"
                multiple
                className="hidden"
                onChange={(e) => {
                  if (e.target.files) addFiles(e.target.files);
                  e.target.value = "";
                }}
              />
            </label>
            <ul className="mt-3 flex flex-col gap-2">
              {sideDesigns.map((d) => (
                <li
                  key={d.id}
                  onClick={() => setSelectedId(d.id)}
                  className={`flex cursor-pointer items-center gap-2 rounded-lg border p-2 ${
                    d.id === selectedId
                      ? "border-sky-500 bg-sky-50"
                      : "border-neutral-200"
                  }`}
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={d.src}
                    alt={d.name}
                    className="h-9 w-9 rounded bg-neutral-100 object-contain"
                  />
                  <span className="min-w-0 flex-1 truncate text-xs text-neutral-600">
                    {d.name}
                  </span>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      moveLayer(d.id, 1);
                    }}
                    title="Traer al frente"
                    className="rounded px-1 text-neutral-400 hover:text-neutral-900"
                  >
                    ↑
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      moveLayer(d.id, -1);
                    }}
                    title="Enviar atrás"
                    className="rounded px-1 text-neutral-400 hover:text-neutral-900"
                  >
                    ↓
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      setDesigns((ds) => ds.filter((x) => x.id !== d.id));
                      if (selectedId === d.id) setSelectedId(null);
                    }}
                    title="Eliminar"
                    className="rounded px-1 text-neutral-400 hover:text-red-600"
                  >
                    ✕
                  </button>
                </li>
              ))}
              {sideDesigns.length === 0 && (
                <li className="text-xs text-neutral-400">
                  Sin diseños en este lado.
                </li>
              )}
            </ul>
          </section>

          {selected && (
            <section className="rounded-xl bg-white p-4 shadow-sm">
              <h2 className="mb-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">
                Diseño seleccionado
              </h2>
              <label className="block text-sm text-neutral-600">
                Tamaño
                <input
                  type="range"
                  min={40}
                  max={950}
                  value={selected.w}
                  onChange={(e) => updateSelected({ w: Number(e.target.value) })}
                  className="mt-1 w-full"
                />
              </label>
              <button
                onClick={() =>
                  updateSelected({
                    side: selected.side === "front" ? "back" : "front",
                  })
                }
                className="mt-3 w-full rounded-lg border border-neutral-200 px-3 py-2 text-sm hover:border-neutral-400"
              >
                Mover a {selected.side === "front" ? "atrás" : "frente"}
              </button>
            </section>
          )}

          <label className="flex items-center gap-2 px-1 text-sm text-neutral-600">
            <input
              type="checkbox"
              checked={showGuide}
              onChange={(e) => setShowGuide(e.target.checked)}
            />
            Mostrar área de impresión
          </label>
        </aside>

        <main
          className={`relative flex-1 rounded-xl bg-white p-4 shadow-sm ${
            dragOver ? "ring-4 ring-sky-300" : ""
          }`}
          onDragOver={(e) => {
            e.preventDefault();
            setDragOver(true);
          }}
          onDragLeave={() => setDragOver(false)}
          onDrop={(e) => {
            e.preventDefault();
            setDragOver(false);
            if (e.dataTransfer.files.length) {
              addFiles(e.dataTransfer.files, svgPoint(e));
            }
          }}
        >
          <svg
            ref={svgRef}
            viewBox={`0 0 ${VIEW_W} ${VIEW_H}`}
            className="mx-auto block max-h-[82vh] w-full touch-none select-none"
            xmlns="http://www.w3.org/2000/svg"
            onPointerDown={() => setSelectedId(null)}
            onPointerMove={onPointerMove}
            onPointerUp={() => {
              dragRef.current = null;
            }}
          >
            <defs>
              <clipPath id="garment-clip">
                {clipPathsFor(garment, side).map((d, i) => (
                  <path key={i} d={d} />
                ))}
              </clipPath>
            </defs>

            <GarmentBase garment={garment} side={side} color={color} dark={dark} />

            <g clipPath="url(#garment-clip)">
              {sideDesigns.map((d) => (
                <image
                  key={d.id}
                  href={d.src}
                  x={d.cx - d.w / 2}
                  y={d.cy - (d.w * d.aspect) / 2}
                  width={d.w}
                  height={d.w * d.aspect}
                  preserveAspectRatio="xMidYMid meet"
                  className="cursor-move"
                  onPointerDown={(e) => startDrag(e, d, "move")}
                />
              ))}
            </g>

            <GarmentOverlay garment={garment} side={side} color={color} dark={dark} />

            {showGuide && (
              <rect
                data-export-ignore
                x={printArea.x}
                y={printArea.y}
                width={printArea.w}
                height={printArea.h}
                fill="none"
                stroke="rgba(14,165,233,0.5)"
                strokeWidth={2}
                strokeDasharray="10 8"
                pointerEvents="none"
              />
            )}

            {selected && selected.side === side && (
              <g data-export-ignore>
                <rect
                  x={selected.cx - selected.w / 2}
                  y={selected.cy - (selected.w * selected.aspect) / 2}
                  width={selected.w}
                  height={selected.w * selected.aspect}
                  fill="none"
                  stroke="#0ea5e9"
                  strokeWidth={2.5}
                  strokeDasharray="8 6"
                  pointerEvents="none"
                />
                <circle
                  cx={selected.cx + selected.w / 2}
                  cy={selected.cy + (selected.w * selected.aspect) / 2}
                  r={14}
                  fill="#0ea5e9"
                  className="cursor-nwse-resize"
                  onPointerDown={(e) => startDrag(e, selected, "resize")}
                />
              </g>
            )}
          </svg>

          {sideDesigns.length === 0 && (
            <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
              <p className="rounded-lg bg-neutral-900/70 px-4 py-2 text-sm text-white">
                Arrastra aquí la imagen de un lenguaje (PNG/SVG) o usa «Subir
                imágenes»
              </p>
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
