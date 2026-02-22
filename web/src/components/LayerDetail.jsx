import React from 'react';
import ProgressBar from './ProgressBar';
import { formatBytes, formatSpeed } from '../utils';

export default function LayerDetail({ layers, status }) {
  if (!layers || layers.length === 0) {
    return (
      <div className="layers-title" style={{ color: 'var(--text-muted)' }}>
        No layer data yet
      </div>
    );
  }

  return (
    <div>
      <div className="layers-title">Layers</div>
      <div className="layer-list">
        {layers.map((layer) => {
          const digest = layer.digest || '';
          const shortDigest = digest.startsWith('sha256:') ? digest.substring(7, 19) : digest.substring(0, 12);
          const layerStatus = layer.percent >= 100 ? 'completed' : layer.totalKnown ? status : 'unknown';

          return (
            <div className="layer-item" key={digest}>
              <span className="layer-digest">{shortDigest}</span>
              <div className="layer-progress">
                <ProgressBar percent={layer.percent} status={layerStatus} height="4px" />
              </div>
              <span className="layer-size">
                {formatBytes(layer.downloadedBytes)} / {layer.totalKnown ? formatBytes(layer.totalBytes) : '?'}
              </span>
              <span className="speed" style={{ minWidth: 80, textAlign: 'right' }}>
                {formatSpeed(layer.bytesPerSec)}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
