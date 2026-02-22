import React from 'react';

export default function ProgressBar({ percent, status, height }) {
  const cls =
    status === 'completed' ? 'status-completed' :
    status === 'error'     ? 'status-error'     :
    status === 'unknown'   ? 'status-unknown'   :
    'status-progress';

  return (
    <div className="progress-track" style={height ? { height } : undefined}>
      <div
        className={`progress-fill ${cls}`}
        style={{ width: `${Math.min(100, Math.max(0, percent || 0))}%` }}
      />
    </div>
  );
}
