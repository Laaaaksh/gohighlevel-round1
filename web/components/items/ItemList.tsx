import type { Item } from "@/types/item";

// Presentational only - takes items and renders them. No fetching here, so
// both the server-rendered and client-rendered demo pages can reuse it.
export function ItemList({ items }: { items: Item[] }) {
  if (items.length === 0) {
    return <p className="empty-state">No items yet.</p>;
  }

  return (
    <ul className="item-list">
      {items.map((item) => (
        <li key={item.id} className="item-row">
          <strong>{item.name}</strong>
          {item.description && <p>{item.description}</p>}
        </li>
      ))}
    </ul>
  );
}
