package webassets

import "embed"

// Files contains the statically exported dashboard. The production Docker build
// replaces out/ with the output of `next build` before compiling the server.
//
// The all: prefix is required so Next.js assets under _next/ are included;
// plain //go:embed skips names that begin with '_' or '.'.
//
//go:embed all:out
var Files embed.FS
