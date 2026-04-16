// Reusable UI Components - Production-grade SaaS components
import { ReactNode } from 'react';
import { colors, borderRadius, shadows, transitions, spacing } from '../lib/design-system';

// Button Component
interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
  children: ReactNode;
}

export function Button({ 
  variant = 'primary', 
  size = 'md', 
  loading = false, 
  children, 
  className = '', 
  disabled,
  ...props 
}: ButtonProps) {
  const baseStyles = `
    inline-flex items-center justify-center
    font-medium transition-all duration-200
    focus:outline-none focus:ring-2 focus:ring-offset-2
    disabled:opacity-50 disabled:cursor-not-allowed
  `;

  const variants = {
    primary: `
      bg-blue-600 text-white hover:bg-blue-700
      focus:ring-blue-500 shadow-sm
    `,
    secondary: `
      bg-white text-neutral-700 border border-neutral-300
      hover:bg-neutral-50 focus:ring-blue-500
    `,
    ghost: `
      text-neutral-600 hover:text-neutral-900 hover:bg-neutral-100
      focus:ring-blue-500
    `,
  };

  const sizes = {
    sm: 'px-3 py-1.5 text-sm rounded-md',
    md: 'px-4 py-2 text-sm rounded-lg',
    lg: 'px-6 py-3 text-base rounded-lg',
  };

  return (
    <button
      className={`${baseStyles} ${variants[variant]} ${sizes[size]} ${className}`}
      disabled={disabled || loading}
      {...props}
    >
      {loading && (
        <svg className="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      )}
      {children}
    </button>
  );
}

// Card Component
interface CardProps {
  children: ReactNode;
  className?: string;
  hover?: boolean;
}

export function Card({ children, className = '', hover = false }: CardProps) {
  return (
    <div
      className={`
        bg-white rounded-xl border border-neutral-200 shadow-sm
        ${hover ? 'hover:shadow-md transition-shadow duration-200' : ''}
        ${className}
      `}
    >
      {children}
    </div>
  );
}

// Badge Component
interface BadgeProps {
  children: ReactNode;
  variant?: 'default' | 'success' | 'warning' | 'error' | 'info';
  size?: 'sm' | 'md';
}

export function Badge({ children, variant = 'default', size = 'md' }: BadgeProps) {
  const variants = {
    default: 'bg-neutral-100 text-neutral-700',
    success: 'bg-green-100 text-green-700',
    warning: 'bg-yellow-100 text-yellow-700',
    error: 'bg-red-100 text-red-700',
    info: 'bg-blue-100 text-blue-700',
  };

  const sizes = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-0.5 text-xs',
  };

  return (
    <span className={`inline-flex items-center rounded-full font-medium ${variants[variant]} ${sizes[size]}`}>
      {children}
    </span>
  );
}

// Metric Card Component
interface MetricCardProps {
  title: string;
  value: string;
  change?: {
    value: string;
    type: 'increase' | 'decrease' | 'neutral';
  };
  icon?: ReactNode;
  trend?: 'up' | 'down' | 'neutral';
}

export function MetricCard({ title, value, change, icon, trend }: MetricCardProps) {
  const trendColors = {
    up: 'text-green-600',
    down: 'text-red-600',
    neutral: 'text-neutral-500',
  };

  return (
    <Card hover className="p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-neutral-600 uppercase tracking-wide">{title}</h3>
        {icon && <div className="text-neutral-400">{icon}</div>}
      </div>
      <div className="flex items-baseline justify-between">
        <p className="text-3xl font-bold text-neutral-900">{value}</p>
        {trend && (
          <span className={`text-sm font-medium ${trendColors[trend]}`}>
            {trend === 'up' ? '↑' : trend === 'down' ? '↓' : '→'}
          </span>
        )}
      </div>
      {change && (
        <div className="mt-2">
          <span className={`text-sm font-medium ${
            change.type === 'increase' ? 'text-green-600' : 
            change.type === 'decrease' ? 'text-red-600' : 'text-neutral-500'
          }`}>
            {change.type === 'increase' ? '+' : change.type === 'decrease' ? '-' : ''}{change.value}
          </span>
        </div>
      )}
    </Card>
  );
}

// Status Indicator Component
interface StatusIndicatorProps {
  status: 'online' | 'offline' | 'warning' | 'error';
  label: string;
}

export function StatusIndicator({ status, label }: StatusIndicatorProps) {
  const statusColors = {
    online: 'bg-green-500',
    offline: 'bg-neutral-400',
    warning: 'bg-yellow-500',
    error: 'bg-red-500',
  };

  return (
    <div className="flex items-center space-x-2">
      <div className={`w-2 h-2 rounded-full ${statusColors[status]}`} />
      <span className="text-sm text-neutral-600">{label}</span>
    </div>
  );
}

// Loading Skeleton Component
export function Skeleton({ className = '' }: { className?: string }) {
  return (
    <div className={`animate-pulse bg-neutral-200 rounded ${className}`} />
  );
}

// Empty State Component
interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description: string;
  action?: ReactNode;
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="text-center py-12">
      {icon && <div className="text-neutral-400 mb-4">{icon}</div>}
      <h3 className="text-lg font-medium text-neutral-900 mb-2">{title}</h3>
      <p className="text-neutral-500 mb-6">{description}</p>
      {action}
    </div>
  );
}

// Search Input Component
interface SearchInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  onSearch?: (value: string) => void;
}

export function SearchInput({ onSearch, className = '', ...props }: SearchInputProps) {
  return (
    <div className="relative">
      <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
        <svg className="h-5 w-5 text-neutral-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
      </div>
      <input
        type="text"
        className={`
          block w-full pl-10 pr-3 py-2 border border-neutral-300 rounded-lg
          bg-white placeholder-neutral-500 text-neutral-900
          focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent
          ${className}
        `}
        onChange={(e) => onSearch?.(e.target.value)}
        {...props}
      />
    </div>
  );
}
