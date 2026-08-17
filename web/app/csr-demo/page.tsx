"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ItemsApi } from "@/lib/api/items";
import { ApiError } from "@/lib/api/client";
import { ItemList } from "@/components/items/ItemList";
import { ItemForm } from "@/components/items/ItemForm";
import { AppRoutes } from "@/lib/constants";
import type { Item } from "@/types/item";

// A discriminated union so "loading" and "errored" can't both be true at
// once - see nextjs-frontend-standards.
type LoadState =
  | { status: "loading" }
  | { status: "error"; message: string }
  | { status: "ready"; items: Item[] };

// "use client" is required here for useState/useEffect - the fetch happens
// in the browser after the initial HTML arrives, which the loading state
// below makes visible.
export default function CSRDemoPage() {
  const [state, setState] = useState<LoadState>({ status: "loading" });
  const [reloadCount, setReloadCount] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setState({ status: "loading" });

    ItemsApi.list()
      .then((items) => {
        if (!cancelled) setState({ status: "ready", items });
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        const message = err instanceof ApiError ? err.message : "Failed to load items.";
        setState({ status: "error", message });
      });

    return () => {
      cancelled = true;
    };
  }, [reloadCount]);

  // Refetch rather than splice the new item into the existing list: when the
  // initial load failed there is no list to splice into, and silently
  // dropping a successful create invites the user to submit it twice.
  function handleCreated() {
    setReloadCount((count) => count + 1);
  }

  return (
    <main className="page">
      <Link href={AppRoutes.home}>&larr; Home</Link>
      <h1>Client-rendered demo</h1>
      <p className="render-badge">
        Rendered on the client - this list is fetched from your browser
        after the page loads.
      </p>

      {state.status === "loading" && <p>Loading…</p>}
      {state.status === "error" && (
        <p role="alert">{state.message}</p>
      )}
      {state.status === "ready" && <ItemList items={state.items} />}

      <h2>Create an item</h2>
      <ItemForm onCreated={handleCreated} />
    </main>
  );
}
