# VGT Aethel – Future Interface Design Plan

## Design objective

Unify Aethel as a futuristic operator environment: spatial depth instead of flat black surfaces, readable intelligence density, translucent command layers and consistent interaction states across every module.

## System layers

1. **Atmosphere** – fixed ocean/space background, restrained grid and cyan/violet light fields.
2. **Navigation shell** – translucent top bar and sidebar with strong active-route indication.
3. **Operational glass** – three elevations for rails, cards and modal command centers.
4. **Information hierarchy** – modern sans-serif for reading, monospace only for telemetry and evidence.
5. **Interaction language** – cyan focus, violet selection, green verified, amber warning and red critical.
6. **Module identity** – Global Watch remains geospatial cyan; SHADOW retains military gold; VGT Code uses cyan-gold engineering accents without breaking the shared shell.

## Performance and accessibility constraints

- Backdrop blur is limited to structural surfaces, not every list row.
- Text contrast remains readable over imagery.
- Focus-visible, hover, active and disabled states are mandatory.
- Reduced-motion users receive static light fields.
- Cesium and high-frequency visualizations keep their own rendering pipeline.

## Delivery

- Global design tokens and atmospheric background.
- Shared shell, cards, controls, tables, modals and scrollbars.
- Targeted Global Watch, SHADOW and VGT Code integration.
- Responsive fallback and production validation.
