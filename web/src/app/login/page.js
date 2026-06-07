'use client';

import React, { useState } from 'react';
import { useAuth } from '@/lib/auth';

export default function Login() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const { login } = useAuth();

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!email.trim() || !password.trim()) return;

    setIsSubmitting(true);
    setError('');

    try {
      await login(email, password);
    } catch (err) {
      setError(err.message || 'Invalid email or password. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center relative overflow-hidden">
      {/* Animated background elements */}
      <div className="absolute top-1/4 -left-20 w-96 h-96 bg-brand-accent-primary/20 rounded-full mix-blend-screen filter blur-[100px] animate-pulse-slow"></div>
      <div className="absolute bottom-1/4 -right-20 w-96 h-96 bg-brand-accent-info/20 rounded-full mix-blend-screen filter blur-[100px] animate-pulse-slow" style={{ animationDelay: '1s' }}></div>

      <div className="w-full max-w-md p-8 glass-card relative z-10 mx-4">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-brand-accent-primary to-brand-accent-info text-white font-bold text-2xl shadow-[0_0_30px_rgba(99,102,241,0.5)] mb-4">
            LP
          </div>
          <h1 className="text-3xl font-bold text-gradient">LedgerPro</h1>
          <p className="text-brand-text-secondary mt-2">Sign in to access your dashboard</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-brand-text-secondary mb-2">
              Email Address
            </label>
            <div className="relative">
              <input
                id="email"
                name="email"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full bg-brand-bg-secondary/50 border border-brand-border-glass rounded-lg px-4 py-3 text-brand-text-primary focus:outline-none focus:ring-2 focus:ring-brand-accent-primary/50 focus:border-brand-accent-primary transition-all backdrop-blur-sm"
                placeholder="you@example.com"
              />
            </div>
          </div>
          
          <div>
            <label htmlFor="password" className="block text-sm font-medium text-brand-text-secondary mb-2">
              Password
            </label>
            <div className="relative">
              <input
                id="password"
                name="password"
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-brand-bg-secondary/50 border border-brand-border-glass rounded-lg px-4 py-3 text-brand-text-primary focus:outline-none focus:ring-2 focus:ring-brand-accent-primary/50 focus:border-brand-accent-primary transition-all backdrop-blur-sm"
                placeholder="••••••••"
              />
            </div>
            {error && (
              <p className="mt-2 text-sm text-brand-accent-danger flex items-center gap-1">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                {error}
              </p>
            )}
          </div>

          <button
            type="submit"
            disabled={isSubmitting || !email.trim() || !password.trim()}
            className="w-full flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-brand-accent-primary hover:bg-brand-accent-primary-hover focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-brand-accent-primary disabled:opacity-50 disabled:cursor-not-allowed transition-all active:scale-[0.98]"
          >
            {isSubmitting ? (
              <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
            ) : (
              'Sign In'
            )}
          </button>
        </form>

        <div className="mt-6 text-center text-sm text-brand-text-secondary">
          Don&apos;t have an account?{' '}
          <a href="/signup" className="font-medium text-brand-accent-info hover:text-brand-accent-primary transition-colors">
            Sign up here
          </a>
        </div>

        <div className="mt-6 text-center text-xs text-brand-text-muted">
          Secured by HTTP-only Session Cookies & RFC 7807 Standard
        </div>
      </div>
    </div>
  );
}
