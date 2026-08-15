// Package settings models the desired state of a remote repository's settings and
// reconciles a hosting provider against it. It owns the schema, the diff and the
// report; it performs no I/O against any provider and must never import
// internal/provider. Providers implement its Group interface instead.
package settings
