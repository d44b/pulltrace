import React, { useState, useMemo } from 'react';
import FilterBar from './components/FilterBar';
import PullRow from './components/PullRow';
import { usePulls, useFilters } from './hooks';

const LogoIcon = () => (
  <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
    <path d="M12 2L2 7l10 5 10-5-10-5z"/>
    <path d="M2 17l10 5 10-5"/>
    <path d="M2 12l10 5 10-5"/>
  </svg>
);

export default function App() {
  const { pulls, connected } = usePulls();
  const { filters, setFilter, filterPulls } = useFilters();
  const [expandedIds, setExpandedIds] = useState(new Set());

  const filteredPulls = useMemo(() => filterPulls(pulls), [pulls, filterPulls]);

  const stats = useMemo(() => {
    const active    = pulls.filter((p) => !p.completedAt && !p.error).length;
    const completed = pulls.filter((p) =>  p.completedAt).length;
    const errors    = pulls.filter((p) =>  p.error).length;
    return { total: pulls.length, active, completed, errors };
  }, [pulls]);

  function toggleExpand(id) {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  }

  return (
    <div className="app">
      {/* ── Header ── */}
      <header className="header">
        <div className="header-left">
          <div className="header-logo"><LogoIcon /></div>
          <span className="header-wordmark">PULL<span>TRACE</span></span>
        </div>
        <div className="header-right">
          <div className={`live-badge${connected ? ' connected' : ''}`}>
            <span className="live-dot" />
            {connected ? 'LIVE' : 'OFFLINE'}
          </div>
        </div>
      </header>

      {/* ── Stats ── */}
      <div className="stats-row">
        <div className="stat-tile total">
          <div className="stat-num">{stats.total}</div>
          <div className="stat-label">Total</div>
        </div>
        <div className="stat-tile active">
          <div className="stat-num">{stats.active}</div>
          <div className="stat-label">Active</div>
        </div>
        <div className="stat-tile done">
          <div className="stat-num">{stats.completed}</div>
          <div className="stat-label">Done</div>
        </div>
        <div className="stat-tile errors">
          <div className="stat-num">{stats.errors}</div>
          <div className="stat-label">Errors</div>
        </div>
      </div>

      {/* ── Filters ── */}
      <FilterBar filters={filters} setFilter={setFilter} />

      {/* ── Pull list ── */}
      {filteredPulls.length === 0 ? (
        <div className="empty-state">
          <div className="empty-radar">
            <div className="empty-radar-ring" />
            <div className="empty-radar-ring-inner" />
            <div className="empty-radar-dot" />
            <div className="empty-radar-sweep" />
          </div>
          <h2>{pulls.length === 0 ? 'Scanning for pulls…' : 'No matching pulls'}</h2>
          <p>
            {pulls.length === 0
              ? 'Image pulls will appear here in real-time when detected by the cluster agents.'
              : 'Try adjusting your filters.'}
          </p>
        </div>
      ) : (
        <div className="pulls-list">
          {filteredPulls.map((pull) => (
            <PullRow
              key={pull.id}
              pull={pull}
              expanded={expandedIds.has(pull.id)}
              onToggle={() => toggleExpand(pull.id)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
