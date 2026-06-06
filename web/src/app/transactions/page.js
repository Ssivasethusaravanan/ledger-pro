'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import { getTransactions, createTransaction, getAccounts } from '@/lib/api';
import { formatRelativeTime, generateUUID, formatCurrency } from '@/lib/utils';
import GlassCard from '@/components/GlassCard';
import Button from '@/components/Button';
import Input from '@/components/Input';
import Select from '@/components/Select';
import Modal from '@/components/Modal';
import { IconSearch } from '@/components/Icons';
import { useAuth } from '@/lib/auth';

export default function TransactionsPage() {
  const { role } = useAuth();
  const [txns, setTxns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  
  // Modal State
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [accounts, setAccounts] = useState([]);
  const [idempotencyKey, setIdempotencyKey] = useState('');
  
  const [formData, setFormData] = useState({
    description: '',
    postings: [
      { account_id: '', amount: '', direction: 'debit' },
      { account_id: '', amount: '', direction: 'credit' }
    ]
  });
  
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState('');

  const fetchTransactions = async () => {
    try {
      setLoading(true);
      const res = await getTransactions('', 50);
      setTxns(res?.data || []);
    } catch (err) {
      setError('Failed to load transactions');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTransactions();
    // Pre-fetch accounts for the dropdown
    getAccounts(0, 100).then(res => {
      setAccounts(res?.data || []);
    }).catch(console.error);
  }, []);

  const openCreateModal = () => {
    setIdempotencyKey(generateUUID());
    setIsCreateOpen(true);
  };

  const addPostingRow = () => {
    setFormData({
      ...formData,
      postings: [...formData.postings, { account_id: '', amount: '', direction: 'debit' }]
    });
  };

  const removePostingRow = (index) => {
    if (formData.postings.length <= 2) return;
    const newPostings = [...formData.postings];
    newPostings.splice(index, 1);
    setFormData({ ...formData, postings: newPostings });
  };

  const updatePosting = (index, field, value) => {
    const newPostings = [...formData.postings];
    newPostings[index][field] = value;
    setFormData({ ...formData, postings: newPostings });
  };

  // Calculate balance
  const totalDebits = formData.postings
    .filter(p => p.direction === 'debit' && p.amount)
    .reduce((sum, p) => sum + parseInt(p.amount, 10), 0);
    
  const totalCredits = formData.postings
    .filter(p => p.direction === 'credit' && p.amount)
    .reduce((sum, p) => sum + parseInt(p.amount, 10), 0);

  const isBalanced = totalDebits === totalCredits && totalDebits > 0;

  const handleCreateSubmit = async (e) => {
    e.preventDefault();
    setSubmitError('');

    if (!isBalanced) {
      setSubmitError('Transaction is unbalanced. Total debits must equal total credits.');
      return;
    }

    // Convert string amounts to integers
    const payload = {
      description: formData.description,
      postings: formData.postings.map(p => ({
        account_id: parseInt(p.account_id, 10),
        amount: parseInt(p.amount, 10),
        direction: p.direction
      }))
    };

    setIsSubmitting(true);

    try {
      await createTransaction(payload, idempotencyKey);
      setIsCreateOpen(false);
      setFormData({
        description: '',
        postings: [
          { account_id: '', amount: '', direction: 'debit' },
          { account_id: '', amount: '', direction: 'credit' }
        ]
      });
      fetchTransactions();
    } catch (err) {
      setSubmitError(err.message || 'Failed to create transaction');
    } finally {
      setIsSubmitting(false);
    }
  };

  const canCreate = role === 'admin' || role === 'write';

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">Ledger Transactions</h1>
          <p className="text-brand-text-secondary mt-1 text-sm">Immutable records of double-entry financial movements.</p>
        </div>
        
        {canCreate && (
          <Button onClick={openCreateModal} className="w-full sm:w-auto">
            Create Transaction
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
              placeholder="Search transactions..."
            />
          </div>
        </div>

        {loading ? (
          <div className="p-8 space-y-4">
            {[1,2,3,4,5].map(i => <div key={i} className="h-12 bg-brand-bg-glass animate-pulse rounded-lg"></div>)}
          </div>
        ) : error ? (
          <div className="p-8 text-center text-brand-accent-danger">{error}</div>
        ) : txns.length === 0 ? (
          <div className="p-16 text-center">
            <div className="mx-auto w-16 h-16 rounded-full bg-brand-bg-glass flex items-center justify-center mb-4">
              <svg className="w-8 h-8 text-brand-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-white">No transactions</h3>
            <p className="mt-1 text-brand-text-secondary">Get started by creating your first transaction.</p>
          </div>
        ) : (
          <div className="overflow-x-auto hide-scrollbar">
            <table className="min-w-full divide-y divide-brand-border-glass">
              <thead className="bg-brand-bg-secondary/20">
                <tr>
                  <th scope="col" className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-brand-text-secondary sm:pl-6">Transaction ID</th>
                  <th scope="col" className="px-3 py-3.5 text-left text-sm font-semibold text-brand-text-secondary">Description</th>
                  <th scope="col" className="px-3 py-3.5 text-right text-sm font-semibold text-brand-text-secondary">Time</th>
                  <th scope="col" className="relative py-3.5 pl-3 pr-4 sm:pr-6"><span className="sr-only">View</span></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-border-glass">
                {txns.map((txn) => (
                  <tr key={txn.id} className="hover:bg-brand-bg-glass/50 transition-colors group">
                    <td className="whitespace-nowrap py-4 pl-4 pr-3 sm:pl-6 text-sm font-mono text-brand-accent-primary">
                      {txn.id}
                    </td>
                    <td className="whitespace-nowrap px-3 py-4 text-sm text-white">
                      {txn.description || <span className="text-brand-text-muted italic">No description</span>}
                    </td>
                    <td className="whitespace-nowrap px-3 py-4 text-sm text-brand-text-secondary text-right">
                      {formatRelativeTime(txn.created_at)}
                    </td>
                    <td className="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6">
                      <Link href={`/transactions/${txn.id}`} className="text-brand-accent-primary hover:text-brand-accent-primary-hover opacity-0 group-hover:opacity-100 transition-opacity">
                        View Details
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </GlassCard>

      <Modal open={isCreateOpen} setOpen={setIsCreateOpen} title="Create Double-Entry Transaction" size="lg">
        <form onSubmit={handleCreateSubmit} className="space-y-6">
          <div className="bg-brand-bg-secondary/50 p-4 rounded-lg border border-brand-border-glass flex items-center justify-between mb-6">
            <div>
              <p className="text-xs text-brand-text-muted uppercase tracking-wider mb-1">Idempotency Key</p>
              <p className="font-mono text-sm text-brand-text-primary">{idempotencyKey}</p>
            </div>
            <Badge variant="primary">Auto-generated</Badge>
          </div>

          <Input
            label="Description"
            name="description"
            value={formData.description}
            onChange={(e) => setFormData({...formData, description: e.target.value})}
            placeholder="e.g. Funding account, Payment received"
          />

          <div>
            <div className="flex items-center justify-between mb-3">
              <label className="block text-sm font-medium text-brand-text-secondary">Postings</label>
              <button 
                type="button" 
                onClick={addPostingRow}
                className="text-sm text-brand-accent-primary hover:text-white transition-colors"
              >
                + Add Row
              </button>
            </div>

            <div className="space-y-3">
              {formData.postings.map((posting, index) => (
                <div key={index} className="flex gap-3 items-start p-3 bg-brand-bg-secondary/30 rounded-lg border border-brand-border-glass">
                  <div className="flex-1">
                    <Select
                      options={[
                        { value: '', label: 'Select Account...' },
                        ...accounts.map(a => ({ value: a.id, label: `${a.account_name} (${a.currency})` }))
                      ]}
                      value={posting.account_id}
                      onChange={(e) => updatePosting(index, 'account_id', e.target.value)}
                      required
                    />
                  </div>
                  
                  <div className="w-32">
                    <Select
                      options={[
                        { value: 'debit', label: 'Debit' },
                        { value: 'credit', label: 'Credit' }
                      ]}
                      value={posting.direction}
                      onChange={(e) => updatePosting(index, 'direction', e.target.value)}
                    />
                  </div>

                  <div className="w-32">
                    <div className="relative rounded-md shadow-sm">
                      <input
                        type="number"
                        min="1"
                        required
                        placeholder="Amount"
                        value={posting.amount}
                        onChange={(e) => updatePosting(index, 'amount', e.target.value)}
                        className="block w-full rounded-lg bg-brand-bg-secondary/50 border border-brand-border-glass py-2.5 pl-3 pr-3 text-brand-text-primary focus:outline-none focus:border-brand-accent-primary sm:text-sm"
                      />
                    </div>
                  </div>

                  {formData.postings.length > 2 && (
                    <button 
                      type="button" 
                      onClick={() => removePostingRow(index)}
                      className="mt-2.5 text-brand-text-muted hover:text-brand-accent-danger transition-colors"
                    >
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  )}
                </div>
              ))}
            </div>

            {/* Balance Checker */}
            <div className="mt-4 flex items-center justify-between p-4 rounded-lg bg-brand-bg-primary border border-brand-border-glass">
              <div className="flex gap-8">
                <div>
                  <p className="text-xs text-brand-text-muted uppercase mb-1">Total Debits</p>
                  <p className={`font-medium ${totalDebits > 0 ? 'text-brand-accent-info' : 'text-brand-text-secondary'}`}>
                    {totalDebits.toLocaleString()}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-brand-text-muted uppercase mb-1">Total Credits</p>
                  <p className={`font-medium ${totalCredits > 0 ? 'text-brand-accent-warning' : 'text-brand-text-secondary'}`}>
                    {totalCredits.toLocaleString()}
                  </p>
                </div>
              </div>
              <div>
                {isBalanced ? (
                  <Badge variant="success">Balanced</Badge>
                ) : (
                  <Badge variant="danger">Unbalanced</Badge>
                )}
              </div>
            </div>
          </div>

          {submitError && (
            <div className="p-3 rounded-md bg-brand-accent-danger/10 border border-brand-accent-danger/20 text-brand-accent-danger text-sm">
              {submitError}
            </div>
          )}

          <div className="pt-4 border-t border-brand-border-glass flex justify-end gap-3">
            <Button type="button" variant="ghost" onClick={() => setIsCreateOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" isLoading={isSubmitting} disabled={!isBalanced}>
              Record Transaction
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
