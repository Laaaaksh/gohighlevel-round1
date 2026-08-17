import Link from "next/link";
import { AppRoutes } from "@/lib/constants";

// Plain landing page - no data fetching, so it stays a server component by
// default with no directive needed.
export default function HomePage() {
  return (
    <main className="page">
      <h1>Hello World</h1>
      <p>
        This is the interview scaffold&apos;s landing page. It links to two
        demo pages that each fetch the same <code>/api/items</code> resource
        from the Go backend, one rendered on the server and one on the
        client.
      </p>
      <nav className="nav-links">
        <Link href={AppRoutes.ssrDemo} className="card-link">
          <strong>Server-rendered demo</strong>
          <span>Fetches on the server - data is present in view-source.</span>
        </Link>
        <Link href={AppRoutes.csrDemo} className="card-link">
          <strong>Client-rendered demo</strong>
          <span>Fetches in the browser with a visible loading state.</span>
        </Link>
      </nav>
    </main>
  );
}
