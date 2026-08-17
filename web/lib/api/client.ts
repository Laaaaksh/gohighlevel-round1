import { API_BASE_URL } from "@/lib/constants";

const HEADER_CONTENT_TYPE = "Content-Type";
const CONTENT_TYPE_JSON = "application/json";

// Per-field detail keyed by the request field the backend rejected, matching
// pkg/apperror's Fields map. Values are optional because a lookup by an
// arbitrary field name may simply be absent.
export type ApiErrorFields = Readonly<Record<string, string | undefined>>;

export const NO_API_ERROR_FIELDS: ApiErrorFields = {};

// Mirrors pkg/apperror/response.go's wire format on the Go backend.
interface ApiErrorBody {
  code: string;
  message: string;
  fields?: ApiErrorFields;
}

function isStringRecord(value: unknown): value is ApiErrorFields {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return false;
  }
  return Object.values(value as Record<string, unknown>).every(
    (entry) => typeof entry === "string",
  );
}

function isApiErrorBody(value: unknown): value is ApiErrorBody {
  if (typeof value !== "object" || value === null || !("message" in value)) {
    return false;
  }
  const candidate = value as { message: unknown; fields?: unknown };
  if (typeof candidate.message !== "string") {
    return false;
  }
  return candidate.fields === undefined || isStringRecord(candidate.fields);
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
    // Empty when the backend did not attribute the failure to named fields -
    // callers must not guess which input was at fault.
    readonly fields: ApiErrorFields = NO_API_ERROR_FIELDS,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// request is the only place in the frontend that calls fetch against the
// Go API - every component goes through the typed functions built on top
// of it in lib/api/*.ts, never fetch directly.
export async function request<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  // Headers, not object spread: HeadersInit is also allowed to be a Headers
  // instance or a string[][], and spreading either silently yields no usable
  // header names.
  const headers = new Headers(init?.headers);
  if (!headers.has(HEADER_CONTENT_TYPE)) {
    headers.set(HEADER_CONTENT_TYPE, CONTENT_TYPE_JSON);
  }

  // no-store: every page in this scaffold wants the current DB state, not a
  // cached response - see nextjs-frontend-standards' SSR/CSR/ISR table.
  const res = await fetch(`${API_BASE_URL}${path}`, {
    cache: "no-store",
    ...init,
    headers,
  });

  if (!res.ok) {
    const body: unknown = await res.json().catch(() => null);
    if (!isApiErrorBody(body)) {
      throw new ApiError(res.status, res.statusText);
    }
    throw new ApiError(res.status, body.message, body.fields ?? NO_API_ERROR_FIELDS);
  }

  if (res.status === NO_CONTENT_STATUS) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

const NO_CONTENT_STATUS = 204;
