import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "InstaBox Sandbox",
  description: "Linux sandbox machines in seconds — clone build",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
