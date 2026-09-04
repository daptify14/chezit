package chezmoi

import "errors"

// Sentinel errors for policy enforcement.
var (
	ErrReadOnly      = errors.New("chezmoi manager is read-only")
	ErrOutsideTarget = errors.New("path is outside target directory")
	ErrPathEmpty     = errors.New("path is empty")
	ErrPathNotAbs    = errors.New("path must be absolute")
	ErrInvalidHash   = errors.New("invalid git commit hash")
)

// Fetch failure classes. GitFetch wraps git's output around one of these so
// callers can render a short reason with errors.Is instead of parsing text.
var (
	ErrFetchTimeout     = errors.New("git fetch timed out")
	ErrFetchUnreachable = errors.New("remote unreachable")
	ErrFetchAuth        = errors.New("remote authentication failed")
)
