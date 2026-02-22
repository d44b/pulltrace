import React from 'react';
import ProgressBar from './ProgressBar';
import LayerDetail from './LayerDetail';
import { formatBytes, formatSpeed, formatEta, parseImageRef, getPullStatus } from '../utils';

export default function PullRow({ pull, expanded, onToggle }) {
  const status = getPullStatus(pull);
  const img = parseImageRef(pull.imageRef);
  const isResolving = pull.imageRef === '__pulling__';

  return (
    <>
      <tr className={expanded ? 'expanded' : ''} onClick={onToggle}>
        <td>
          <span className={`expand-arrow ${expanded ? 'open' : ''}`}>
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="9 18 15 12 9 6" />
            </svg>
          </span>
        </td>
        <td>
          {isResolving ? (
            <div className="image-name">
              <span className="image-resolving">Resolving image&hellip;</span>
            </div>
          ) : (
            <div className="image-name">
              {img.registry && <span className="image-registry">{img.registry}/</span>}
              <span className="image-repo">{img.name}</span>
              {img.tag && <span className="image-tag">{img.tag}</span>}
            </div>
          )}
        </td>
        <td>
          {pull.nodeName ? (
            <span className="node-name">{pull.nodeName}</span>
          ) : (
            <span className="eta">--</span>
          )}
        </td>
        <td>
          <div className="progress-bar-container">
            <ProgressBar percent={pull.percent} status={status} />
            <div className="progress-text">
              <span>
                {formatBytes(pull.downloadedBytes)} / {pull.totalKnown ? formatBytes(pull.totalBytes) : '?'}
              </span>
              <span>
                {pull.layersDone}/{pull.layerCount} layers
              </span>
            </div>
          </div>
        </td>
        <td>
          <span className="speed">{formatSpeed(pull.bytesPerSec)}</span>
        </td>
        <td>
          <span className="eta">{status === 'completed' ? '--' : formatEta(pull.etaSeconds)}</span>
        </td>
        <td>
          <StatusBadge status={status} totalKnown={pull.totalKnown} isResolving={isResolving} />
        </td>
      </tr>
      {expanded && (
        <tr className="expand-row">
          <td colSpan={7}>
            <div className="expand-content">
              <LayerDetail layers={pull.layers || []} status={status} />
              {pull.pods && pull.pods.length > 0 && (
                <div className="pods-section">
                  <div className="pods-title">Waiting Pods</div>
                  <div className="pod-list">
                    {pull.pods.map((pod, i) => (
                      <span className="pod-chip" key={i}>
                        <span className="pod-ns">{pod.namespace}/</span>
                        <span className="pod-name">{pod.podName}</span>
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

function StatusBadge({ status, totalKnown, isResolving }) {
  if (status === 'error') {
    return <span className="badge badge-error"><span className="badge-dot" /> Error</span>;
  }
  if (status === 'completed') {
    return <span className="badge badge-completed"><span className="badge-dot" /> Done</span>;
  }
  if (isResolving || !totalKnown) {
    return <span className="badge badge-unknown"><span className="badge-dot" /> Resolving</span>;
  }
  return <span className="badge badge-progress"><span className="badge-dot" /> Pulling</span>;
}
