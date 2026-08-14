// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import "encoding/base64"

// A document whose last line is broken, for garbled text. Kept as source and
// encoded at startup rather than pasted in as a base64 blob.
const serverIconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">` +
	`<rect width="32" height="32" rx="7" fill="#2f6feb"/>` +
	`<path d="M11 7h7l5 5v13a1.6 1.6 0 0 1-1.6 1.6H11A1.6 1.6 0 0 1 9.4 25V8.6A1.6 1.6 0 0 1 11 7z" fill="#fff"/>` +
	`<path d="M18 7l5 5h-5z" fill="#bcd4ff"/>` +
	`<path d="M12.4 16h8M12.4 20h2.4m2.4 0h1.2m2.4 0h1.2" stroke="#2f6feb" stroke-width="1.7" stroke-linecap="round"/>` +
	`</svg>`

// Self-contained: a remote URL would make a client's server list hit the network.
var serverIconDataURI = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(serverIconSVG))
