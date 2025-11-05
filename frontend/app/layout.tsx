import React from "react";
import Link from "next/link";
import "./globals.css";

export const metadata = {
  title: "Premium Names",
  description: "Labels and tags UI",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="bg-gray-50 text-gray-900">
        <header className="border-b border-gray-200 bg-white">
          <div className="container py-3">
            <nav className="flex items-center gap-6">
              <Link href="/" className="font-semibold text-blue-700">Dashboard</Link>
              <Link href="/add" className="hover:text-blue-700">Add List</Link>
              <Link href="/labels" className="hover:text-blue-700">Labels</Link>
              <Link href="/tags" className="hover:text-blue-700">Tags</Link>
              {/* Batches route removed */}
            </nav>
          </div>
        </header>
        <main className="container py-6">{children}</main>
      </body>
    </html>
  );
}
