export const DEFAULT_TIMEZONE = 'Asia/Jakarta';

/**
 * Parses an ISO string or Date and formats it in Asia/Jakarta (WIB) by default.
 */
export function formatDisplayTime(
  input: string | Date | number,
  timeZone: string = DEFAULT_TIMEZONE
): string {
  const date = typeof input === 'string' || typeof input === 'number' ? new Date(input) : input;
  if (isNaN(date.getTime())) {
    return '—';
  }

  const formatter = new Intl.DateTimeFormat('en-GB', {
    timeZone,
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });

  return `${formatter.format(date)} WIB`;
}

/**
 * Formats date-only string (YYYY-MM-DD) in UTC.
 */
export function formatDateOnlyUTC(input: string | Date | number): string {
  const date = typeof input === 'string' || typeof input === 'number' ? new Date(input) : input;
  if (isNaN(date.getTime())) {
    return '—';
  }
  return date.toISOString().split('T')[0];
}

/**
 * Formats a timestamp into an ISO string in UTC.
 */
export function formatISO(input: string | Date | number): string {
  const date = typeof input === 'string' || typeof input === 'number' ? new Date(input) : input;
  if (isNaN(date.getTime())) {
    return '';
  }
  return date.toISOString();
}
