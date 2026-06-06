'use client';

import React, { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { getAccount, getAccountBalance, getAccountPostings } from '@/lib/api';
import { formatCurrency, formatDate, formatRelativeTime } from '@/lib/utils';
import Link from 'next/link';
import GlassCard from '@/components/GlassCard';
import Badge from '@/components/Badge';
import Button from '@/components/Button';

export default function AccountDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id;
  
  const [account, setAccount] = useState(null);
  const [balance, setBalance] = useState(null);
  const [postings, setPostings] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const [accRes, balRes, postRes] = await Promise.all([
          getAccount(id),
          getAccountBalance(id),
          getAccountPostings(id)
        ]);
        setAccount(accRes);
        setBalance(balRes);
        setPostings(postRes?.data || []);
      } catch (err) {
        setError('Failed to load account details');
        console.error(err);
      } finally {
        setLoading(false);
      }
    }
    
    if (id) {
      fetchData();
    }
  }, [id]);

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-8 w-32 bg-brand-bg-glass rounded mb-6"></div>
        <div className="h-48 bg-brand-bg-glass rounded-2xl"></div>
        <div className="h-96 bg-brand-bg-glass rounded-2xl"></div>
      </div>
    );
  }

  if (error || !account) {
    return (
      <div className="text-center py-20">
        <h2 className="text-xl font-semibold text-white">Account Not Found</h2>
        <p className="mt-2 text-brand-text-secondary">{error || "The account you're looking for doesn't exist or you don't have permission to view it."}</p>
        <Button className="mt-6" onClick={() => router.push('/accounts')}>Back to Accounts</Button>
      </div>
    );
  }

  const getBadgeVariant = (type) => {
    switch(type) {
      case 'asset': return 'primary';
      case 'liability': return 'danger';
      case 'equity': return 'info';
      case 'revenue': return 'success';
      case 'expense': return 'warning';
      default: return 'default';
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <button 
          onClick={() => router.push('/accounts')}
          className="flex items-center text-sm font-medium text-brand-text-secondary hover:text-white transition-colors mb-4 group"
        >
          <svg className="mr-2 h-4 w-4 transform group-hover:-translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Back to Accounts
        </button>
        
        <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
          <div>
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-3xl font-bold text-white tracking-tight">{account.account_name}</h1>
              <Badge variant={getBadgeVariant(account.account_type)} className="capitalize">
                {account.account_type}
              </Badge>
            </div>
            <p className="text-brand-text-secondary text-sm font-mono">{account.id}</p>
          </div>
          <div className="text-right">
            <p className="text-sm text-brand-text-secondary mb-1">Created</p>
            <p className="text-white text-sm font-medium">{formatDate(account.created_at)}</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <GlassCard className="lg:col-span-1 flex flex-col justify-center">
          <h2 className="text-sm font-medium text-brand-text-secondary mb-2">Current Balance</h2>
          <div className="text-4xl font-bold text-white tracking-tight break-words">
            {formatCurrency(balance?.balance || 0, account.currency)}
          </div>
          
          <div className="mt-6 pt-6 border-t border-brand-border-glass grid grid-cols-2 gap-4">
            <div>
              <p className="text-xs text-brand-text-muted mb-1 uppercase tracking-wider">Total Debits</p>
              <p className="text-brand-text-primary font-medium">{formatCurrency(balance?.total_debits || 0, account.currency)}</p>
            </div>
            <div>
              <p className="text-xs text-brand-text-muted mb-1 uppercase tracking-wider">Total Credits</p>
              <p className="text-brand-text-primary font-medium">{formatCurrency(balance?.total_credits || 0, account.currency)}</p>
            </div>
          </div>
        </GlassCard>

        <GlassCard className="lg:col-span-2">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium text-white">Metadata</h2>
          </div>
          <div className="bg-brand-bg-secondary/50 rounded-lg p-4 font-mono text-sm text-brand-text-secondary overflow-x-auto border border-brand-border-glass">
            {account.metadata ? (
              <pre>{JSON.stringify(account.metadata, null, 2)}</pre>
            ) : (
              <span className="text-brand-text-muted italic">No metadata attached</span>
            )}
          </div>
        </GlassCard>
      </div>

      <GlassCard noPadding className="overflow-hidden">
        <div className="p-4 sm:p-6 border-b border-brand-border-glass flex justify-between items-center bg-brand-bg-secondary/30">
          <h2 className="text-lg font-medium text-white">Recent Postings</h2>
          <Badge variant="primary">{postings.length}</Badge>
        </div>
        
        {postings.length === 0 ? (
          <div className="p-16 text-center">
              <div className="mx-auto w-16 h-16 rounded-full bg-brand-bg-glass flex items-center justify-center mb-4">
                <svg className="w-8 h-8 text-brand-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-white">No Postings</h3>
              <p className="mt-1 text-brand-text-secondary max-w-sm mx-auto">This account has no postings yet.</p>
          </div>
        ) : (
          <div className="overflow-x-auto hide-scrollbar">
            <table className="min-w-full divide-y divide-brand-border-glass">
              <thead className="bg-brand-bg-secondary/20">
                <tr>
                  <th scope="col" className="py-3 pl-4 pr-3 text-left text-xs font-medium text-brand-text-secondary uppercase tracking-wider">Transaction</th>
                  <th scope="col" className="px-3 py-3 text-left text-xs font-medium text-brand-text-secondary uppercase tracking-wider">Time</th>
                  <th scope="col" className="px-3 py-3 text-left text-xs font-medium text-brand-text-secondary uppercase tracking-wider">Direction</th>
                  <th scope="col" className="px-3 py-3 text-right text-xs font-medium text-brand-text-secondary uppercase tracking-wider">Amount</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-border-glass">
                {postings.map((posting) => (
                  <tr key={posting.id} className="hover:bg-brand-bg-glass/50 transition-colors group">
                    <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm">
                      <div className="flex flex-col">
                        <Link href={`/transactions/${posting.transaction_id}`} className="font-mono text-brand-accent-primary hover:text-brand-accent-primary-hover transition-colors">
                          {posting.transaction_id.substring(0, 8)}...
                        </Link>
                        <span className="text-brand-text-secondary text-xs mt-1 truncate max-w-[200px]" title={posting.transaction_description}>
                          {posting.transaction_description || 'No description'}
                        </span>
                      </div>
                    </td>
                    <td className="whitespace-nowrap px-3 py-4 text-sm text-brand-text-secondary">
                      {formatRelativeTime(posting.created_at)}
                    </td>
                    <td className="whitespace-nowrap px-3 py-4 text-sm">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium uppercase ${
                        posting.direction === 'debit' ? 'bg-brand-accent-info/20 text-brand-accent-info' : 'bg-brand-accent-warning/20 text-brand-accent-warning'
                      }`}>
                        {posting.direction}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-3 py-4 text-sm font-medium text-right text-white">
                      {formatCurrency(posting.amount, account.currency)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </GlassCard>
    </div>
  );
}
