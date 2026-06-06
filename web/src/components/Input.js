import React, { forwardRef } from 'react';
import { classNames } from '@/lib/utils';

const Input = forwardRef(({ 
  label, 
  error, 
  icon: Icon,
  className,
  id,
  ...props 
}, ref) => {
  const inputId = id || props.name;
  
  return (
    <div className={className}>
      {label && (
        <label htmlFor={inputId} className="block text-sm font-medium text-brand-text-secondary mb-1.5">
          {label}
        </label>
      )}
      <div className="relative rounded-md shadow-sm">
        {Icon && (
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <Icon className="h-5 w-5 text-brand-text-muted" aria-hidden="true" />
          </div>
        )}
        <input
          ref={ref}
          id={inputId}
          className={classNames(
            "block w-full rounded-lg bg-brand-bg-secondary/50 border py-2.5 text-brand-text-primary backdrop-blur-sm transition-all focus:outline-none sm:text-sm",
            Icon ? "pl-10" : "pl-3",
            error 
              ? "border-brand-accent-danger text-brand-accent-danger focus:border-brand-accent-danger focus:ring-1 focus:ring-brand-accent-danger" 
              : "border-brand-border-glass focus:border-brand-accent-primary focus:ring-1 focus:ring-brand-accent-primary",
            props.disabled && "opacity-50 cursor-not-allowed bg-brand-bg-primary"
          )}
          {...props}
        />
      </div>
      {error && (
        <p className="mt-1.5 text-sm text-brand-accent-danger flex items-center gap-1">
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {error}
        </p>
      )}
    </div>
  );
});

Input.displayName = 'Input';

export default Input;
