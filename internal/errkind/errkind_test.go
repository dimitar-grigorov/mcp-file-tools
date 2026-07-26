// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package errkind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"
)

func TestWrapPreservesMessageAndCause(t *testing.T) {
	cause := fs.ErrPermission
	err := Wrap(Permission, fmt.Errorf("failed to write target: %w", cause))

	if got, want := err.Error(), "failed to write target: permission denied"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Fatal("wrapped error must preserve errors.Is compatibility")
	}
	if got := Of(err); got != Permission {
		t.Fatalf("Of() = %v, want %v", got, Permission)
	}

	var typed *Error
	if !errors.As(err, &typed) {
		t.Fatal("wrapped error must support errors.As to *Error")
	}
}

func TestWrapNilReturnsNil(t *testing.T) {
	if err := Wrap(Filesystem, nil); err != nil {
		t.Fatalf("Wrap(nil) = %v, want nil", err)
	}
}

func TestNewSentinelIsMatchableWhenWrapped(t *testing.T) {
	sentinel := New(AccessDenied, "access denied - path outside allowed directories")
	wrapped := fmt.Errorf("%w: %s", sentinel, "/etc/passwd")

	if !errors.Is(wrapped, sentinel) {
		t.Fatal("errors.Is must match a typed sentinel through fmt.Errorf")
	}
	if got := Of(wrapped); got != AccessDenied {
		t.Fatalf("Of() = %v, want %v", got, AccessDenied)
	}
	if got, want := sentinel.Error(), "access denied - path outside allowed directories"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestOfClassifiesStandardErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{name: "cancelled", err: context.Canceled, want: Cancelled},
		{name: "deadline", err: context.DeadlineExceeded, want: Cancelled},
		{name: "not found", err: fmt.Errorf("open: %w", fs.ErrNotExist), want: NotFound},
		{name: "permission", err: fmt.Errorf("open: %w", fs.ErrPermission), want: Permission},
		{name: "unknown", err: errors.New("boom"), want: Unknown},
		{name: "nil", err: nil, want: Unknown},
		{name: "typed wins over stdlib", err: Wrap(InvalidPath, fs.ErrNotExist), want: InvalidPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Of(tt.err); got != tt.want {
				t.Fatalf("Of(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestOfUsesFirstTypedErrorInJoin(t *testing.T) {
	primary := New(SymlinkEscape, "symlink escape")
	secondary := New(Filesystem, "cleanup failed")

	if got := Of(errors.Join(primary, secondary)); got != SymlinkEscape {
		t.Fatalf("Of(join) = %v, want %v", got, SymlinkEscape)
	}
}

func TestKindStringIsStable(t *testing.T) {
	if got, want := SymlinkEscape.String(), "symlink_escape"; got != want {
		t.Fatalf("SymlinkEscape.String() = %q, want %q", got, want)
	}
	if got, want := Kind(255).String(), "unknown"; got != want {
		t.Fatalf("unknown Kind.String() = %q, want %q", got, want)
	}
}

func TestNilErrorStringDoesNotPanic(t *testing.T) {
	var e *Error
	if got := e.Error(); got != "<nil>" {
		t.Fatalf("(*Error)(nil).Error() = %q, want \"<nil>\"", got)
	}
	if e.Unwrap() != nil {
		t.Fatal("(*Error)(nil).Unwrap() must be nil")
	}
	if got := (&Error{Kind: NotFound}).Error(); got != "not_found" {
		t.Fatalf("Error() with no cause = %q, want \"not_found\"", got)
	}
}
