// This file exists only to stop the root Go module's ./... pattern from
// descending into web/node_modules (some npm packages bundle Go source).
// web/ is a Node/Next.js project - this module is never built or imported.
module gohighlevel-round1-web-boundary

go 1.25
