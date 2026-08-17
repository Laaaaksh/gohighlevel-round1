"use client";

import { useState, type FormEvent } from "react";
import { ItemsApi } from "@/lib/api/items";
import { ApiError } from "@/lib/api/client";
import type { Item } from "@/types/item";

const FIELD_NAME = "name";
const FIELD_NAME_ERROR = "name-error";
const FIELD_DESCRIPTION = "description";

// A discriminated union makes "submitting and errored" unrepresentable -
// see nextjs-frontend-standards on modeling async state explicitly.
type SubmitState =
  | { status: "idle" }
  | { status: "submitting" }
  | { status: "error"; message: string };

export function ItemForm({ onCreated }: { onCreated: (item: Item) => void }) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [state, setState] = useState<SubmitState>({ status: "idle" });

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setState({ status: "submitting" });

    try {
      const created = await ItemsApi.create({ name, description });
      onCreated(created);
      setName("");
      setDescription("");
      setState({ status: "idle" });
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to create item.";
      setState({ status: "error", message });
    }
  }

  const submitting = state.status === "submitting";
  const errored = state.status === "error";

  return (
    <form onSubmit={handleSubmit} className="item-form">
      <div className="form-field">
        <label htmlFor={FIELD_NAME}>Name</label>
        <input
          id={FIELD_NAME}
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          disabled={submitting}
          aria-invalid={errored}
          aria-describedby={errored ? FIELD_NAME_ERROR : undefined}
        />
      </div>
      <div className="form-field">
        <label htmlFor={FIELD_DESCRIPTION}>Description</label>
        <input
          id={FIELD_DESCRIPTION}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          disabled={submitting}
        />
      </div>
      <button type="submit" disabled={submitting}>
        {submitting ? "Creating…" : "Create item"}
      </button>
      {state.status === "error" && (
        <p id={FIELD_NAME_ERROR} role="alert" className="form-error">
          {state.message}
        </p>
      )}
    </form>
  );
}
