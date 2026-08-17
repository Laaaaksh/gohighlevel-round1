"use client";

// Catches a render-time exception in this route (distinct from the inline
// fetch-error state in page.tsx, which is expected and handled explicitly).
export default function Error({
  error,
  reset,
}: {
  error: Error;
  reset: () => void;
}) {
  return (
    <main className="page" role="alert">
      <h2>Something went wrong</h2>
      <p>{error.message}</p>
      <button onClick={reset}>Try again</button>
    </main>
  );
}
