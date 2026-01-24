package handler

const (
	// DefaultEncoding is the default encoding used when none is specified
	DefaultEncoding = "cp1251"
)

// Handler handles all file tool operations
type Handler struct {
	defaultEncoding string
}

// NewHandler creates a new Handler with default settings
func NewHandler() *Handler {
	return &Handler{
		defaultEncoding: DefaultEncoding,
	}
}
