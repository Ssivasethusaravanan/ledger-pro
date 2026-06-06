import React from 'react';
import { classNames } from '@/lib/utils';

export default function Skeleton({ className, ...props }) {
  return (
    <div
      className={classNames(
        "animate-shimmer bg-[linear-gradient(90deg,rgba(255,255,255,0.03)_25%,rgba(255,255,255,0.08)_50%,rgba(255,255,255,0.03)_75%)] bg-[length:200%_100%] rounded-md",
        className
      )}
      {...props}
    />
  );
}
