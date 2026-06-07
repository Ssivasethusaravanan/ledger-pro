'use client';

import { useState } from 'react';
import { usePathname } from 'next/navigation';
import { AuthProvider, useAuth } from '@/lib/auth';
import Sidebar from '@/components/Sidebar';
import TopBar from '@/components/TopBar';

function LayoutContent({ children }) {
  const { isAuthenticated, isLoading } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const pathname = usePathname();

  // If loading, show a full screen shimmer
  if (isLoading) {
    return (
      <div className="flex h-screen w-full items-center justify-center bg-brand-bg-primary">
        <div className="w-16 h-16 border-4 border-brand-accent-primary border-t-transparent rounded-full animate-spin"></div>
      </div>
    );
  }

  // If on login or signup page or not authenticated, don't show the dashboard shell
  if (pathname === '/login' || pathname === '/signup' || !isAuthenticated) {
    return <main className="h-full">{children}</main>;
  }

  // Dashboard Shell
  return (
    <div className="h-full flex">
      <Sidebar mobileMenuOpen={mobileMenuOpen} setMobileMenuOpen={setMobileMenuOpen} />
      
      <div className="flex flex-1 flex-col overflow-hidden">
        <TopBar setMobileMenuOpen={setMobileMenuOpen} />
        
        <main className="flex-1 overflow-y-auto bg-brand-bg-primary/50 p-4 sm:p-6 lg:p-8">
          <div className="mx-auto max-w-7xl animate-fade-in-up">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}

export default function ClientWrapper({ children }) {
  return (
    <AuthProvider>
      <LayoutContent>{children}</LayoutContent>
    </AuthProvider>
  );
}
