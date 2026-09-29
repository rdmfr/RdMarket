import { describe, it } from 'node:test';
import assert from 'node:assert';
import { formatDisplayTime, formatDateOnlyUTC, formatISO } from './time';

describe('Time Module', () => {
  it('formats display time in Asia/Jakarta', () => {
    const iso = '2026-03-29T10:00:00Z';
    const display = formatDisplayTime(iso);
    assert.ok(display.includes('WIB'), 'Must display WIB');
    assert.strictEqual(formatDisplayTime('invalid-date'), '—');
  });

  it('formats date only in UTC', () => {
    const iso = '2026-03-29T23:59:59Z';
    assert.strictEqual(formatDateOnlyUTC(iso), '2026-03-29');
    assert.strictEqual(formatDateOnlyUTC('invalid'), '—');
  });

  it('formats ISO correctly', () => {
    const d = new Date('2026-03-29T12:00:00Z');
    assert.strictEqual(formatISO(d), '2026-03-29T12:00:00.000Z');
  });
});
