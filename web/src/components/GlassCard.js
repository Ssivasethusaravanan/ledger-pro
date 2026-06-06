import React from 'react';
import { classNames } from '@/lib/utils';

export default function GlassCard({ children, className, noPadding = false, ...props }) {
  return (
    <div 
      className={classNames(
        "glass-card",
        !noPadding && "p-6",
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}
