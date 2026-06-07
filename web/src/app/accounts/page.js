'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { getAccounts, createAccount } from '@/lib/api';
import { formatRelativeTime } from '@/lib/utils';
import GlassCard from '@/components/GlassCard';
import Button from '@/components/Button';
import Input from '@/components/Input';
import Select from '@/components/Select';
import Badge from '@/components/Badge';
import SlideOver from '@/components/SlideOver';
import { IconSearch } from '@/components/Icons';
import { useAuth } from '@/lib/auth';

export default function AccountsPage() {
  const { role } = useAuth();
  const [accounts, setAccounts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  
  // SlideOver State
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [formData, setFormData] = useState({
    account_name: '',
    account_type: 'asset',
    currency: 'USD',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState('');

  const fetchAccounts = async () => {
    try {
      setLoading(true);
      const res = await getAccounts(0, 50); // Get first 50
      setAccounts(res?.data || []);
    } catch (err) {
      setError('Failed to load accounts');
      if (!err.message?.includes('authentication required')) {
        console.error(err);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, []);

  const handleCreateSubmit = async (e) => {
    e.preventDefault();
    setSubmitError('');
    setIsSubmitting(true);

    try {
      await createAccount(formData);
      setIsCreateOpen(false);
      setFormData({ account_name: '', account_type: 'asset', currency: 'USD' });
      fetchAccounts(); // Refresh list
    } catch (err) {
      setSubmitError(err.message || 'Failed to create account');
    } finally {
      setIsSubmitting(false);
    }
  };

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

  const canCreate = role === 'admin' || role === 'write';

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">Chart of Accounts</h1>
          <p className="text-brand-text-secondary mt-1 text-sm">Manage your ledger accounts and view balances.</p>
        </div>
        
        {canCreate && (
          <Button onClick={() => setIsCreateOpen(true)} className="w-full sm:w-auto">
            Create Account
          </Button>
        )}
      </div>

      <GlassCard noPadding className="overflow-hidden">
        <div className="p-4 border-b border-brand-border-glass bg-brand-bg-secondary/30 flex items-center gap-4">
          <div className="relative flex-1 max-w-md">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <IconSearch className="h-5 w-5 text-brand-text-muted" />
            </div>
            <input
              type="text"
              className="block w-full rounded-md bg-brand-bg-primary border border-brand-border-glass py-2 pl-10 pr-3 text-sm text-brand-text-primary placeholder-brand-text-muted focus:border-brand-accent-primary focus:outline-none focus:ring-1 focus:ring-brand-accent-primary transition-colors"
              placeholder="Search accounts..."
            />
          </div>
        </div>

        {loading ? (
          <div className="p-8 space-y-4">
            {[1,2,3,4,5].map(i => <div key={i} className="h-12 bg-brand-bg-glass animate-pulse rounded-lg"></div>)}
          </div>
        ) : error ? (
          <div className="p-8 text-center text-brand-accent-danger">{error}</div>
        ) : accounts.length === 0 ? (
          <div className="p-16 text-center">
            <div className="mx-auto w-16 h-16 rounded-full bg-brand-bg-glass flex items-center justify-center mb-4">
              <svg className="w-8 h-8 text-brand-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-white">No accounts</h3>
            <p className="mt-1 text-brand-text-secondary">Get started by creating your first account.</p>
          </div>
        ) : (
          <div className="overflow-x-auto hide-scrollbar">
            <table className="min-w-full divide-y divide-brand-border-glass">
              <thead className="bg-brand-bg-secondary/20">
                <tr>
                  <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-brand-text-secondary sm:pl-6">Name</th>
                  <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-brand-text-secondary">Type</th>
                  <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-brand-text-secondary">Currency</th>
                  <th scope="col" className="px-3 py-3.5 text-right text-sm font-semibold text-brand-text-secondary">Created</th>
                  <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">View</span></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-border-glass">
                {accounts.map((account) => (
                  <tr key={account.id} className="hover:bg-brand-bg-glass/50 transition-colors group">
                    <td className="whitespace-nowrap py-4 pl-4 pr-3 sm:pl-6 text-sm font-medium text-white">
                      {account.account_name}
                    </td>
                    <td className="whitespace-nowrap px-3 py-4 text-sm">
                      <Badge variant={getBadgeVariant(account.account_type)} className="capitalize">
                        {account.account_type}
                      </Badge>
                    </td>
                    <td className="whitespace-nowrap px-3 py-4 text-sm">
                      <span className="inline-flex items-center rounded-md bg-brand-bg-glass px-2 py-1 text-xs font-medium text-brand-text-secondary border border-brand-border-glass uppercase">
                        {account.currency}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-3 py-4 text-sm text-brand-text-secondary text-right">
                      {formatRelativeTime(account.created_at)}
                    </td>
                    <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6">
                      <Link href={`/accounts/${account.id}`} className="text-brand-accent-primary hover:text-brand-accent-primary-hover opacity-0 group-hover:opacity-100 transition-opacity">
                        View<span className="sr-only">, {account.account_name}</span>
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </GlassCard>

      <SlideOver open={isCreateOpen} setOpen={setIsCreateOpen} title="Create Account">
        <form onSubmit={handleCreateSubmit} className="space-y-6">
          <Input
            label="Account Name"
            name="account_name"
            value={formData.account_name}
            onChange={(e) => setFormData({...formData, account_name: e.target.value})}
            required
            placeholder="e.g. Cash, Revenue, Taxes"
          />

          <Select
            label="Account Type"
            name="account_type"
            value={formData.account_type}
            onChange={(e) => setFormData({...formData, account_type: e.target.value})}
            options={[
              { value: 'asset', label: 'Asset' },
              { value: 'liability', label: 'Liability' },
              { value: 'equity', label: 'Equity' },
              { value: 'revenue', label: 'Revenue' },
              { value: 'expense', label: 'Expense' },
            ]}
          />

          <Select
            label="Currency"
            name="currency"
            value={formData.currency}
            onChange={(e) => setFormData({...formData, currency: e.target.value})}
            options={[
              { value: 'USD', label: 'USD - US Dollar' },
              { value: 'EUR', label: 'EUR - Euro' },
              { value: 'GBP', label: 'GBP - British Pound' },
              { value: 'INR', label: 'INR - Indian Rupee' },
              { value: 'JPY', label: 'JPY - Japanese Yen' },
            ]}
          />

          {submitError && (
            <div className="p-3 rounded-md bg-brand-accent-danger/10 border border-brand-accent-danger/20 text-brand-accent-danger text-sm">
              {submitError}
            </div>
          )}

          <div className="pt-4 border-t border-brand-border-glass flex justify-end gap-3">
            <Button type="button" variant="ghost" onClick={() => setIsCreateOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" isLoading={isSubmitting}>
              Create Account
            </Button>
          </div>
        </form>
      </SlideOver>
    </div>
  );
}
