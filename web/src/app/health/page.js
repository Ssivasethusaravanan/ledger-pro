'use client';

import React, { useState, useEffect } from 'react';
import { getHealth } from '@/lib/api';
import GlassCard from '@/components/GlassCard';

const StatusDot = ({ status }) => {
  if (status === 'up') {
    return <div className="w-3 h-3 rounded-full bg-brand-accent-success shadow-[0_0_8px_rgba(16,185,129,0.8)] animate-pulse-slow"></div>;
  }
  return <div className="w-3 h-3 rounded-full bg-brand-accent-danger shadow-[0_0_8px_rgba(239,68,68,0.8)]"></div>;
};

export default function HealthPage() {
  const [health, setHealth] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [lastUpdated, setLastUpdated] = useState(new Date());

  const fetchHealth = async () => {
    try {
      const res = await getHealth();
      setHealth(res);
      setError('');
      setLastUpdated(new Date());
    } catch (err) {
      setError('Failed to connect to health endpoint. System may be down.');
      if (!err.message?.includes('authentication required')) {
        console.error(err);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHealth();
    // Poll every 10 seconds
    const interval = setInterval(fetchHealth, 10000);
    return () => clearInterval(interval);
  }, []);

  if (loading && !health) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-8 w-48 bg-brand-bg-glass rounded mb-6"></div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="h-40 bg-brand-bg-glass rounded-2xl"></div>
          <div className="h-40 bg-brand-bg-glass rounded-2xl"></div>
          <div className="h-40 bg-brand-bg-glass rounded-2xl"></div>
        </div>
      </div>
    );
  }

  const isHealthy = health?.status === 'pass';

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-3">
            System Health
            {isHealthy ? (
              <span className="inline-flex items-center rounded-md bg-brand-accent-success/10 px-2 py-1 text-xs font-medium text-brand-accent-success border border-brand-accent-success/20">
                All Systems Operational
              </span>
            ) : (
              <span className="inline-flex items-center rounded-md bg-brand-accent-danger/10 px-2 py-1 text-xs font-medium text-brand-accent-danger border border-brand-accent-danger/20">
                System Degraded
              </span>
            )}
          </h1>
          <p className="text-brand-text-secondary mt-1 text-sm">Live status of backend infrastructure components.</p>
        </div>
        <div className="text-right">
          <p className="text-xs text-brand-text-muted mb-1">Last Updated</p>
          <p className="text-sm text-white font-medium tabular-nums">{lastUpdated.toLocaleTimeString()}</p>
        </div>
      </div>

      {error && (
        <div className="p-4 rounded-lg bg-brand-accent-danger/10 border border-brand-accent-danger/20 text-brand-accent-danger">
          {error}
        </div>
      )}

      {health && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* PostgreSQL */}
            <GlassCard className="relative overflow-hidden group">
              <div className="absolute top-0 right-0 p-4 opacity-5">
                <svg className="w-20 h-20 text-brand-text-primary" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 2C6.48 2 2 4.02 2 6.5C2 8.98 6.48 11 12 11C17.52 11 22 8.98 22 6.5C22 4.02 17.52 2 12 2ZM12 13C6.48 13 2 10.98 2 8.5V11.5C2 13.98 6.48 16 12 16C17.52 16 22 13.98 22 11.5V8.5C22 10.98 17.52 13 12 13ZM12 18C6.48 18 2 15.98 2 13.5V16.5C2 18.98 6.48 21 12 21C17.52 21 22 18.98 22 16.5V13.5C22 15.98 17.52 18 12 18Z" />
                </svg>
              </div>
              <div className="relative z-10">
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-lg font-medium text-white">PostgreSQL</h2>
                  <StatusDot status={health.checks?.postgres?.status} />
                </div>
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-brand-text-secondary">Latency</span>
                    <span className="text-white font-mono">{health.checks?.postgres?.latency_ms || 0} ms</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-brand-text-secondary">Total Connections</span>
                    <span className="text-white font-mono">{health.checks?.postgres?.stats?.total_connections || 0}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-brand-text-secondary">Idle Connections</span>
                    <span className="text-white font-mono">{health.checks?.postgres?.stats?.idle_connections || 0}</span>
                  </div>
                </div>
              </div>
            </GlassCard>

            {/* Redis */}
            <GlassCard className="relative overflow-hidden group">
              <div className="absolute top-0 right-0 p-4 opacity-5">
                <svg className="w-20 h-20 text-brand-accent-danger" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 2C6.48 2 2 4.02 2 6.5C2 8.98 6.48 11 12 11C17.52 11 22 8.98 22 6.5C22 4.02 17.52 2 12 2ZM12 13C6.48 13 2 10.98 2 8.5V11.5C2 13.98 6.48 16 12 16C17.52 16 22 13.98 22 11.5V8.5C22 10.98 17.52 13 12 13ZM12 18C6.48 18 2 15.98 2 13.5V16.5C2 18.98 6.48 21 12 21C17.52 21 22 18.98 22 16.5V13.5C22 15.98 17.52 18 12 18Z" />
                </svg>
              </div>
              <div className="relative z-10">
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-lg font-medium text-white">Redis Cache</h2>
                  <StatusDot status={health.checks?.redis?.status} />
                </div>
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-brand-text-secondary">Latency</span>
                    <span className="text-white font-mono">{health.checks?.redis?.latency_ms || 0} ms</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-brand-text-secondary">Pool Size</span>
                    <span className="text-white font-mono">{health.checks?.redis?.stats?.pool_size || 0}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-brand-text-secondary">Idle Conns</span>
                    <span className="text-white font-mono">{health.checks?.redis?.stats?.idle_conns || 0}</span>
                  </div>
                </div>
              </div>
            </GlassCard>

            {/* RabbitMQ */}
            <GlassCard className="relative overflow-hidden group">
              <div className="absolute top-0 right-0 p-4 opacity-5">
                <svg className="w-20 h-20 text-brand-accent-warning" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M20 18.66V10A8 8 0 004 10v8.66l-2 2V22h20v-1.34l-2-2zM12 2c2.76 0 5 2.24 5 5v3H7V7c0-2.76 2.24-5 5-5z" />
                </svg>
              </div>
              <div className="relative z-10">
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-lg font-medium text-white">RabbitMQ</h2>
                  <StatusDot status={health.checks?.rabbitmq?.status} />
                </div>
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-brand-text-secondary">Latency</span>
                    <span className="text-white font-mono">{health.checks?.rabbitmq?.latency_ms || 0} ms</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-brand-text-secondary">Protocol</span>
                    <span className="text-white font-mono">AMQP 0-9-1</span>
                  </div>
                </div>
              </div>
            </GlassCard>
          </div>

          <GlassCard className="mt-6">
            <h2 className="text-lg font-medium text-white mb-4">Server Details</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
              <div>
                <p className="text-xs text-brand-text-muted uppercase tracking-wider mb-1">Version</p>
                <p className="text-white font-medium">{health.version}</p>
              </div>
              <div>
                <p className="text-xs text-brand-text-muted uppercase tracking-wider mb-1">Environment</p>
                <p className="text-white font-medium capitalize">Production</p>
              </div>
            </div>
          </GlassCard>
        </>
      )}
    </div>
  );
}
