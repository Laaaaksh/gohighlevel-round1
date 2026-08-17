import { API_BASE_URL } from "@/lib/constants";

const HEADER_CONTENT_TYPE = "Content-Type";
const CONTENT_TYPE_JSON = "application/json";

// Mirrors pkg/apperror/response.go's wire format on the Go backend.
interface ApiErrorBody {
  code: string;
  message: string;
  fields?: Record<string, string>;
}

function isApiErrorBody(value: unknown): value is ApiErrorBody {
  return (
    typeof value === "object" &&
    value !== null &&
    "message" in value &&
    typeof (value as { message: unknown }).message === "string"
  );
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
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
  // no-store: every page in this scaffold wants the current DB state, not a
  // cached response - see nextjs-frontend-standards' SSR/CSR/ISR table.
  const res = await fetch(`${API_BASE_URL}${path}`, {
    cache: "no-store",
    ...init,
    headers: { [HEADER_CONTENT_TYPE]: CONTENT_TYPE_JSON, ...init?.headers },
  });

  if (!res.ok) {
    const body: unknown = await res.json().catch(() => null);
    const message = isApiErrorBody(body) ? body.message : res.statusText;
    throw new ApiError(res.status, message);
  }

  if (res.status === NO_CONTENT_STATUS) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

const NO_CONTENT_STATUS = 204;
