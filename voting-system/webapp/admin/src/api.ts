export const BASE_URL_DEFAULT = (import.meta as any).env?.VITE_API_BASE_URL || '';

export async function api<T>(path: string, opts: RequestInit, token?: string) : Promise<T> {
  const res = await fetch(path, {
    ...opts,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(opts.headers || {}),
    },
  });
  const body = await res.text();
  try {
    const json = JSON.parse(body);
    if (!res.ok) throw new Error(json?.message || json?.error || res.statusText);
    return json as T;
  } catch (e) {
    if (!res.ok) throw new Error(body || res.statusText);
    throw e;
  }
}

export function toEpochSeconds(dtLocal: string): number {
  if (!dtLocal) return 0;
  const ms = Date.parse(dtLocal);
  return Math.floor(ms / 1000);
}

export function formatTs(value: any): string {
  if (!value) return '-';
  const n = typeof value === 'number' ? value : (String(value).match(/^\d+$/) ? Number(value) : NaN);
  const d = isNaN(n) ? new Date(value) : new Date((n.toString().length <= 10 ? n * 1000 : n));
  if (isNaN(d.getTime())) return String(value);
  return d.toLocaleString();
} 