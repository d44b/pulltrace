import React from 'react';
import { formatBytes, formatSpeed } from '../utils';

export default function LayerDetail({ layers }) {
  if (!layers || layers.length === 0) {
    return <p style={{ color: 'var(--t-muted)', fontSize: 12, fontFamily: 'var(--mono)' }}>No layer data yet</p>;
  }

  return (
    <div>
      <div className="layers-heading">Layers ({layers.length})</div>
      <div className="layer-rows">
        {layers.map((layer) => {
          const digest = layer.digest || '';
          const short = digest.startsWith('sha256:') ? digest.slice(7, 19) : digest.slice(0, 12);
          const done = layer.percent >= 100;
          const pct = Math.min(100, Math.max(0, layer.percent || 0));
          const speed = layer.bytesPerSec > 0 ? formatSpeed(layer.bytesPerSec) : null;

          return (
            <div className="layer-row" key={digest}>
              <span className="layer-hash">{short || '············'}</span>
              <div className="layer-bar-wrap">
                <div className="layer-bar">
                  <div
                    className={`layer-bar-fill${done ? ' done' : ''}`}
                    style={{ width: `${pct}%`, transition: 'width 0.4s ease' }}
                  />
                </div>
              </div>
              <span className="layer-bytes">
                {formatBytes(layer.downloadedBytes)} / {layer.totalKnown ? formatBytes(layer.totalBytes) : '?'}
              </span>
              {speed && <span className="layer-speed">{speed}</span>}
            </div>
          );
        })}
      </div>
    </div>
  );
}
