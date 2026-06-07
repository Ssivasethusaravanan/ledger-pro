'use client';

import React, { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { getTransaction, getDocuments, uploadDocument } from '@/lib/api';
import { formatRelativeTime, formatCurrency, formatDate } from '@/lib/utils';
import GlassCard from '@/components/GlassCard';
import Badge from '@/components/Badge';
import Button from '@/components/Button';
import { useAuth } from '@/lib/auth';

export default function TransactionDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id;
  const { role } = useAuth();
  
  const [txn, setTxn] = useState(null);
  const [documents, setDocuments] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Upload state
  const [isUploading, setIsUploading] = useState(false);
  const [uploadError, setUploadError] = useState('');

  const fetchData = React.useCallback(async () => {
    try {
      setLoading(true);
      const [txnRes, docRes] = await Promise.all([
        getTransaction(id),
        getDocuments(id)
      ]);
      setTxn(txnRes);
      setDocuments(docRes?.data || []);
    } catch (err) {
      setError('Failed to load transaction details');
      if (!err.message?.includes('authentication required')) {
        console.error(err);
      }
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (id) {
      fetchData();
    }
  }, [id, fetchData]);

  const handleFileUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setIsUploading(true);
    setUploadError('');

    try {
      await uploadDocument(id, file);
      // Refresh documents
      const docRes = await getDocuments(id);
      setDocuments(docRes?.data || []);
    } catch (err) {
      setUploadError(err.message || 'Failed to upload document');
    } finally {
      setIsUploading(false);
      // Reset input
      e.target.value = null;
    }
  };

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-8 w-32 bg-brand-bg-glass rounded mb-6"></div>
        <div className="h-48 bg-brand-bg-glass rounded-2xl"></div>
        <div className="h-96 bg-brand-bg-glass rounded-2xl"></div>
      </div>
    );
  }

  if (error || !txn) {
    return (
      <div className="text-center py-20">
        <h2 className="text-xl font-semibold text-white">Transaction Not Found</h2>
        <p className="mt-2 text-brand-text-secondary">{error || "The transaction you're looking for doesn't exist."}</p>
        <Button className="mt-6" onClick={() => router.push('/transactions')}>Back to Transactions</Button>
      </div>
    );
  }

  const canUpload = role === 'admin' || role === 'write';

  // Calculate total volume
  const totalVolume = txn.postings
    ?.filter(p => p.direction === 'debit')
    ?.reduce((sum, p) => sum + p.amount, 0) || 0;

  return (
    <div className="space-y-6">
      <div>
        <button 
          onClick={() => router.push('/transactions')}
          className="flex items-center text-sm font-medium text-brand-text-secondary hover:text-white transition-colors mb-4 group"
        >
          <svg className="mr-2 h-4 w-4 transform group-hover:-translate-x-1 transition-transform" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Back to Transactions
        </button>
        
        <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
          <div>
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-3xl font-bold text-white tracking-tight">Transaction</h1>
              <Badge variant="success">Committed</Badge>
            </div>
            <p className="text-brand-accent-info text-sm font-mono">{txn.id}</p>
          </div>
          <div className="text-right">
            <p className="text-sm text-brand-text-secondary mb-1">Time</p>
            <p className="text-white text-sm font-medium">{formatDate(txn.created_at)}</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <GlassCard className="lg:col-span-2">
          <h2 className="text-lg font-medium text-white mb-4">Postings</h2>
          
          <div className="overflow-x-auto hide-scrollbar">
            <table className="min-w-full divide-y divide-brand-border-glass">
              <thead className="bg-brand-bg-secondary/20">
                <tr>
                  <th scope="col" className="py-3 pl-4 pr-3 text-left text-xs font-medium text-brand-text-secondary uppercase tracking-wider">Account ID</th>
                  <th scope="col" className="px-3 py-3 text-left text-xs font-medium text-brand-text-secondary uppercase tracking-wider">Direction</th>
                  <th scope="col" className="px-3 py-3 text-right text-xs font-medium text-brand-text-secondary uppercase tracking-wider">Amount</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-border-glass">
                {txn.postings?.map((posting, idx) => (
                  <tr key={idx} className={posting.direction === 'debit' ? 'bg-brand-accent-info/5' : 'bg-brand-accent-warning/5'}>
                    <td className="whitespace-nowrap py-3 pl-4 pr-3 text-sm font-mono text-brand-text-primary">
                      {posting.account_id}
                    </td>
                    <td className="whitespace-nowrap px-3 py-3 text-sm">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium uppercase ${
                        posting.direction === 'debit' ? 'bg-brand-accent-info/20 text-brand-accent-info' : 'bg-brand-accent-warning/20 text-brand-accent-warning'
                      }`}>
                        {posting.direction}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-3 py-3 text-sm font-medium text-right text-white">
                      {posting.amount.toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
              <tfoot className="bg-brand-bg-secondary/40 border-t border-brand-border-glass">
                <tr>
                  <td colSpan={2} className="px-4 py-3 text-sm font-medium text-brand-text-secondary text-right uppercase tracking-wider">
                    Total Volume
                  </td>
                  <td className="px-3 py-3 text-sm font-bold text-white text-right">
                    {totalVolume.toLocaleString()}
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>
          
          <div className="mt-6">
            <h3 className="text-sm font-medium text-brand-text-secondary mb-2">Description</h3>
            <p className="text-white">{txn.description || <span className="text-brand-text-muted italic">No description provided</span>}</p>
          </div>
        </GlassCard>

        <div className="space-y-6">
          <GlassCard>
            <h2 className="text-lg font-medium text-white mb-4">Metadata</h2>
            <div className="space-y-4">
              <div>
                <p className="text-xs text-brand-text-muted uppercase tracking-wider mb-1">Idempotency Key</p>
                <p className="font-mono text-sm text-brand-text-primary break-all bg-brand-bg-secondary p-2 rounded border border-brand-border-glass">
                  {txn.idempotency_key}
                </p>
              </div>
            </div>
          </GlassCard>

          <GlassCard>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-medium text-white">Documents</h2>
              <Badge variant="primary">{documents.length}</Badge>
            </div>
            
            {documents.length > 0 ? (
              <ul className="space-y-3 mb-6">
                {documents.map((doc) => (
                  <li key={doc.id} className="flex items-center justify-between p-3 rounded-lg bg-brand-bg-secondary/50 border border-brand-border-glass group">
                    <div className="flex items-center gap-3 overflow-hidden">
                      <svg className="w-8 h-8 text-brand-accent-primary shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                      </svg>
                      <div className="overflow-hidden">
                        <p className="text-sm font-medium text-white truncate">{doc.filename}</p>
                        <p className="text-xs text-brand-text-secondary">{(doc.size_bytes / 1024).toFixed(1)} KB • {formatRelativeTime(doc.created_at)}</p>
                      </div>
                    </div>
                    <a 
                      href={doc.download_url} 
                      target="_blank" 
                      rel="noopener noreferrer"
                      className="p-2 text-brand-text-muted hover:text-brand-accent-primary hover:bg-brand-bg-primary rounded-md transition-colors"
                      title="Download"
                    >
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                      </svg>
                    </a>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-brand-text-secondary mb-6 italic">No documents attached.</p>
            )}

            {canUpload && (
              <div>
                {uploadError && <p className="text-brand-accent-danger text-xs mb-2">{uploadError}</p>}
                
                <div className="relative">
                  <input 
                    type="file" 
                    id="file-upload" 
                    className="sr-only" 
                    onChange={handleFileUpload}
                    disabled={isUploading}
                  />
                  <label 
                    htmlFor="file-upload" 
                    className={`flex items-center justify-center w-full px-4 py-2 text-sm font-medium text-white bg-brand-bg-secondary border border-brand-border-glass rounded-lg hover:bg-brand-bg-secondary/80 cursor-pointer transition-colors ${isUploading ? 'opacity-50 cursor-not-allowed' : ''}`}
                  >
                    {isUploading ? (
                      <>
                        <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        Uploading...
                      </>
                    ) : (
                      <>
                        <svg className="mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                        </svg>
                        Upload Document
                      </>
                    )}
                  </label>
                </div>
              </div>
            )}
          </GlassCard>
        </div>
      </div>
    </div>
  );
}
