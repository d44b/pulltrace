import React from 'react';
import ProgressBar from './ProgressBar';
import LayerDetail from './LayerDetail';
import { formatBytes, formatSpeed, formatEta, parseImageRef, getPullStatus } from '../utils';

export default function PullRow({ pull, layers, expanded, onToggle }) {
  const status = getPullStatus(pull);
  const img = parseImageRef(pull.imageRef);

  return (
    <>
      <tr className={expanded ? 'expanded' : ''} onClick={onToggle}>
        <td>
          <span className={`expand-arrow ${expanded ? 'open' : ''}`}>&#9654;</span>
        </td>
        <td>
          <div className="image-name">
            {img.registry && <span className="image-registry">{img.registry}/</span>}
            {img.name}
            {img.tag && <span className="image-tag">{img.tag}</span>}
          </div>
        </td>
        <td>
          <span className="node-name">{pull.nodeName || '--'}</span>
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
          <StatusBadge status={status} totalKnown={pull.totalKnown} />
        </td>
      </tr>
      {expanded && (
        <tr className="expand-row">
          <td colSpan={7}>
            <div className="expand-content">
              <LayerDetail layers={layers} status={status} />
              {pull.pods && pull.pods.length > 0 && (
                <div className="pods-section">
                  <div className="pods-title">Correlated Pods</div>
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

function StatusBadge({ status, totalKnown }) {
  if (status === 'error') return <span className="badge badge-error">Error</span>;
  if (status === 'completed') return <span className="badge badge-completed">Done</span>;
  if (!totalKnown) return <span className="badge badge-unknown">Unknown total</span>;
  return <span className="badge badge-progress">Pulling</span>;
}
