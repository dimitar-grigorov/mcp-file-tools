// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package main

import (
	"bytes"
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

// Subset of wininet.h needed for one HTTPS GET.
const (
	internetOpenTypePreconfig = 0 // use the system proxy configuration
	internetServiceHTTP       = 3
	internetDefaultHTTPSPort  = 443

	internetFlagSecure       = 0x00800000
	internetFlagReload       = 0x80000000
	internetFlagNoCacheWrite = 0x04000000

	internetOptionConnectTimeout = 2
	internetOptionSendTimeout    = 5
	internetOptionReceiveTimeout = 6

	httpQueryStatusCode = 19
	httpQueryFlagNumber = 0x20000000
)

// WinInet's codes live in wininet.dll, not the system message table, so Go renders them
// as a bare "winapi error #12029". These are the ones a download realistically hits.
var wininetMessages = map[syscall.Errno]string{
	12002: "the request timed out",
	12005: "malformed URL",
	12007: "cannot resolve the host name",
	12029: "cannot connect to the server",
	12030: "the connection was aborted",
	12031: "the connection was reset by the server",
	12037: "the server's certificate has expired",
	12038: "the server's certificate is for a different host",
	12045: "the server's certificate is not from a trusted authority (a TLS-inspecting proxy?)",
	12057: "cannot check the server's certificate revocation status",
	12152: "invalid response from the server",
	12163: "no network connection",
}

// explain adds WinInet's message text; the numeric code stays for searchability.
func explain(err error) error {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		if text, ok := wininetMessages[errno]; ok {
			return fmt.Errorf("%s (wininet %d)", text, uintptr(errno))
		}
	}
	return err
}

const (
	phaseTimeoutMs = 30_000
	readChunkSize  = 64 << 10
)

var (
	wininet = syscall.NewLazyDLL("wininet.dll")

	procInternetOpen        = wininet.NewProc("InternetOpenW")
	procInternetConnect     = wininet.NewProc("InternetConnectW")
	procInternetSetOption   = wininet.NewProc("InternetSetOptionW")
	procInternetReadFile    = wininet.NewProc("InternetReadFile")
	procInternetCloseHandle = wininet.NewProc("InternetCloseHandle")
	procHTTPOpenRequest     = wininet.NewProc("HttpOpenRequestW")
	procHTTPSendRequest     = wininet.NewProc("HttpSendRequestW")
	procHTTPQueryInfo       = wininet.NewProc("HttpQueryInfoW")
)

// httpGet fetches https://host/path into memory, following redirects. WinInet rather
// than net/http, so TLS comes from Schannel and the system proxy is honoured.
func httpGet(host, path string) ([]byte, error) {
	if err := wininet.Load(); err != nil {
		return nil, fmt.Errorf("wininet.dll: %w", err)
	}

	// Every uintptr(unsafe.Pointer(..)) below must stay inside its Call argument list;
	// that is the only form the runtime keeps alive across the syscall.
	agentW, err := syscall.UTF16PtrFromString(userAgent)
	if err != nil {
		return nil, err
	}
	hostW, err := syscall.UTF16PtrFromString(host)
	if err != nil {
		return nil, err
	}
	pathW, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	verbW, err := syscall.UTF16PtrFromString("GET")
	if err != nil {
		return nil, err
	}

	session, _, err := procInternetOpen.Call(
		uintptr(unsafe.Pointer(agentW)), internetOpenTypePreconfig, 0, 0, 0)
	if session == 0 {
		return nil, fmt.Errorf("InternetOpen: %w", explain(err))
	}
	defer procInternetCloseHandle.Call(session)

	for _, option := range [...]uintptr{
		internetOptionConnectTimeout,
		internetOptionSendTimeout,
		internetOptionReceiveTimeout,
	} {
		timeout := uint32(phaseTimeoutMs)
		// best effort: a rejected timeout should not fail the launch
		procInternetSetOption.Call(session, option,
			uintptr(unsafe.Pointer(&timeout)), unsafe.Sizeof(timeout))
	}

	connection, _, err := procInternetConnect.Call(session,
		uintptr(unsafe.Pointer(hostW)), internetDefaultHTTPSPort,
		0, 0, internetServiceHTTP, 0, 0)
	if connection == 0 {
		return nil, fmt.Errorf("connecting to %s: %w", host, explain(err))
	}
	defer procInternetCloseHandle.Call(connection)

	// RELOAD|NO_CACHE_WRITE bypasses the WinInet cache both ways.
	request, _, err := procHTTPOpenRequest.Call(connection,
		uintptr(unsafe.Pointer(verbW)), uintptr(unsafe.Pointer(pathW)), 0, 0, 0,
		internetFlagSecure|internetFlagReload|internetFlagNoCacheWrite, 0)
	if request == 0 {
		return nil, fmt.Errorf("HttpOpenRequest: %w", explain(err))
	}
	defer procInternetCloseHandle.Call(request)

	if ok, _, err := procHTTPSendRequest.Call(request, 0, 0, 0, 0); ok == 0 {
		return nil, fmt.Errorf("requesting %s: %w", path, explain(err))
	}

	status, err := statusCode(request)
	if err != nil {
		return nil, err
	}
	if status < 200 || status > 299 {
		return nil, fmt.Errorf("HTTP %d", status)
	}

	var body bytes.Buffer
	chunk := make([]byte, readChunkSize)
	for {
		var read uint32
		ok, _, err := procInternetReadFile.Call(request,
			uintptr(unsafe.Pointer(&chunk[0])), readChunkSize,
			uintptr(unsafe.Pointer(&read)))
		if ok == 0 {
			return nil, fmt.Errorf("reading the response: %w", explain(err))
		}
		if read == 0 {
			return body.Bytes(), nil
		}
		body.Write(chunk[:read])
	}
}

func statusCode(request uintptr) (uint32, error) {
	var status uint32
	size := uint32(unsafe.Sizeof(status))

	ok, _, err := procHTTPQueryInfo.Call(request, httpQueryStatusCode|httpQueryFlagNumber,
		uintptr(unsafe.Pointer(&status)), uintptr(unsafe.Pointer(&size)), 0)
	if ok == 0 {
		return 0, fmt.Errorf("HttpQueryInfo: %w", explain(err))
	}
	return status, nil
}
