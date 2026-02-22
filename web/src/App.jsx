import React, { useState, useMemo } from 'react';
import FilterBar from './components/FilterBar';
import PullRow from './components/PullRow';
import { usePulls, useFilters } from './hooks';
import { getPullStatus } from './utils';

export default function App() {
  const { pulls, layers, connected } = usePulls();
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
          <span className="header-logo" role="img" aria-label="package">
            &#x1F4E6;
          </span>
          <h1>Pulltrace</h1>
        </div>
        <div className="connection-badge">
          <span className={`connection-dot ${connected ? 'connected' : ''}`} />
          {connected ? 'Live' : 'Disconnected'}
        </div>
      </header>

      <FilterBar filters={filters} setFilter={setFilter} />

      <div className="stats-bar">
        <div className="stat">
          Total: <span className="stat-value">{stats.total}</span>
        </div>
        <div className="stat">
          Active: <span className="stat-value">{stats.active}</span>
        </div>
        <div className="stat">
          Completed: <span className="stat-value">{stats.completed}</span>
        </div>
        {stats.errors > 0 && (
          <div className="stat">
            Errors: <span className="stat-value" style={{ color: 'var(--accent-red)' }}>{stats.errors}</span>
          </div>
        )}
      </div>

      {filteredPulls.length === 0 ? (
        <div className="empty-state">
          <div className="empty-state-icon">&#x1F50D;</div>
          <h2>{pulls.length === 0 ? 'No active pulls' : 'No matching pulls'}</h2>
          <p>
            {pulls.length === 0
              ? 'Image pulls will appear here when detected by the agent.'
              : 'Try adjusting your filters.'}
          </p>
        </div>
      ) : (
        <table className="pulls-table">
          <thead>
            <tr>
              <th style={{ width: 28 }}></th>
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
                layers={layers[pull.id] || {}}
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
