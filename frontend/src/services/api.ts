import ky, { type KyResponse } from 'ky';
import { z } from 'zod';
import { ApiError, type ErrorCode, ErrorCodes } from '../types/errors';

function getCookie(name: string): string | null {
  if (typeof document === 'undefined') return null;
  const match = document.cookie.match(new RegExp('(^|;\\s*)(' + name + ')=([^;]*)'));
  return match ? decodeURIComponent(match[3]) : null;
}

export const apiClient = ky.create({
  prefix: '/api/v1',
  timeout: 10000,
  credentials: 'include',
  retry: {
    limit: 2,
    methods: ['get'],
    statusCodes: [408, 500, 502, 503, 504],
  },
  headers: {
    Accept: 'application/json',
  },
  hooks: {
    beforeRequest: [
      ({ request }) => {
        // Automatically inject CSRF token on mutating requests if present in cookie
        const method = request.method.toUpperCase();
        if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
          const csrf = getCookie('rdm_csrf');
          if (csrf) {
            request.headers.set('X-CSRF-Token', csrf);
          }
        }
      },
    ],
    afterResponse: [
      async ({ response }: { response: KyResponse }) => {
        if (!response.ok) {
          let code: ErrorCode | string = ErrorCodes.INTERNAL_ERROR;
          let message = `HTTP Error ${response.status}: ${response.statusText}`;

          try {
            const body = await response.clone().json();
            if (body && body.error) {
              code = body.error.code || code;
              message = body.error.message || message;
            }
          } catch {
            // response was not JSON
          }

          throw new ApiError({ code, message });
        }
      },
    ],
  },
});

/**
 * Zod validation helper that validates data against schema, logs detailed failures,
 * and surfaces a typed ApiError.
 */
export function validateResponse<T>(schema: z.ZodType<T>, rawData: unknown): T {
  const result = schema.safeParse(rawData);
  if (!result.success) {
    console.error('[API Validation Failure]', result.error.format());
    throw new ApiError({
      code: ErrorCodes.UNPROCESSABLE_ENTITY,
      message: `Data contract validation failed: ${result.error.issues.map((i) => `${i.path.join('.')}: ${i.message}`).join(', ')}`,
    });
  }
  return result.data;
}
