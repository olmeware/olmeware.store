import type { Metadata } from "next";
import MockupEditor from "./mockup-editor";

export const metadata: Metadata = {
  title: "Editor de mockups — OLMEWARE",
  description: "Crea mockups de prendas con logos de tecnologías",
};

export default function AdminCreatePage() {
  return <MockupEditor />;
}
