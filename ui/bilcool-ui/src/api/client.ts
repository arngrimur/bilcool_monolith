export async function apiFetch<T>(input: RequestInfo, init?: RequestInit): Promise<T> {
  const token = localStorage.getItem('bilcool_token');
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    'Correlation-Id': crypto.randomUUID(),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...init?.headers,
  };
  const res = await fetch(input, { ...init, headers });
  if (res.status === 401) {
    localStorage.removeItem('bilcool_token');
    window.dispatchEvent(new Event('auth:logout'));
    throw new Error('Unauthorized');
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: 'Unknown error' }));
    throw Object.assign(new Error(err.message ?? 'Request failed'), { status: res.status, body: err });
  }
  if (res.status === 204 || res.headers.get('content-length') === '0') return undefined as T;
  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}
