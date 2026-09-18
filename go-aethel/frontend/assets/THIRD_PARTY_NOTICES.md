# Global Watch map data

## TopoJSON borders

`world-atlas-110m.topojson` is the self-hosted `countries-110m.json` dataset
from [TopoJSON World Atlas](https://github.com/topojson/world-atlas), Copyright
2013–2019 Michael Bostock.

The source dataset is used under the ISC license:

> Permission to use, copy, modify, and/or distribute this software for any
> purpose with or without fee is hereby granted, provided that the above
> copyright notice and this permission notice appear in all copies.

It is embedded solely for local rendering in Aethel Global Watch. No map tiles,
CDN requests, or third-party map runtime are used.

## Earth surface texture

`earth_day.jpg` / `earth_day_8k.jpg` (if present) are **local equirectangular
basemaps** used only for orthographic globe texturing in Global Watch. Load order:

1. `./assets/earth_day_8k.jpg` then `earth_day_4k.jpg` then `earth_day.jpg`
2. `GET /v1/assets/earth-texture` (serves the largest on-disk candidate, e.g. root `1.jpg`)
3. Runtime bake from TopoJSON world atlas

High resolution (4k–8k, multi-MB up to ~15 MB) is intentional: AETHEL runs on the
operator’s PC with no tile CDN and no runtime map downloads.

Operator install:

```powershell
.\scripts\install_earth_texture.ps1 -Download
# or: -Source path\to\equirectangular.jpg
```

You may replace the map with any equirectangular (2:1) Earth imagery you have
rights to use. Recommended public-domain source: **NASA Blue Marble** (US
Government work). Optional installer URL for Solar System Scope’s 8k day map is
based on NASA-derived data for personal use — verify their site license for your
deployment. AETHEL does not fetch map tiles at runtime.

## God's Eye View Geospatial Technology

Portions of the 3D geospatial rendering architecture, aircraft SVG vector silhouettes,
and camera control concepts adapted from [God's Eye View](https://github.com/bilawalsidhu/gods-eye-view).

Copyright (c) 2026 Bilawal Sidhu
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

