package sandbox

import (
	"embed"
	"net/http"
	"strings"
)

// Embed sandbox.html and defaultCode.js
//
//go:embed sandbox.html defaultCode.js
var SandboxAssets embed.FS

// ServeSandbox serves the embedded sandbox files
func ServeSandbox(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "defaultCode.js") {
		// Serve the JavaScript file
		data, err := SandboxAssets.ReadFile("defaultCode.js")
		if err != nil {
			http.Error(w, "failed to read defaultCode.js", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(data)
		return
	}
	// Serve the HTML file
	data, err := SandboxAssets.ReadFile("sandbox.html")
	if err != nil {
		http.Error(w, "failed to read sandbox.html", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}
