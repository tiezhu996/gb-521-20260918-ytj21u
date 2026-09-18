export interface ApiErrorBody {
  code: string;
  message: string;
  details?: unknown;
}

export interface ApiEnvelope<T> {
  data?: T;
  error?: ApiErrorBody;
  request_id: string;
  meta?: PageMeta;
}

export interface PageMeta {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

export interface PagedResult<T> {
  items: T[];
  meta: PageMeta;
}

export class ApiRequestError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly requestId: string,
    public readonly details?: unknown,
  ) {
    super(message);
    this.name = 'ApiRequestError';
  }
}
