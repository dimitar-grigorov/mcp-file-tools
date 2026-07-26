// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

// Package errkind tags an error with a stable failure category, message unchanged.
package errkind

import (
	"context"
	"errors"
	"io/fs"
)

// Kind is a stable failure category.
type Kind uint8

const (
	Unknown Kind = iota
	InvalidInput
	InvalidPath
	AccessDenied
	SymlinkEscape
	NotFound
	Permission
	Encoding
	Cancelled
	Filesystem
)

func (k Kind) String() string {
	switch k {
	case InvalidInput:
		return "invalid_input"
	case InvalidPath:
		return "invalid_path"
	case AccessDenied:
		return "access_denied"
	case SymlinkEscape:
		return "symlink_escape"
	case NotFound:
		return "not_found"
	case Permission:
		return "permission"
	case Encoding:
		return "encoding"
	case Cancelled:
		return "cancelled"
	case Filesystem:
		return "filesystem"
	default:
		return "unknown"
	}
}

// Error carries a Kind; message and errors.Is/As chain are preserved.
type Error struct {
	Kind Kind
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Kind.String()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// New creates a typed sentinel error.
func New(kind Kind, message string) *Error {
	return &Error{Kind: kind, Err: errors.New(message)}
}

// Wrap tags err with a kind. Nil stays nil.
func Wrap(kind Kind, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Err: err}
}

// Of returns the first explicit kind in the tree, else classifies stdlib sentinels.
func Of(err error) Kind {
	if err == nil {
		return Unknown
	}

	var typed *Error
	if errors.As(err, &typed) && typed != nil {
		return typed.Kind
	}

	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return Cancelled
	case errors.Is(err, fs.ErrNotExist):
		return NotFound
	case errors.Is(err, fs.ErrPermission):
		return Permission
	default:
		return Unknown
	}
}
