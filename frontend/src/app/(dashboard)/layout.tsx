'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

const navItems = [
  { href: '/', label: 'Dashboard', icon: '📊' },
  { href: '/waste', label: 'Waste', icon: '⚠️' },
  { href: '/recommendations', label: 'Recommendations', icon: '💡' },
  { href: '/actions', label: 'Actions', icon: '⚙️' },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="min-h-screen flex bg-neutral-50">
      {/* Sidebar */}
      <aside className="w-64 bg-white border-r border-neutral-200 flex flex-col">
        <div className="h-16 flex items-center px-6 border-b border-neutral-200">
          <Link href="/" className="flex items-center space-x-3">
            <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-blue-600 rounded-lg flex items-center justify-center">
              <span className="text-white font-bold text-sm">DC</span>
            </div>
            <span className="text-lg font-semibold text-neutral-900">DevCost AI</span>
          </Link>
        </div>
        
        <nav className="flex-1 p-4 space-y-1">
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={`
                flex items-center space-x-3 px-3 py-2.5 rounded-lg text-sm font-medium
                transition-all duration-200
                ${
                  pathname === item.href
                    ? 'bg-blue-50 text-blue-700 border-l-4 border-blue-700'
                    : 'text-neutral-600 hover:bg-neutral-50 hover:text-neutral-900'
                }
              `}
            >
              <span className="text-lg">{item.icon}</span>
              <span>{item.label}</span>
            </Link>
          ))}
        </nav>
        
        {/* Footer */}
        <div className="p-4 border-t border-neutral-200">
          <div className="flex items-center space-x-2 px-3 py-2">
            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
            <span className="text-xs text-neutral-500">Demo Environment</span>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <header className="h-16 bg-white border-b border-neutral-200 flex items-center px-8">
          <div className="flex items-center justify-between w-full">
            <div className="flex items-center space-x-4">
              <h1 className="text-xl font-semibold text-neutral-900">
                {navItems.find((item) => item.href === pathname)?.label || 'Dashboard'}
              </h1>
              <div className="h-6 w-px bg-neutral-300"></div>
              <span className="text-sm text-neutral-500">
                {new Date().toLocaleDateString('en-US', { 
                  weekday: 'short', 
                  month: 'short', 
                  day: 'numeric' 
                })}
              </span>
            </div>
            
            <div className="flex items-center space-x-3">
              <button className="p-2 text-neutral-500 hover:text-neutral-700 hover:bg-neutral-100 rounded-lg transition-colors">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                </svg>
              </button>
              <div className="w-8 h-8 bg-gradient-to-br from-purple-500 to-purple-600 rounded-full flex items-center justify-center">
                <span className="text-white font-medium text-sm">JD</span>
              </div>
            </div>
          </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 p-8 overflow-auto">
          {children}
        </main>
      </div>
    </div>
  );
}
