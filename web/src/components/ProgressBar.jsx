import React from 'react';

export default function ProgressBar({ percent, status, height }) {
  const cls =
    status === 'completed'
      ? 'completed'
      : status === 'error'
        ? 'error'
        : status === 'unknown'
          ? 'unknown'
          : '';

  return (
    <div className="progress-bar" style={height ? { height } : undefined}>
      <div
        className={`progress-fill ${cls}`}
        style={{ width: `${Math.min(100, Math.max(0, percent || 0))}%` }}
      />
    </div>
  );
}
