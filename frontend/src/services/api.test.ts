import { describe, it } from 'node:test';
import assert from 'node:assert';
import { z } from 'zod';
import { validateResponse } from './api';
import { ApiError, ErrorCodes } from '../types/errors';

describe('API Service & Validation', () => {
  it('validates response matching Zod schema', () => {
    const RateSchema = z.object({
      pair: z.string(),
      rate: z.number(),
    });

    const validData = { pair: 'USD/IDR', rate: 16250 };
    const parsed = validateResponse(RateSchema, validData);
    assert.deepStrictEqual(parsed, validData);
  });

  it('throws ApiError with UNPROCESSABLE_ENTITY on schema mismatch', () => {
    const RateSchema = z.object({
      pair: z.string(),
      rate: z.number(),
    });

    const invalidData = { pair: 'USD/IDR', rate: 'not-a-number' };
    assert.throws(
      () => validateResponse(RateSchema, invalidData),
      (err: unknown) => {
        return (
          err instanceof ApiError &&
          err.code === ErrorCodes.UNPROCESSABLE_ENTITY &&
          err.message.includes('rate')
        );
      }
    );
  });
});
