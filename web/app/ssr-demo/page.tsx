import Link from "next/link";
import { ItemsApi } from "@/lib/api/items";
import { ItemList } from "@/components/items/ItemList";
import { AppRoutes } from "@/lib/constants";

// No "use client" - this runs on the server. The fetch below happens before
// any HTML reaches the browser, so the item data is present in view-source,
// not just in the hydrated DOM. See error.tsx and loading.tsx for the
// required async-boundary states.
export default async function SSRDemoPage() {
  const items = await ItemsApi.list();

  return (
    <main className="page">
      <Link href={AppRoutes.home}>&larr; Home</Link>
      <h1>Server-rendered demo</h1>
      <p className="render-badge">
        Rendered on the server - this list was fetched before the HTML was
        sent to your browser.
      </p>
      <ItemList items={items} />
    </main>
  );
}
