export function formatBytes(bytes) {
  if (bytes == null || bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const val = bytes / Math.pow(k, i);
  return `${val.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

export function formatSpeed(bytesPerSec) {
  if (!bytesPerSec || bytesPerSec === 0) return '--';
  return `${formatBytes(bytesPerSec)}/s`;
}

export function formatEta(seconds) {
  if (seconds == null || seconds <= 0) return '--';
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600) {
    const m = Math.floor(seconds / 60);
    const s = Math.round(seconds % 60);
    return `${m}m ${s}s`;
  }
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  return `${h}h ${m}m`;
}

export function parseImageRef(ref) {
  if (!ref) return { registry: '', name: '', tag: '' };
  let registry = '';
  let rest = ref;

  // Check if first part is a registry (contains . or :)
  const parts = ref.split('/');
  if (parts.length > 1 && (parts[0].includes('.') || parts[0].includes(':'))) {
    registry = parts[0];
    rest = parts.slice(1).join('/');
  }

  const tagIdx = rest.lastIndexOf(':');
  if (tagIdx !== -1) {
    return { registry, name: rest.substring(0, tagIdx), tag: rest.substring(tagIdx) };
  }
  return { registry, name: rest, tag: '' };
}

export function getPullStatus(pull) {
  if (pull.error) return 'error';
  if (pull.completedAt) return 'completed';
  if (!pull.totalKnown) return 'unknown';
  return 'progress';
}
