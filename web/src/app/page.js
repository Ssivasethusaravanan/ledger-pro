'use client';

import React, { useEffect, useState } from 'react';
import { getHealth, getAccounts, getTransactions } from '@/lib/api';
import MetricCard from '@/components/MetricCard';
import GlassCard from '@/components/GlassCard';
import { IconAccounts, IconTransactions } from '@/components/Icons';
import { formatCurrency, formatRelativeTime } from '@/lib/utils';
import Badge from '@/components/Badge';

export default function Dashboard() {
  const [data, setData] = useState({ accounts: [], txns: [] });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchData() {
      try {
        const [accRes, txnRes] = await Promise.all([
          getAccounts(0, 5),
          getTransactions('', 10)
        ]);
        setData({ 
          accounts: accRes?.data || [], 
          txns: txnRes?.data || [] 
        });
      } catch (e) {
        console.error(e);
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard 
          title="Total Accounts" 
          value={data.accounts.length} 
          icon={IconAccounts} 
          loading={loading} 
        />
        <MetricCard 
          title="Recent Transactions" 
          value={data.txns.length} 
          icon={IconTransactions} 
          loading={loading} 
        />
        <MetricCard 
          title="System Health" 
          value="Healthy" 
          format="text"
          loading={loading} 
        />
        <MetricCard 
          title="API Version" 
          value="v2.0.0" 
          format="text"
          loading={loading} 
        />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <GlassCard className="h-full">
            <h2 className="text-lg font-medium text-white mb-4">Recent Transactions</h2>
            
            {loading ? (
              <div className="space-y-4">
                {[1,2,3].map(i => <div key={i} className="h-16 bg-brand-bg-glass animate-pulse rounded-lg"></div>)}
              </div>
            ) : data.txns.length === 0 ? (
              <div className="text-center py-10">
                <IconTransactions className="mx-auto h-12 w-12 text-brand-text-muted opacity-50" />
                <h3 className="mt-2 text-sm font-semibold text-white">No transactions</h3>
                <p className="mt-1 text-sm text-brand-text-secondary">Get started by creating a new transaction.</p>
              </div>
            ) : (
              <div className="overflow-x-auto hide-scrollbar">
                <table className="min-w-full divide-y divide-brand-border-glass">
                  <thead>
                    <tr>
                      <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-brand-text-secondary">ID</th>
                      <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-brand-text-secondary">Description</th>
                      <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-brand-text-secondary">Time</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-brand-border-glass">
                    {data.txns.map((txn) => (
                      <tr key={txn.id} className="hover:bg-brand-bg-glass/50 transition-colors">
                        <td className="whitespace-nowrap px-3 py-4 text-sm font-mono text-brand-accent-info">
                          {txn.id.substring(0, 8)}...
                        </td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-white">
                          {txn.description || 'No description'}
                        </td>
                        <td className="whitespace-nowrap px-3 py-4 text-sm text-brand-text-secondary">
                          {formatRelativeTime(txn.created_at)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </GlassCard>
        </div>

        <div>
          <GlassCard className="h-full">
            <h2 className="text-lg font-medium text-white mb-4">Top Accounts</h2>
            
            {loading ? (
              <div className="space-y-4">
                {[1,2,3].map(i => <div key={i} className="h-12 bg-brand-bg-glass animate-pulse rounded-lg"></div>)}
              </div>
            ) : data.accounts.length === 0 ? (
              <div className="text-center py-10 text-brand-text-muted text-sm">No accounts found.</div>
            ) : (
              <ul className="space-y-3">
                {data.accounts.map((acc) => (
                  <li key={acc.id} className="flex items-center justify-between p-3 rounded-lg bg-brand-bg-secondary/50 border border-brand-border-glass">
                    <div>
                      <p className="text-sm font-medium text-white">{acc.account_name}</p>
                      <Badge variant="primary" size="sm" className="mt-1">{acc.account_type}</Badge>
                    </div>
                    <div className="text-right">
                      <p className="text-xs text-brand-text-secondary uppercase">{acc.currency}</p>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </GlassCard>
        </div>
      </div>
    </div>
  );
}
