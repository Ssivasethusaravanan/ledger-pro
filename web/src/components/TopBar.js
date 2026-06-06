'use client';

import React from 'react';
import { useAuth } from '@/lib/auth';
import { IconLogout, IconMenu } from './Icons';
import { usePathname } from 'next/navigation';

export default function TopBar({ setMobileMenuOpen }) {
  const { role, logout } = useAuth();
  const pathname = usePathname();
  
  // Format pathname into title
  const getPageTitle = () => {
    if (pathname === '/') return 'Dashboard';
    if (pathname.startsWith('/accounts')) return 'Accounts';
    if (pathname.startsWith('/transactions')) return 'Transactions';
    if (pathname.startsWith('/health')) return 'System Health';
    return 'LedgerPro';
  };

  const getRoleColor = (r) => {
    switch (r) {
      case 'admin': return 'bg-brand-accent-danger/20 text-brand-accent-danger border border-brand-accent-danger/30';
      case 'write': return 'bg-brand-accent-warning/20 text-brand-accent-warning border border-brand-accent-warning/30';
      case 'read':  return 'bg-brand-accent-success/20 text-brand-accent-success border border-brand-accent-success/30';
      default:      return 'bg-brand-bg-glass text-brand-text-secondary';
    }
  };

  return (
    <div className="sticky top-0 z-30 flex h-16 shrink-0 items-center gap-x-4 border-b border-brand-border-glass bg-brand-bg-primary/80 backdrop-blur-md px-4 sm:gap-x-6 sm:px-6 lg:px-8">
      <button
        type="button"
        className="-m-2.5 p-2.5 text-brand-text-secondary lg:hidden hover:text-brand-text-primary transition-colors"
        onClick={() => setMobileMenuOpen(true)}
      >
        <span className="sr-only">Open sidebar</span>
        <IconMenu className="h-6 w-6" />
      </button>

      {/* Separator */}
      <div className="h-6 w-px bg-brand-border-glass lg:hidden" aria-hidden="true" />

      <div className="flex flex-1 gap-x-4 self-stretch lg:gap-x-6">
        <div className="flex flex-1 items-center">
          <h1 className="text-xl font-semibold text-brand-text-primary">{getPageTitle()}</h1>
        </div>
        <div className="flex items-center gap-x-4 lg:gap-x-6">
          
          {role && (
            <div className={`px-2.5 py-1 rounded-full text-xs font-medium uppercase tracking-wider ${getRoleColor(role)}`}>
              {role} ROLE
            </div>
          )}

          {/* Separator */}
          <div className="hidden lg:block lg:h-6 lg:w-px lg:bg-brand-border-glass" aria-hidden="true" />

          <button
            onClick={logout}
            className="flex items-center gap-2 text-sm font-semibold leading-6 text-brand-text-secondary hover:text-brand-accent-danger transition-colors"
          >
            <span className="hidden sm:inline">Logout</span>
            <IconLogout className="h-5 w-5" />
          </button>
        </div>
      </div>
    </div>
  );
}
