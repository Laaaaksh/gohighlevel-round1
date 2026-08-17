// Every route, storage key, and status string lives here - no inline
// literals in components or the API layer, matching the Go backend's
// no-magic-values rule.

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const ApiRoutes = {
  health: "/health",
  items: "/api/items",
  itemById: (id: string) => `/api/items/${id}`,
} as const;

export const AppRoutes = {
  home: "/",
  ssrDemo: "/ssr-demo",
  csrDemo: "/csr-demo",
} as const;
