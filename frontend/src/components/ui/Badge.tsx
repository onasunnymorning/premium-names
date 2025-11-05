import { ReactNode } from 'react';
import { clsx } from 'clsx';

interface BadgeProps {
  children: ReactNode;
  variant?: 'default' | 'secondary' | 'destructive' | 'outline';
  className?: string;
  onClick?: () => void;
}

export function Badge({ 
  children, 
  variant = 'default', 
  className,
  onClick 
}: BadgeProps) {
  const baseClasses = 'inline-flex items-center rounded-md px-2 py-1 text-xs font-medium';
  
  const variantClasses = {
    default: 'bg-blue-50 text-blue-700 ring-1 ring-inset ring-blue-700/10',
    secondary: 'bg-gray-50 text-gray-600 ring-1 ring-inset ring-gray-500/10',
    destructive: 'bg-red-50 text-red-700 ring-1 ring-inset ring-red-600/10',
    outline: 'text-gray-900 ring-1 ring-inset ring-gray-200',
  };

  return (
    <span 
      className={clsx(
        baseClasses, 
        variantClasses[variant], 
        onClick && 'cursor-pointer hover:opacity-80',
        className
      )}
      onClick={onClick}
    >
      {children}
    </span>
  );
}
