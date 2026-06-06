import React, { useEffect } from 'react';
import { classNames } from '@/lib/utils';

export default function SlideOver({ open, setOpen, title, children }) {
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

  return (
    <>
      {/* Background Overlay */}
      {open && (
        <div 
          className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm transition-opacity" 
          onClick={() => setOpen(false)}
        />
      )}

      {/* SlideOver Panel */}
      <div 
        className={classNames(
          "fixed inset-y-0 right-0 z-50 w-full max-w-md bg-brand-bg-secondary border-l border-brand-border-glass shadow-2xl transform transition-transform duration-300 ease-in-out sm:max-w-md",
          open ? "translate-x-0" : "translate-x-full"
        )}
      >
        <div className="flex h-full flex-col">
          {/* Header */}
          <div className="flex items-center justify-between px-4 py-6 sm:px-6 border-b border-brand-border-glass bg-brand-bg-primary">
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
          <div className="relative flex-1 px-4 py-6 sm:px-6 overflow-y-auto">
            {children}
          </div>
        </div>
      </div>
    </>
  );
}
