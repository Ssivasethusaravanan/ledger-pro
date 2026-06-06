import React from 'react';
import GlassCard from './GlassCard';
import Skeleton from './Skeleton';

export default function MetricCard({ title, value, icon: Icon, loading, format = 'number', trend }) {
  
  const formattedValue = () => {
    if (value === undefined || value === null) return '0';
    if (format === 'currency') {
      return (value / 100).toLocaleString('en-IN', { style: 'currency', currency: 'INR', maximumFractionDigits: 0 });
    }
    return value.toLocaleString();
  };

  return (
    <GlassCard className="relative overflow-hidden group">
      <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity duration-300 transform group-hover:scale-110">
        {Icon && <Icon className="w-16 h-16 text-brand-accent-primary" />}
      </div>
      
      <div className="relative z-10 flex flex-col h-full justify-between">
        <div className="flex items-center gap-2 text-brand-text-secondary font-medium">
          {Icon && <Icon className="w-5 h-5 text-brand-accent-primary/80" />}
          <h3>{title}</h3>
        </div>
        
        <div className="mt-4">
          {loading ? (
            <Skeleton className="h-10 w-24" />
          ) : (
            <div className="flex items-baseline gap-2">
              <span className="text-3xl font-bold text-white tracking-tight">{formattedValue()}</span>
              {trend && (
                <span className={`text-sm font-medium ${trend > 0 ? 'text-brand-accent-success' : 'text-brand-accent-danger'}`}>
                  {trend > 0 ? '+' : ''}{trend}%
                </span>
              )}
            </div>
          )}
        </div>
      </div>
    </GlassCard>
  );
}
