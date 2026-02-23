import React, { useRef } from 'react';
import LayerDetail from './LayerDetail';
import { formatBytes, formatEta, parseImageRef, getPullStatus } from '../utils';

const ChevronIcon = () => (
  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="9 18 15 12 9 6"/>
  </svg>
);

function formatSpeed(bps) {
  if (!bps || bps <= 0 || !isFinite(bps)) return null;
  const units = ['B', 'KB', 'MB', 'GB'];
  const k = 1024;
  const i = Math.min(3, Math.floor(Math.log(bps) / Math.log(k)));
  return `${(bps / Math.pow(k, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}/s`;
}

export default function PullRow({ pull, expanded, onToggle }) {
  const status = getPullStatus(pull);
  const img = parseImageRef(pull.imageRef);
  const isResolving = pull.imageRef === '__pulling__';
  const pct = Math.min(100, Math.max(0, pull.percent || 0));
  const layers = pull.layers || [];

  // High-water mark: bar never goes backwards (prevents flicker when server
  // discovers new layers and totalBytes grows, temporarily dropping percent).
  const maxPctRef = useRef(0);
  const prevStartRef = useRef(pull.startedAt);
  if (prevStartRef.current !== pull.startedAt) {
    prevStartRef.current = pull.startedAt;
    maxPctRef.current = 0;
  }
  maxPctRef.current = Math.max(maxPctRef.current, pct);

  // Use high-water mark for display so bar + text only ever advance forward.
  // For errors, show the real value (failed at X%).
  const displayPct = status === 'error' ? pct : maxPctRef.current;

  // Treat pct=100 as visually complete even if completedAt hasn't arrived yet.
  // This eliminates the race-condition where the server sends percent=100 but
  // still has non-zero bytesPerSec and no completedAt.
  const effectiveStatus = (displayPct >= 100 && status !== 'error') ? 'completed' : status;
  const showLive = (effectiveStatus === 'progress' || effectiveStatus === 'unknown') && displayPct < 100;
  const speed = showLive ? formatSpeed(pull.bytesPerSec) : null;

  // Elapsed time: use completedAt if available, otherwise use current time for
  // pulls that reached 100% (so the field is never empty for finished pulls).
  const elapsedSec = (() => {
    if (!pull.startedAt) return null;
    const end = pull.completedAt ? new Date(pull.completedAt) : (pct >= 100 ? new Date() : null);
    if (!end) return null;
    return (end - new Date(pull.startedAt)) / 1000;
  })();
  const elapsed = !showLive && elapsedSec != null ? formatEta(elapsedSec) : null;

  return (
    <>
      <div className="pull-row" onClick={onToggle} role="button" tabIndex={0}
        onKeyDown={(e) => e.key === 'Enter' && onToggle()}>

        {/* Status dot — uses effectiveStatus so 100% flips green immediately */}
        <div className={`row-dot status-${effectiveStatus}`} />

        {/* Image name */}
        <div className="row-image">
          <div className="row-image-text">
            {isResolving ? (
              <span className="img-resolving">Resolving…</span>
            ) : (
              <>
                {img.registry && <span className="img-registry">{img.registry}/</span>}
                <span className="img-repo">{img.name}</span>
                {img.tag && <span className="img-tag">{img.tag}</span>}
              </>
            )}
          </div>
        </div>

        {/* Node */}
        <div className="row-node">{pull.nodeName || '—'}</div>

        {/* Progress bar — uses effectiveStatus for color, displayPct for width */}
        <div className="row-bar-track">
          <div className={`row-bar-fill status-${effectiveStatus}`} style={{ width: `${displayPct}%` }} />
        </div>

        {/* Percent */}
        <div className="row-pct">
          {displayPct >= 100 ? '100%'
            : displayPct > 0 ? `${Math.round(displayPct)}%`
            : '—'}
        </div>

        {/* Speed */}
        <div className={`row-speed${!speed ? ' zero' : ''}`}>
          {speed || '—'}
        </div>

        {/* Size */}
        <div className="row-size">
          {formatBytes(pull.totalKnown ? pull.totalBytes : (pull.downloadedBytes || 0))}
        </div>

        {/* ETA while active, elapsed once done */}
        <div className={`row-eta${showLive ? ' active' : ' done'}`}>
          {showLive
            ? (pull.etaSeconds > 0 ? formatEta(pull.etaSeconds) : '—')
            : (elapsed || '—')}
        </div>

        {/* Expand chevron */}
        <div className={`row-chevron${expanded ? ' open' : ''}`}>
          <ChevronIcon />
        </div>
      </div>

      {/* Expanded detail */}
      {expanded && (
        <div className="row-detail">
          {pull.pods && pull.pods.length > 0 && (
            <div className="detail-pods">
              <span className="detail-pods-label">Pods</span>
              {pull.pods.map((pod, i) => (
                <span className="pod-chip" key={i}>
                  <span className="pod-ns">{pod.namespace}/</span>
                  <span className="pod-name">{pod.podName}</span>
                </span>
              ))}
            </div>
          )}
          <LayerDetail layers={layers} />
        </div>
      )}
    </>
  );
}
