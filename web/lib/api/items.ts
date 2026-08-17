import { request } from "@/lib/api/client";
import { ApiRoutes } from "@/lib/constants";
import type { CreateItemInput, Item } from "@/types/item";

const HTTP_METHOD_POST = "POST";

export const HealthApi = {
  check: () => request<{ status: string; database: string }>(ApiRoutes.health),
};

// getItems/getItem/createItem are the copyable pattern for a new resource's
// frontend API layer - one function per endpoint, all going through the
// shared request() helper in lib/api/client.ts.
export const ItemsApi = {
  list: () => request<Item[]>(ApiRoutes.items),

  get: (id: string) => request<Item>(ApiRoutes.itemById(id)),

  create: (input: CreateItemInput) =>
    request<Item>(ApiRoutes.items, {
      method: HTTP_METHOD_POST,
      body: JSON.stringify(input),
    }),
};
