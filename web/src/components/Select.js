import React, { forwardRef } from 'react';
import { classNames } from '@/lib/utils';

const Select = forwardRef(({ 
  label, 
  error, 
  options = [],
  className,
  id,
  ...props 
}, ref) => {
  const selectId = id || props.name;
  
  return (
    <div className={className}>
      {label && (
        <label htmlFor={selectId} className="block text-sm font-medium text-brand-text-secondary mb-1.5">
          {label}
        </label>
      )}
      <div className="relative">
        <select
          ref={ref}
          id={selectId}
          className={classNames(
            "block w-full appearance-none rounded-lg bg-brand-bg-secondary/50 border py-2.5 pl-3 pr-10 text-brand-text-primary backdrop-blur-sm transition-all focus:outline-none sm:text-sm",
            error 
              ? "border-brand-accent-danger text-brand-accent-danger focus:border-brand-accent-danger focus:ring-1 focus:ring-brand-accent-danger" 
              : "border-brand-border-glass focus:border-brand-accent-primary focus:ring-1 focus:ring-brand-accent-primary",
            props.disabled && "opacity-50 cursor-not-allowed bg-brand-bg-primary"
          )}
          {...props}
        >
          {options.map((opt) => (
            <option key={opt.value} value={opt.value} className="bg-brand-bg-primary text-white">
              {opt.label}
            </option>
          ))}
        </select>
        <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-brand-text-muted">
          <svg className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clipRule="evenodd" />
          </svg>
        </div>
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

Select.displayName = 'Select';

export default Select;
