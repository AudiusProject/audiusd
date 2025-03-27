package sandbox

import (
	"embed"
	"net/http"
)

//go:embed sandbox.html
var SandboxAssets embed.FS

// ServeSandbox serves the embedded sandbox files
func ServeSandbox(w http.ResponseWriter, r *http.Request) {
	// Serve the HTML file
	data, err := SandboxAssets.ReadFile("sandbox.html")
	if err != nil {
		http.Error(w, "failed to read sandbox.html", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}
