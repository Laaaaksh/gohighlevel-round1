"use client";

export default function Error({
  error,
  reset,
}: {
  error: Error;
  reset: () => void;
}) {
  return (
    <main className="page" role="alert">
      <h2>Could not load items</h2>
      <p>{error.message}</p>
      <button onClick={reset}>Try again</button>
    </main>
  );
}
