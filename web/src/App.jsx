import React, { useState, useMemo } from 'react';
import FilterBar from './components/FilterBar';
import PullRow from './components/PullRow';
import { usePulls, useFilters } from './hooks';

export default function App() {
  const { pulls, connected } = usePulls();
  const { filters, setFilter, filterPulls } = useFilters();
  const [expandedIds, setExpandedIds] = useState(new Set());

  const filteredPulls = useMemo(() => filterPulls(pulls), [pulls, filterPulls]);

  const stats = useMemo(() => {
    const active = pulls.filter((p) => !p.completedAt && !p.error).length;
    const completed = pulls.filter((p) => p.completedAt).length;
    const errors = pulls.filter((p) => p.error).length;
    return { total: pulls.length, active, completed, errors };
  }, [pulls]);

  function toggleExpand(id) {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  return (
    <div className="app">
      <header className="header">
        <div className="header-left">
          <div className="header-logo">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
              <polyline points="7.5 4.21 12 6.81 16.5 4.21" />
              <polyline points="7.5 19.79 7.5 14.6 3 12" />
              <polyline points="21 12 16.5 14.6 16.5 19.79" />
              <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
              <line x1="12" y1="22.08" x2="12" y2="12" />
            </svg>
          </div>
          <h1>Pulltrace</h1>
        </div>
        <div className="header-right">
          <div className="connection-badge">
            <span className={`connection-dot ${connected ? 'connected' : ''}`} />
            {connected ? 'Live' : 'Disconnected'}
          </div>
        </div>
      </header>

      <div className="stats-bar">
        <div className="stat-card">
          <div className="stat-icon total">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
            </svg>
          </div>
          <div className="stat-info">
            <span className="stat-value">{stats.total}</span>
            <span className="stat-label">Total</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon active">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
            </svg>
          </div>
          <div className="stat-info">
            <span className="stat-value">{stats.active}</span>
            <span className="stat-label">Active</span>
          </div>
        </div>
        <div className="stat-card">
          <div className="stat-icon completed">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="20 6 9 17 4 12" />
            </svg>
          </div>
          <div className="stat-info">
            <span className="stat-value">{stats.completed}</span>
            <span className="stat-label">Completed</span>
          </div>
        </div>
        {stats.errors > 0 && (
          <div className="stat-card">
            <div className="stat-icon errors">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="10" />
                <line x1="15" y1="9" x2="9" y2="15" />
                <line x1="9" y1="9" x2="15" y2="15" />
              </svg>
            </div>
            <div className="stat-info">
              <span className="stat-value">{stats.errors}</span>
              <span className="stat-label">Errors</span>
            </div>
          </div>
        )}
      </div>

      <FilterBar filters={filters} setFilter={setFilter} />

      {filteredPulls.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
          </div>
          <h2>{pulls.length === 0 ? 'No active pulls' : 'No matching pulls'}</h2>
          <p>
            {pulls.length === 0
              ? 'Image pulls will appear here in real-time when detected by the cluster agents.'
              : 'Try adjusting your filters to find what you\'re looking for.'}
          </p>
        </div>
      ) : (
        <table className="pulls-table">
          <thead>
            <tr>
              <th style={{ width: 40 }}></th>
              <th>Image</th>
              <th>Node</th>
              <th>Progress</th>
              <th>Speed</th>
              <th>ETA</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {filteredPulls.map((pull) => (
              <PullRow
                key={pull.id}
                pull={pull}
                expanded={expandedIds.has(pull.id)}
                onToggle={() => toggleExpand(pull.id)}
              />
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
