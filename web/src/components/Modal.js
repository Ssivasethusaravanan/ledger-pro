import React, { useEffect } from 'react';
import { classNames } from '@/lib/utils';
import GlassCard from './GlassCard';

export default function Modal({ open, setOpen, title, children, size = 'md' }) {
  // Close on escape key
  useEffect(() => {
    const handleEscape = (e) => {
      if (e.key === 'Escape') {
        setOpen(false);
      }
    };
    
    if (open) {
      window.addEventListener('keydown', handleEscape);
    }
    
    return () => window.removeEventListener('keydown', handleEscape);
  }, [open, setOpen]);

  const sizes = {
    sm: "max-w-md",
    md: "max-w-lg",
    lg: "max-w-2xl",
    xl: "max-w-4xl"
  };

  return (
    <>
      {/* Background Overlay */}
      {open && (
        <div 
          className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm transition-opacity flex items-center justify-center p-4 sm:p-6" 
          onClick={() => setOpen(false)}
        >
          {/* Modal Panel */}
          <div 
            className={classNames(
              "w-full transform transition-all",
              sizes[size],
              open ? "scale-100 opacity-100" : "scale-95 opacity-0"
            )}
            onClick={(e) => e.stopPropagation()}
          >
            <GlassCard noPadding className="overflow-hidden flex flex-col max-h-[90vh]">
              {/* Header */}
              <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-glass bg-brand-bg-primary/50">
                <h2 className="text-xl font-semibold text-white">{title}</h2>
                <button
                  type="button"
                  className="rounded-md text-brand-text-secondary hover:text-white focus:outline-none transition-colors"
                  onClick={() => setOpen(false)}
                >
                  <span className="sr-only">Close panel</span>
                  <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              
              {/* Content */}
              <div className="flex-1 px-6 py-6 overflow-y-auto hide-scrollbar">
                {children}
              </div>
            </GlassCard>
          </div>
        </div>
      )}
    </>
  );
}
