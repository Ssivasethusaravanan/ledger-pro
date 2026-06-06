import React from 'react';
import { classNames } from '@/lib/utils';

export default function Badge({ children, variant = 'default', size = 'md', className }) {
  const variants = {
    default: "bg-brand-bg-glass text-brand-text-secondary border-brand-border-glass",
    success: "bg-brand-accent-success/10 text-brand-accent-success border-brand-accent-success/20",
    warning: "bg-brand-accent-warning/10 text-brand-accent-warning border-brand-accent-warning/20",
    danger: "bg-brand-accent-danger/10 text-brand-accent-danger border-brand-accent-danger/20",
    info: "bg-brand-accent-info/10 text-brand-accent-info border-brand-accent-info/20",
    primary: "bg-brand-accent-primary/10 text-brand-accent-primary border-brand-accent-primary/20",
  };

  const sizes = {
    sm: "px-2 py-0.5 text-xs",
    md: "px-2.5 py-0.5 text-sm",
    lg: "px-3 py-1 text-sm font-medium",
  };

  return (
    <span className={classNames(
      "inline-flex items-center rounded-full border whitespace-nowrap font-medium",
      variants[variant],
      sizes[size],
      className
    )}>
      {children}
    </span>
  );
}
