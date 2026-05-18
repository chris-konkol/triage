package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCCodeToHTTP(t *testing.T) {
	cases := []struct {
		code codes.Code
		want int
	}{
		{codes.NotFound, http.StatusNotFound},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.AlreadyExists, http.StatusConflict},
		{codes.Unauthenticated, http.StatusUnauthorized},
		{codes.PermissionDenied, http.StatusForbidden},
		{codes.Internal, http.StatusInternalServerError},
		{codes.OK, http.StatusInternalServerError}, // default branch
		{codes.Unavailable, http.StatusInternalServerError},
	}

	for _, tc := range cases {
		err := status.Error(tc.code, "test message")
		got := grpcCodeToHTTP(err)
		if got != tc.want {
			t.Errorf("grpcCodeToHTTP(%v) = %d, want %d", tc.code, got, tc.want)
		}
	}
}

func TestGRPCMessage_ExtractsStatusMessage(t *testing.T) {
	err := status.Error(codes.NotFound, "ticket not found")
	if msg := grpcMessage(err); msg != "ticket not found" {
		t.Errorf("grpcMessage = %q, want %q", msg, "ticket not found")
	}
}

func TestGRPCMessage_FallsBackToErrorString(t *testing.T) {
	// A plain error (not a gRPC status) should fall back to err.Error().
	err := &plainError{"something went wrong"}
	if msg := grpcMessage(err); msg != "something went wrong" {
		t.Errorf("grpcMessage = %q, want %q", msg, "something went wrong")
	}
}

func TestWriteError_WritesJSONBody(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, http.StatusBadRequest, "bad input")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	if body != `{"error":"bad input"}` {
		t.Errorf("body = %q, want %q", body, `{"error":"bad input"}`)
	}
}

// plainError is a non-gRPC error for testing the fallback path in grpcMessage.
type plainError struct{ msg string }

func (e *plainError) Error() string { return e.msg }
