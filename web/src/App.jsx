import React, { useState, useMemo } from 'react';
import FilterBar from './components/FilterBar';
import PullRow from './components/PullRow';
import { usePulls, useFilters } from './hooks';

const LogoIcon = ({ size = 20 }) => (
  <svg width={size} height={size} viewBox="0 0 256 256" fill="none" strokeLinecap="round" strokeLinejoin="round">
    <defs>
      <linearGradient id="logo-ring" x1="0%" y1="0%" x2="100%" y2="100%">
        <stop offset="0%" stopColor="#58a6ff"/>
        <stop offset="100%" stopColor="#3fb950"/>
      </linearGradient>
    </defs>
    <circle cx="128" cy="128" r="100" stroke="#58a6ff" strokeWidth="14" opacity="0.20"/>
    <path d="M128 28a100 100 0 0 1 100 100" stroke="url(#logo-ring)" strokeWidth="14"/>
    <path d="M80 116l48-24 48 24-48 24-48-24Z" stroke="#58a6ff" strokeWidth="12"/>
    <path d="M80 148l48 24 48-24" stroke="#3fb950" strokeWidth="12"/>
  </svg>
);

function splitSpeed(bps) {
  if (!bps || bps <= 0 || !isFinite(bps)) return { num: '0', unit: 'B/s' };
  const units = ['B', 'KB', 'MB', 'GB'];
  const k = 1024;
  const i = Math.min(3, Math.floor(Math.log(bps) / Math.log(k)));
  return { num: (bps / Math.pow(k, i)).toFixed(i > 0 ? 1 : 0), unit: `${units[i]}/s` };
}

function speedToBarPct(bps) {
  if (!bps || bps <= 0) return 0;
  // Log scale: 1 KB/s (log=3) → 0%, 1 GB/s (log=9) → ~96%
  return Math.max(1, Math.min(96, ((Math.log10(bps) - 3) / 6) * 100));
}

export default function App() {
  const { pulls, connected } = usePulls();
  const { filters, setFilter, filterPulls } = useFilters();
  const [expandedIds, setExpandedIds] = useState(new Set());

  const filteredPulls = useMemo(() => {
    const list = filterPulls(pulls);
    return [...list].sort((a, b) => {
      const aActive = !a.completedAt && !a.error;
      const bActive = !b.completedAt && !b.error;
      if (aActive !== bActive) return aActive ? -1 : 1;
      return (b.startedAt || '').localeCompare(a.startedAt || '');
    });
  }, [pulls, filterPulls]);

  const stats = useMemo(() => {
    const active    = pulls.filter((p) => !p.completedAt && !p.error).length;
    const completed = pulls.filter((p) =>  p.completedAt).length;
    const errors    = pulls.filter((p) =>  p.error).length;
    return { total: pulls.length, active, completed, errors };
  }, [pulls]);

  const totalSpeed = useMemo(
    () => pulls.filter(p => !p.completedAt && !p.error).reduce((s, p) => s + (p.bytesPerSec || 0), 0),
    [pulls]
  );

  const speedBarPct = useMemo(() => speedToBarPct(totalSpeed), [totalSpeed]);
  const heroSpeed = useMemo(() => splitSpeed(totalSpeed), [totalSpeed]);

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
          <div className="header-logo"><LogoIcon size={55} /></div>
          <span className="header-wordmark">Pulltrace</span>
        </div>
        <div className={`live-badge${connected ? ' connected' : ''}`}>
          <span className="live-dot" />
          {connected ? 'LIVE' : 'OFFLINE'}
        </div>
      </header>

      {/* ── Speed panel ── */}
      <div className="speed-panel">
        <div className="speed-hero">
          <div className="speed-hero-label">Total Throughput</div>
          <div className="speed-hero-value">
            <span className={`speed-hero-num${totalSpeed <= 0 ? ' zero' : ''}`}>
              {heroSpeed.num}
            </span>
            <span className="speed-hero-unit">{heroSpeed.unit}</span>
          </div>
          <div className="speed-bar-wrap">
            <div className="speed-bar-fill" style={{ width: `${speedBarPct}%` }} />
          </div>
        </div>

        <div className="speed-stats">
          <div className="speed-stat total">
            <div className="speed-stat-num">{stats.total}</div>
            <div className="speed-stat-label">Total</div>
          </div>
          <div className="speed-stat active">
            <div className="speed-stat-num">{stats.active}</div>
            <div className="speed-stat-label">Active</div>
          </div>
          <div className="speed-stat done">
            <div className="speed-stat-num">{stats.completed}</div>
            <div className="speed-stat-label">Done</div>
          </div>
          <div className="speed-stat errors">
            <div className="speed-stat-num">{stats.errors}</div>
            <div className="speed-stat-label">Errors</div>
          </div>
        </div>
      </div>

      {/* ── Filters ── */}
      <FilterBar filters={filters} setFilter={setFilter} />

      {/* ── Pull list ── */}
      {filteredPulls.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon"><LogoIcon /></div>
          <h2>{pulls.length === 0 ? 'Scanning for pulls…' : 'No matching pulls'}</h2>
          <p>
            {pulls.length === 0
              ? 'Image pulls will appear here in real-time when detected by the cluster agents.'
              : 'Try adjusting your filters.'}
          </p>
        </div>
      ) : (
        <div className="pulls-table">
          <div className="table-header">
            <div />
            <div className="th">Image</div>
            <div className="th">Node</div>
            <div className="th">Progress</div>
            <div className="th">%</div>
            <div className="th">Speed</div>
            <div className="th">Size</div>
            <div className="th">ETA</div>
            <div />
          </div>
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
