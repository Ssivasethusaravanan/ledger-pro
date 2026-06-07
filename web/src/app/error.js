'use client';

import { useEffect } from 'react';
import Button from '@/components/Button';
import GlassCard from '@/components/GlassCard';

export default function Error({ error, reset }) {
  useEffect(() => {
    // Log the error to an error reporting service if needed
    console.error('App Error:', error);
  }, [error]);

  return (
    <div className="flex h-full min-h-[60vh] w-full flex-col items-center justify-center p-4">
      <GlassCard className="max-w-md w-full text-center space-y-6 py-12">
        <div className="mx-auto w-16 h-16 rounded-full bg-brand-accent-danger/20 flex items-center justify-center">
          <svg className="w-8 h-8 text-brand-accent-danger" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        
        <div>
          <h2 className="text-xl font-bold text-white tracking-tight">Something went wrong</h2>
          <p className="mt-2 text-sm text-brand-text-secondary px-4">
            {error.message || "An unexpected error occurred. Please try again."}
          </p>
        </div>
        
        <div className="pt-4 flex justify-center">
          <Button onClick={() => reset()} variant="primary">
            Try again
          </Button>
        </div>
      </GlassCard>
    </div>
  );
}
