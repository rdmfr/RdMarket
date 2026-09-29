export const ErrorCodes = {
  INTERNAL_ERROR: 'INTERNAL_ERROR',
  BAD_REQUEST: 'BAD_REQUEST',
  UNAUTHORIZED: 'UNAUTHORIZED',
  FORBIDDEN: 'FORBIDDEN',
  NOT_FOUND: 'NOT_FOUND',
  UNPROCESSABLE_ENTITY: 'UNPROCESSABLE_ENTITY',
  RATE_LIMITED: 'RATE_LIMITED',
  SERVICE_UNAVAILABLE: 'SERVICE_UNAVAILABLE',
  INVALID_CREDENTIALS: 'INVALID_CREDENTIALS',
  CSRF_INVALID: 'CSRF_INVALID',
  INSUFFICIENT_DATA: 'INSUFFICIENT_DATA',
  SIMULATION_FORBIDDEN: 'SIMULATION_FORBIDDEN',
} as const;

export type ErrorCode = (typeof ErrorCodes)[keyof typeof ErrorCodes];

export interface ApiErrorPayload {
  code: ErrorCode | string;
  message: string;
}

export class ApiError extends Error {
  public readonly code: ErrorCode | string;

  constructor(payload: ApiErrorPayload) {
    super(payload.message);
    this.name = 'ApiError';
    this.code = payload.code;
  }
}
