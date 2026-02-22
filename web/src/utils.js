export function formatBytes(bytes) {
  if (bytes == null || bytes <= 0 || !isFinite(bytes)) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const val = bytes / Math.pow(k, i);
  return `${val.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

export function formatSpeed(bytesPerSec) {
  if (!bytesPerSec || bytesPerSec <= 0 || !isFinite(bytesPerSec)) return '--';
  return `${formatBytes(bytesPerSec)}/s`;
}

export function formatEta(seconds) {
  if (seconds == null || seconds <= 0) return '--';
  // Use ceil for seconds so we never show "0s" while time remains.
  if (seconds < 60) return `${Math.ceil(seconds)}s`;
  if (seconds < 3600) {
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return s > 0 ? `${m}m ${s}s` : `${m}m`;
  }
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  return m > 0 ? `${h}h ${m}m` : `${h}h`;
}

export function parseImageRef(ref) {
  if (!ref || ref === '__pulling__') return { registry: '', name: '', tag: '', resolving: ref === '__pulling__' };
  let registry = '';
  let rest = ref;

  // Check if first path segment is a registry (contains . or :)
  const parts = ref.split('/');
  if (parts.length > 1 && (parts[0].includes('.') || parts[0].includes(':'))) {
    registry = parts[0];
    rest = parts.slice(1).join('/');
  }

  // Split tag/digest from the name (last colon or @)
  const atIdx = rest.indexOf('@');
  if (atIdx !== -1) {
    return { registry, name: rest.substring(0, atIdx), tag: rest.substring(atIdx) };
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
  if (!pull.totalKnown || pull.imageRef === '__pulling__') return 'unknown';
  return 'progress';
}
