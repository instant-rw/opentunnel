package webassets

import "embed"

// Files contains the statically exported dashboard. The production Docker build
// replaces out/ with the output of `next build` before compiling the server.
//
//go:embed out
var Files embed.FS
