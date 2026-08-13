package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// versionedAssets are the first-party files referenced from the templates whose
// contents change between deploys. The brand images belong here alongside the
// stylesheet and the scripts: they move only when the palette moves, and a cache
// holding a previous mark beside the current interface is the same class of
// mismatch as a cache holding a previous stylesheet. Every path listed must
// exist under the public directory, or it is served unversioned.
var versionedAssets = []string{
	"/app.css",
	"/index.js",
	"/render-body.js",
	"/har.js",
	"/endpoint.js",
	"/logo.svg",
	"/favicon.ico",
	"/apple-touch-icon.png",
	"/social-card.png",
}

// assetIndex maps a served static path to a short hash of its contents. The
// stylesheet and the page scripts live at fixed paths, so without a version in
// the URL a CDN or a browser can pair a cached copy of one deploy with the HTML
// of the next: every class name the new markup asks for resolves to nothing and
// the page renders unstyled. Versioning the URL makes a changed asset a
// different URL, which no cache can confuse for the old one.
type assetIndex struct {
	dir string
	// reload re-hashes an asset whose modification time has moved, so an edit
	// is picked up without a restart. It is off wherever the files cannot
	// change under a running process, which spares every render a stat call.
	reload bool

	mu       sync.Mutex
	versions map[string]string
	stamps   map[string]time.Time
}

// newAssetIndex hashes every versioned asset once, so a render never pays for
// reading a file that has not changed.
func newAssetIndex(dir string, reload bool) *assetIndex {
	index := &assetIndex{
		dir:      dir,
		reload:   reload,
		versions: make(map[string]string, len(versionedAssets)),
		stamps:   make(map[string]time.Time, len(versionedAssets)),
	}
	for _, path := range versionedAssets {
		index.versions[path] = index.hash(path)
	}
	return index
}

// url is exposed to templates as `asset`. An asset the index could not read
// falls back to its plain path: an unversioned URL is a caching risk, but a
// broken reference is a broken page.
func (a *assetIndex) url(path string) string {
	version := a.versions[path]
	if a.reload {
		version = a.refresh(path)
	}
	if version == "" {
		return path
	}
	return path + "?v=" + version
}

// refresh re-hashes path only when its modification time has moved, so an
// edited asset is picked up without re-reading every asset on every render.
func (a *assetIndex) refresh(path string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if info, err := os.Stat(a.dir + path); err == nil && !info.ModTime().Equal(a.stamps[path]) {
		a.stamps[path] = info.ModTime()
		a.versions[path] = a.hash(path)
	}
	return a.versions[path]
}

func (a *assetIndex) hash(path string) string {
	f, err := os.Open(a.dir + path)
	if err != nil {
		slog.Warn("asset missing, serving unversioned", "path", path, "err", err)
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		slog.Warn("asset unreadable, serving unversioned", "path", path, "err", err)
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))[:10]
}
