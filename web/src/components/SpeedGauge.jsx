import React from 'react';

const R = 42;
const CX = 50;
const CY = 52;
const CIRC = 2 * Math.PI * R;      // 263.89
const TRACK = CIRC * 0.75;         // 197.92  (270° arc)

// logarithmic scale: 1 KB/s → 1 GB/s across 270°
const LOG_MIN = Math.log(1024);            // 1 KB/s
const LOG_MAX = Math.log(1024 ** 3);      // 1 GB/s

function speedToPct(bps) {
  if (!bps || bps < 1) return 0;
  const logVal = Math.log(Math.max(bps, 1024));
  return Math.min(1, Math.max(0, (logVal - LOG_MIN) / (LOG_MAX - LOG_MIN)));
}

function fmtGauge(bps) {
  if (!bps || bps <= 0) return { num: '—', unit: '' };
  if (bps >= 1024 ** 3) return { num: (bps / 1024 ** 3).toFixed(1), unit: 'GB/s' };
  if (bps >= 1024 ** 2) return { num: (bps / 1024 ** 2).toFixed(1), unit: 'MB/s' };
  if (bps >= 1024)      return { num: (bps / 1024).toFixed(1),      unit: 'KB/s' };
  return { num: String(bps), unit: 'B/s' };
}

// Ticks at 10 KB, 100 KB, 1 MB, 10 MB, 100 MB, 1 GB
const TICK_BPS = [10*1024, 100*1024, 1024**2, 10*1024**2, 100*1024**2, 1024**3];

function tickCoords(bps) {
  const pct = speedToPct(bps);
  const angle = (135 + pct * 270) * (Math.PI / 180); // rad, SVG clockwise from right
  return {
    x1: CX + (R - 7) * Math.cos(angle),
    y1: CY + (R - 7) * Math.sin(angle),
    x2: CX + (R - 2) * Math.cos(angle),
    y2: CY + (R - 2) * Math.sin(angle),
  };
}

export default function SpeedGauge({ bytesPerSec }) {
  const pct = speedToPct(bytesPerSec);
  const fillLen = pct * TRACK;
  const gapLen  = CIRC - fillLen;

  const active = bytesPerSec > 0;
  const color  = pct > 0.72 ? '#f59e0b' : '#22d3ee';
  const glow   = pct > 0.72 ? 'rgba(245,158,11,0.35)' : 'rgba(34,211,238,0.35)';

  const { num, unit } = fmtGauge(bytesPerSec);

  return (
    <div className="speed-gauge">
      <svg viewBox="0 0 100 104" width="96" height="96">
        {/* Tick marks */}
        {TICK_BPS.map((t) => {
          const tc = tickCoords(t);
          return (
            <line
              key={t}
              x1={tc.x1} y1={tc.y1} x2={tc.x2} y2={tc.y2}
              stroke="rgba(255,255,255,0.08)"
              strokeWidth="1"
              strokeLinecap="round"
            />
          );
        })}

        {/* Track arc */}
        <circle
          cx={CX} cy={CY} r={R}
          fill="none"
          stroke="rgba(255,255,255,0.04)"
          strokeWidth="7"
          strokeDasharray={`${TRACK} ${CIRC - TRACK}`}
          strokeLinecap="round"
          transform={`rotate(135 ${CX} ${CY})`}
        />

        {/* Value arc */}
        <circle
          cx={CX} cy={CY} r={R}
          fill="none"
          stroke={color}
          strokeWidth="7"
          strokeDasharray={`${fillLen} ${gapLen}`}
          strokeLinecap="round"
          transform={`rotate(135 ${CX} ${CY})`}
          style={{
            transition: 'stroke-dasharray 0.45s cubic-bezier(0.4,0,0.2,1), stroke 0.35s',
            filter: active ? `drop-shadow(0 0 6px ${glow})` : 'none',
            opacity: active ? 1 : 0,
          }}
        />

        {/* Center number */}
        <text
          x={CX} y={CY - 5}
          textAnchor="middle"
          dominantBaseline="middle"
          fontSize="17"
          fontWeight="700"
          fill={active ? '#dde4f8' : '#243045'}
          fontFamily="'IBM Plex Mono', monospace"
          style={{ transition: 'fill 0.3s' }}
        >
          {num}
        </text>

        {unit && (
          <text
            x={CX} y={CY + 13}
            textAnchor="middle"
            dominantBaseline="middle"
            fontSize="8"
            fontWeight="500"
            fill={active ? color : '#243045'}
            fontFamily="'IBM Plex Mono', monospace"
            style={{ transition: 'fill 0.3s' }}
          >
            {unit}
          </text>
        )}

        {/* Bottom label */}
        <text
          x={CX} y="100"
          textAnchor="middle"
          fontSize="7"
          fontWeight="600"
          fill="#243045"
          fontFamily="'IBM Plex Mono', monospace"
          letterSpacing="0.12em"
        >
          SPEED
        </text>
      </svg>
    </div>
  );
}
