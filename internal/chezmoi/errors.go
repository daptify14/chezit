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
