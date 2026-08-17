import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "GoHighLevel Round 1 Scaffold",
  description: "Interview scaffold: Go + Gin + MongoDB backend, Next.js frontend.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
