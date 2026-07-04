import type { Metadata } from "next";
import { Roboto } from "next/font/google";
import "./globals.css";

const roboto = Roboto({
  variable: "--font-roboto",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "OLMEWARE STORE",
  description: "Tech Clothing & More",
};

const RootLayout = ({ children }: Readonly<{ children: React.ReactNode }>) => (
  <html lang="en" className={`${roboto.variable} h-full antialiased`}>
    <body className="min-h-full flex flex-col">{children}</body>
  </html>
);

export default RootLayout;
