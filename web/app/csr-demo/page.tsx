"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ItemsApi } from "@/lib/api/items";
import { ApiError } from "@/lib/api/client";
import { ItemList } from "@/components/items/ItemList";
import { ItemForm } from "@/components/items/ItemForm";
import { AppRoutes } from "@/lib/constants";
import type { Item } from "@/types/item";

// A discriminated union so "errored" and "ready" can't both be true at once
// - see nextjs-frontend-standards. There is deliberately no "loading" member:
// loading is derived during render (see `result` below) instead of stored, so
// the effect never has to call setState synchronously.
type LoadResult =
  | { status: "error"; message: string }
  | { status: "ready"; items: Item[] };

// Which fetch produced this result, so an older settled result is never
// mistaken for the answer to the request that is currently in flight.
type SettledLoad = LoadResult & { requestId: number };

// "use client" is required here for useState/useEffect - the fetch happens
// in the browser after the initial HTML arrives, which the loading state
// below makes visible.
export default function CSRDemoPage() {
  // Bumped whenever a fetch should start; the effect re-runs on every change.
  const [requestId, setRequestId] = useState(0);
  const [settled, setSettled] = useState<SettledLoad | null>(null);

  useEffect(() => {
    let cancelled = false;

    ItemsApi.list()
      .then((items) => {
        if (!cancelled) setSettled({ requestId, status: "ready", items });
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        const message = err instanceof ApiError ? err.message : "Failed to load items.";
        setSettled({ requestId, status: "error", message });
      });

    return () => {
      cancelled = true;
    };
  }, [requestId]);

  // Derived, not stored: a settled result only counts once its id matches the
  // request in flight, so bumping requestId shows the loading state on the
  // very next render - even when the previous attempt errored - without the
  // effect setting state synchronously.
  const result = settled?.requestId === requestId ? settled : null;

  // Refetch rather than splice the new item into the existing list: when the
  // initial load failed there is no list to splice into, and silently
  // dropping a successful create invites the user to submit it twice.
  function handleCreated() {
    setRequestId((current) => current + 1);
  }

  return (
    <main className="page">
      <Link href={AppRoutes.home}>&larr; Home</Link>
      <h1>Client-rendered demo</h1>
      <p className="render-badge">
        Rendered on the client - this list is fetched from your browser
        after the page loads.
      </p>

      {result === null && <p>Loading…</p>}
      {result?.status === "error" && (
        <p role="alert">{result.message}</p>
      )}
      {result?.status === "ready" && <ItemList items={result.items} />}

      <h2>Create an item</h2>
      <ItemForm onCreated={handleCreated} />
    </main>
  );
}
