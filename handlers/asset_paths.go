package handlers

// Local asset path resolution for Global Watch basemap and similar static files.
// Prefer frontend/assets (embedded + on-disk); fall back to root drop-ins (e.g. 1.jpg).
// High-resolution equirectangular maps (4k–8k, multi-MB) are first-class: local-only, no CDN.

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
)

// earthTextureCandidates lists filesystem locations for the equirectangular Earth basemap.
// Order is preference for same-size ties; resolveEarthTexturePath picks the largest file.
func earthTextureCandidates() []string {
	return []string{
		filepath.Join("frontend", "assets", "earth_day_8k.jpg"),
		filepath.Join("frontend", "assets", "earth_day_4k.jpg"),
		filepath.Join("frontend", "assets", "earth_day.jpg"),
		filepath.Join("frontend", "assets", "earth_day.jpeg"),
		filepath.Join("frontend", "assets", "1.jpg"),
		"1.jpg", // operator drop-in at project / process cwd
		filepath.Join("assets", "earth_day.jpg"),
		filepath.Join("vgt_workspace", "earth_day.jpg"),
	}
}

// EnsureEarthTextureOnDisk copies the best root drop-in into frontend/assets when
// no high-res basemap exists yet. Safe no-op if a large dest is already present.
func EnsureEarthTextureOnDisk() {
	// Prefer keeping an existing high-res map ( > ~500KB suggests real imagery, not a stub ).
	if p, err := resolveEarthTexturePath(); err == nil {
		if st, stErr := os.Stat(p); stErr == nil && st.Size() > 500*1024 {
			return
		}
	}
	dest := filepath.Join("frontend", "assets", "earth_day.jpg")
	if st, err := os.Stat(dest); err == nil && st.Size() > 500*1024 {
		return
	}
	for _, src := range []string{
		"1.jpg",
		filepath.Join("frontend", "assets", "1.jpg"),
		filepath.Join("frontend", "assets", "earth_day_8k.jpg"),
		filepath.Join("frontend", "assets", "earth_day_4k.jpg"),
	} {
		in, err := os.Open(src)
		if err != nil {
			continue
		}
		info, _ := in.Stat()
		if info != nil && info.Size() < 50*1024 {
			_ = in.Close()
			continue
		}
		_ = os.MkdirAll(filepath.Dir(dest), 0755)
		out, err := os.Create(dest)
		if err != nil {
			_ = in.Close()
			log.Printf("earth texture: cannot create %s: %v", dest, err)
			return
		}
		n, err := io.Copy(out, in)
		_ = in.Close()
		_ = out.Close()
		if err != nil || n < 50*1024 {
			log.Printf("earth texture: copy failed from %s: n=%d err=%v", src, n, err)
			_ = os.Remove(dest)
			continue
		}
		log.Printf("earth texture: installed %s (%d bytes) from %s", dest, n, src)
		return
	}
}

// resolveEarthTexturePath returns the largest available equirectangular Earth map on disk.
func resolveEarthTexturePath() (string, error) {
	type cand struct {
		path string
		size int64
	}
	var found []cand
	for _, p := range earthTextureCandidates() {
		st, err := os.Stat(p)
		if err != nil || st.IsDir() || st.Size() <= 10*1024 {
			continue
		}
		found = append(found, cand{path: p, size: st.Size()})
	}
	if len(found) == 0 {
		return "", os.ErrNotExist
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].size != found[j].size {
			return found[i].size > found[j].size
		}
		return found[i].path < found[j].path
	})
	return found[0].path, nil
}
