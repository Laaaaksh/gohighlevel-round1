"use client";

import { useState, type FormEvent } from "react";
import { ItemsApi } from "@/lib/api/items";
import { ApiError, NO_API_ERROR_FIELDS, type ApiErrorFields } from "@/lib/api/client";
import type { Item } from "@/types/item";

// Field ids double as the keys pkg/apperror uses in its Fields map, so a
// per-field message from the backend lands on the input it belongs to.
const FIELD_NAME = "name";
const FIELD_DESCRIPTION = "description";
const FIELD_NAME_ERROR = "name-error";
const FIELD_DESCRIPTION_ERROR = "description-error";

const FALLBACK_ERROR_MESSAGE = "Failed to create item.";

// A discriminated union makes "submitting and errored" unrepresentable -
// see nextjs-frontend-standards on modeling async state explicitly.
type SubmitState =
  | { status: "idle" }
  | { status: "submitting" }
  | { status: "error"; message: string; fields: ApiErrorFields };

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
      const apiErr = err instanceof ApiError ? err : null;
      setState({
        status: "error",
        message: apiErr?.message ?? FALLBACK_ERROR_MESSAGE,
        fields: apiErr?.fields ?? NO_API_ERROR_FIELDS,
      });
    }
  }

  const submitting = state.status === "submitting";
  // Only the fields the backend named are marked invalid; a transport failure
  // or a generic 400 leaves every input unmarked rather than blaming one.
  const fieldErrors = state.status === "error" ? state.fields : NO_API_ERROR_FIELDS;
  const nameError = fieldErrors[FIELD_NAME];
  const descriptionError = fieldErrors[FIELD_DESCRIPTION];

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
          aria-invalid={nameError !== undefined}
          aria-describedby={nameError !== undefined ? FIELD_NAME_ERROR : undefined}
        />
        {nameError !== undefined && (
          <p id={FIELD_NAME_ERROR} className="form-error">
            {nameError}
          </p>
        )}
      </div>
      <div className="form-field">
        <label htmlFor={FIELD_DESCRIPTION}>Description</label>
        <input
          id={FIELD_DESCRIPTION}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          disabled={submitting}
          aria-invalid={descriptionError !== undefined}
          aria-describedby={
            descriptionError !== undefined ? FIELD_DESCRIPTION_ERROR : undefined
          }
        />
        {descriptionError !== undefined && (
          <p id={FIELD_DESCRIPTION_ERROR} className="form-error">
            {descriptionError}
          </p>
        )}
      </div>
      <button type="submit" disabled={submitting}>
        {submitting ? "Creating…" : "Create item"}
      </button>
      {state.status === "error" && (
        <p role="alert" className="form-error">
          {state.message}
        </p>
      )}
    </form>
  );
}
