import { HttpResponse } from 'msw';

export type MockEnvelope<T> = {
  code: number;
  msg: string;
  data: T;
};

export function ok<T>(data: T, msg = 'ok'): HttpResponse<MockEnvelope<T>> {
  return HttpResponse.json<MockEnvelope<T>>({ code: 0, msg, data });
}

export function fail<T = null>(
  code: number,
  msg: string,
  data: T = null as T
): HttpResponse<MockEnvelope<T>> {
  return HttpResponse.json<MockEnvelope<T>>({ code, msg, data });
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export async function readJsonObject(request: Request): Promise<Record<string, unknown>> {
  try {
    const body: unknown = await request.json();
    return isRecord(body) ? body : {};
  } catch {
    return {};
  }
}

export function readString(body: Record<string, unknown>, key: string): string | null {
  const value = body[key];
  return typeof value === 'string' ? value : null;
}

export function readBoolean(body: Record<string, unknown>, key: string): boolean | null {
  const value = body[key];
  return typeof value === 'boolean' ? value : null;
}

export function readNumber(body: Record<string, unknown>, key: string): number | null {
  const value = body[key];
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}
