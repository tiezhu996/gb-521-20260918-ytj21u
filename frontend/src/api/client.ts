import { ApiRequestError, type ApiEnvelope, type PagedResult } from '../types/api';

const TOKEN_KEY = 'mine-ventilation-token';

export function getStoredToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY);
}

export function setStoredToken(token: string | null): void {
  if (token) sessionStorage.setItem(TOKEN_KEY, token);
  else sessionStorage.removeItem(TOKEN_KEY);
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('Accept', 'application/json');
  if (init.body) headers.set('Content-Type', 'application/json');
  const token = getStoredToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);

  let response: Response;
  try {
    response = await fetch(path, { ...init, headers });
  } catch (cause) {
    throw new ApiRequestError(0, 'NETWORK_UNAVAILABLE', '无法连接服务，请检查网络后重试', '', cause);
  }
  const envelope = (await response.json().catch(() => ({ request_id: '' }))) as ApiEnvelope<T>;
  if (!response.ok || envelope.error) {
    throw new ApiRequestError(
      response.status,
      envelope.error?.code ?? 'UNEXPECTED_RESPONSE',
      envelope.error?.message ?? `服务返回 ${response.status}`,
      envelope.request_id,
      envelope.error?.details,
    );
  }
  if (envelope.data === undefined) throw new ApiRequestError(response.status, 'EMPTY_RESPONSE', '服务未返回预期数据', envelope.request_id);
  return envelope.data;
}

export async function requestPage<T>(path: string): Promise<PagedResult<T>> {
  const token = getStoredToken();
  let response: Response;
  try {
    response = await fetch(path, { headers: { Accept: 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) } });
  } catch (cause) {
    throw new ApiRequestError(0, 'NETWORK_UNAVAILABLE', '无法连接服务，请检查网络后重试', '', cause);
  }
  const envelope = (await response.json().catch(() => ({ request_id: '' }))) as ApiEnvelope<T[]>;
  if (!response.ok || envelope.error) {
    throw new ApiRequestError(response.status, envelope.error?.code ?? 'UNEXPECTED_RESPONSE', envelope.error?.message ?? `服务返回 ${response.status}`, envelope.request_id, envelope.error?.details);
  }
  return {
    items: envelope.data ?? [],
    meta: envelope.meta ?? { page: 1, page_size: 100, total: envelope.data?.length ?? 0, total_pages: 1 },
  };
}
