import React from 'react';
import SpeedGauge from './SpeedGauge';
import LayerDetail from './LayerDetail';
import { formatBytes, formatSpeed, formatEta, parseImageRef, getPullStatus } from '../utils';

const ChevronIcon = () => (
  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="9 18 15 12 9 6"/>
  </svg>
);

const NodeIcon = () => (
  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/>
    <line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/>
  </svg>
);

export default function PullRow({ pull, expanded, onToggle }) {
  const status = getPullStatus(pull);
  const img = parseImageRef(pull.imageRef);
  const isResolving = pull.imageRef === '__pulling__';
  const isActive = status === 'progress' || status === 'unknown';
  const pct = Math.min(100, Math.max(0, pull.percent || 0));

  // Layer strips: up to 20 shown
  const layers = pull.layers || [];

  return (
    <div className={`pull-card status-${status}`}>
      {/* ── Header ── */}
      <div className="card-header" onClick={onToggle} role="button" tabIndex={0}
        onKeyDown={(e) => e.key === 'Enter' && onToggle()}>
        <button className={`expand-btn${expanded ? ' open' : ''}`} tabIndex={-1} aria-hidden>
          <ChevronIcon />
        </button>

        <div className="image-block">
          {isResolving ? (
            <span className="img-resolving">Resolving image…</span>
          ) : (
            <div className="image-name-row">
              {img.registry && <span className="img-registry">{img.registry}/</span>}
              <span className="img-repo">{img.name}</span>
              {img.tag && <span className="img-tag">{img.tag}</span>}
            </div>
          )}
        </div>

        <div className="card-right">
          {pull.nodeName && (
            <span className="node-chip">
              <NodeIcon /> {pull.nodeName}
            </span>
          )}
          <StatusBadge status={status} isResolving={isResolving} totalKnown={pull.totalKnown} />
        </div>
      </div>

      {/* ── Body: active pulls get gauge + progress ── */}
      {isActive && (
        <div className="card-body">
          <div className="progress-col">
            {/* Progress bar */}
            <div className="progress-track">
              <div
                className={`progress-fill status-${status}`}
                style={{ width: `${pct}%` }}
              />
            </div>

            {/* Bytes + layers meta */}
            <div className="progress-meta">
              <span>
                {formatBytes(pull.downloadedBytes)}&nbsp;/&nbsp;
                {pull.totalKnown ? formatBytes(pull.totalBytes) : '?'}
              </span>
              <span>{pull.layersDone || 0} / {pull.layerCount || 0} layers</span>
            </div>

            {/* Layer strips */}
            {layers.length > 0 && (
              <div className="layer-strips">
                {layers.slice(0, 32).map((l, i) => {
                  const done = l.percent >= 100;
                  const active = !done && (l.bytesPerSec > 0 || l.downloadedBytes > 0);
                  return (
                    <div
                      key={l.digest || i}
                      className={`layer-strip${done ? ' done' : active ? ' active' : ''}`}
                    />
                  );
                })}
              </div>
            )}
          </div>

          {/* Speed gauge */}
          <div className="gauge-col">
            <SpeedGauge bytesPerSec={pull.bytesPerSec} />
            <div className="gauge-eta">
              {pull.etaSeconds > 0 ? (
                <><span className="gauge-eta-label">ETA </span>{formatEta(pull.etaSeconds)}</>
              ) : '—'}
            </div>
          </div>
        </div>
      )}

      {/* ── Compact summary for completed/error ── */}
      {!isActive && (
        <div className="card-summary" onClick={onToggle} style={{ cursor: 'pointer' }}>
          <span className="summary-size">
            {formatBytes(pull.downloadedBytes || pull.totalBytes)}
          </span>
          <span style={{ color: 'var(--t-muted)' }}>
            {pull.layerCount ? `${pull.layerCount} layers` : ''}
          </span>
        </div>
      )}

      {/* ── Waiting pods ── */}
      {pull.pods && pull.pods.length > 0 && (
        <div className="card-pods">
          <span className="pods-label">Pods</span>
          {pull.pods.map((pod, i) => (
            <span className="pod-chip" key={i}>
              <span className="pod-ns">{pod.namespace}/</span>
              <span className="pod-name">{pod.podName}</span>
            </span>
          ))}
        </div>
      )}

      {/* ── Expanded layer detail ── */}
      {expanded && (
        <div className="card-layers">
          <LayerDetail layers={layers} status={status} />
        </div>
      )}
    </div>
  );
}

function StatusBadge({ status, isResolving, totalKnown }) {
  if (status === 'error')     return <span className="badge badge-error"><span className="badge-dot"/>Error</span>;
  if (status === 'completed') return <span className="badge badge-completed"><span className="badge-dot"/>Done</span>;
  if (isResolving || !totalKnown) return <span className="badge badge-unknown"><span className="badge-dot"/>Resolving</span>;
  return <span className="badge badge-progress"><span className="badge-dot"/>Pulling</span>;
}
