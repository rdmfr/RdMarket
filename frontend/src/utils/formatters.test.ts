import { describe, it } from 'node:test';
import assert from 'node:assert';
import { formatIDR, formatChange, formatPercent, formatTimestampWIB } from './formatters';

describe('Formatters', () => {
  it('formats IDR currency correctly', () => {
    assert.strictEqual(formatIDR(16200.5), '16.200,50');
    assert.strictEqual(formatIDR(16200, true), '16.200');
    assert.strictEqual(formatIDR(null), '—');
  });

  it('formats changes with explicit sign', () => {
    assert.strictEqual(formatChange(50.25), '+50,25');
    assert.strictEqual(formatChange(-30.0), '-30,00');
    assert.strictEqual(formatChange(0), '0,00');
    assert.strictEqual(formatChange(null), '—');
  });

  it('formats percentage with explicit sign', () => {
    assert.strictEqual(formatPercent(1.25), '+1,25%');
    assert.strictEqual(formatPercent(-0.5), '-0,50%');
    assert.strictEqual(formatPercent(0), '0,00%');
    assert.strictEqual(formatPercent(null), '—');
  });

  it('formats timestamp in Asia/Jakarta (WIB)', () => {
    // 2026-03-29 00:00:00 UTC should be 07:00 WIB
    const iso = '2026-03-29T00:00:00Z';
    const formatted = formatTimestampWIB(iso);
    assert.ok(formatted.includes('WIB'), 'Must include WIB timezone');
    assert.ok(formatted.includes('07:00') || formatted.includes('07.00'), 'Must be 07:00 in WIB');
  });
});
