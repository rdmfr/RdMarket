const idrFormatter = new Intl.NumberFormat('id-ID', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

const idrWholeFormatter = new Intl.NumberFormat('id-ID', {
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

const percentFormatter = new Intl.NumberFormat('id-ID', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

const integerFormatter = new Intl.NumberFormat('id-ID');

/**
 * Formats a currency rate into IDR representation (e.g. 16.450,25 or 16.450)
 */
export function formatIDR(value: number | null | undefined, whole = false): string {
  if (value === null || value === undefined || isNaN(value)) {
    return '—';
  }
  return whole ? idrWholeFormatter.format(value) : idrFormatter.format(value);
}

/**
 * Formats a numeric change with explicit sign (+ / -)
 */
export function formatChange(value: number | null | undefined, whole = false): string {
  if (value === null || value === undefined || isNaN(value)) {
    return '—';
  }
  const formatted = whole ? idrWholeFormatter.format(Math.abs(value)) : idrFormatter.format(Math.abs(value));
  if (value > 0) {
    return `+${formatted}`;
  } else if (value < 0) {
    return `-${formatted}`;
  }
  return `0,00`;
}

/**
 * Formats a percentage change with explicit sign (+ / -)
 */
export function formatPercent(value: number | null | undefined): string {
  if (value === null || value === undefined || isNaN(value)) {
    return '—';
  }
  const formatted = percentFormatter.format(Math.abs(value));
  if (value > 0) {
    return `+${formatted}%`;
  } else if (value < 0) {
    return `-${formatted}%`;
  }
  return `0,00%`;
}

/**
 * Formats integer count
 */
export function formatCount(value: number | null | undefined): string {
  if (value === null || value === undefined || isNaN(value)) {
    return '—';
  }
  return integerFormatter.format(value);
}

/**
 * Formats a date/time to Asia/Jakarta (WIB) timezone: "29 Sep 2026, 14:30 WIB"
 */
export function formatTimestampWIB(dateOrIso: string | number | Date | null | undefined): string {
  if (!dateOrIso) return '—';
  const d = new Date(dateOrIso);
  if (isNaN(d.getTime())) return '—';

  // Format in Asia/Jakarta
  const options: Intl.DateTimeFormatOptions = {
    timeZone: 'Asia/Jakarta',
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  };

  const parts = new Intl.DateTimeFormat('id-ID', options).format(d);
  return `${parts} WIB`;
}

/**
 * Returns human-readable relative time string: "2m ago", "1h ago", etc.
 */
export function formatRelativeTime(dateOrIso: string | number | Date | null | undefined): string {
  if (!dateOrIso) return '';
  const d = new Date(dateOrIso);
  if (isNaN(d.getTime())) return '';

  const diffSec = Math.floor((Date.now() - d.getTime()) / 1000);
  if (diffSec < 0) return 'just now';
  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour}h ago`;
  const diffDay = Math.floor(diffHour / 24);
  return `${diffDay}d ago`;
}
