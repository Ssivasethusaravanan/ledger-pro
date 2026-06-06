'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { classNames } from '@/lib/utils';
import { IconDashboard, IconAccounts, IconTransactions, IconHealth } from './Icons';

const navigation = [
  { name: 'Dashboard', href: '/', icon: IconDashboard },
  { name: 'Accounts', href: '/accounts', icon: IconAccounts },
  { name: 'Transactions', href: '/transactions', icon: IconTransactions },
  { name: 'System Health', href: '/health', icon: IconHealth },
];

export default function Sidebar({ mobileMenuOpen, setMobileMenuOpen }) {
  const pathname = usePathname();

  return (
    <>
      {/* Mobile background overlay */}
      {mobileMenuOpen && (
        <div 
          className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm lg:hidden transition-opacity"
          onClick={() => setMobileMenuOpen(false)}
        />
      )}

      {/* Sidebar component */}
      <div className={classNames(
        "fixed inset-y-0 left-0 z-50 w-64 glass-panel transform transition-transform duration-300 ease-in-out lg:translate-x-0 lg:static lg:inset-auto",
        mobileMenuOpen ? "translate-x-0" : "-translate-x-full"
      )}>
        <div className="flex h-16 shrink-0 items-center px-6 border-b border-brand-border-glass">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-brand-accent-primary to-brand-accent-info flex items-center justify-center text-white font-bold text-lg shadow-[0_0_15px_rgba(99,102,241,0.5)]">
              LP
            </div>
            <span className="text-xl font-bold text-gradient">LedgerPro</span>
          </div>
        </div>
        
        <nav className="flex flex-1 flex-col px-4 py-6 overflow-y-auto">
          <ul role="list" className="flex flex-1 flex-col gap-y-2">
            {navigation.map((item) => {
              const isActive = pathname === item.href || (item.href !== '/' && pathname.startsWith(item.href));
              
              return (
                <li key={item.name}>
                  <Link
                    href={item.href}
                    onClick={() => setMobileMenuOpen(false)}
                    className={classNames(
                      isActive 
                        ? 'bg-brand-accent-primary/10 text-brand-accent-primary border-l-2 border-brand-accent-primary'
                        : 'text-brand-text-secondary hover:text-brand-text-primary hover:bg-brand-bg-glass border-l-2 border-transparent',
                      'group flex gap-x-3 rounded-r-md p-3 text-sm leading-6 font-semibold transition-all duration-200'
                    )}
                  >
                    <item.icon
                      className={classNames(
                        isActive ? 'text-brand-accent-primary' : 'text-brand-text-muted group-hover:text-brand-text-primary',
                        'h-6 w-6 shrink-0 transition-colors'
                      )}
                    />
                    {item.name}
                  </Link>
                </li>
              );
            })}
          </ul>
          
          <div className="mt-auto pt-6 px-2 text-xs text-brand-text-muted flex items-center gap-2">
            <div className="w-2 h-2 rounded-full bg-brand-accent-success animate-pulse-slow"></div>
            API Connected (v2.0.0)
          </div>
        </nav>
      </div>
    </>
  );
}
