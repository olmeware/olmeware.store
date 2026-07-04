import type { Metadata } from "next";
import MockupEditor from "./mockup-editor";

export const metadata: Metadata = {
  title: "Mockup editor — OLMEWARE Admin",
  description: "Compose garment mockups with tech logos",
};

const AdminCreatePage = () => <MockupEditor />;

export default AdminCreatePage;
