# Frontend static assets (local, no CDN)

| File | Purpose |
|------|---------|
| `earth_day.jpg` | Equirectangular Earth basemap for Global Watch globe |
| `earth_day_8k.jpg` / `earth_day_4k.jpg` | Optional high-res overrides (preferred when present) |
| `world-atlas-110m.topojson` | Country borders (TopoJSON World Atlas, ISC) |
| `THIRD_PARTY_NOTICES.md` | Licenses |

## Earth basemap quality

Global Watch samples a **local** equirectangular (2:1) day map onto an orthographic sphere.
There is **no tile CDN** and no internet latency at runtime, so large files are fine:

| Resolution | Approx. size | Notes |
|------------|--------------|--------|
| 2048×1024 | ~1–2 MB | usable |
| 4096×2048 | ~3–8 MB | good |
| **8192×4096** | **~8–15 MB** | **recommended for desktop** |

### Install / upgrade

```powershell
# One-time network download of 8k day map into frontend/assets/
.\scripts\install_earth_texture.ps1 -Download

# Or drop your own NASA Blue Marble / equirectangular JPG:
.\scripts\install_earth_texture.ps1 -Source "D:\maps\blue_marble_8k.jpg"
```

Also accepted at runtime:

1. `frontend/assets/earth_day_8k.jpg` (preferred)
2. `frontend/assets/earth_day.jpg`
3. Project root `1.jpg` (copied on startup when no large map exists)
4. `GET /v1/assets/earth-texture` (serves the **largest** on-disk candidate)

The HUD line `TEX:8192x4096/jpg` confirms high-res load. If you still see `TEX:1536…`, hard-reload after installing a new map (old WebView cache).

If no JPG is present, AETHEL bakes a land/ocean texture from the TopoJSON atlas.
