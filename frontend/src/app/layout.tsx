import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: 'DevCost AI - Cost Optimization Dashboard',
  description: 'AWS cost optimization and waste detection dashboard',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="bg-gray-50 min-h-screen">
        {children}
      </body>
    </html>
  );
}
