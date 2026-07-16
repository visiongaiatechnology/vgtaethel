# Package main test compile fix proof

## Skeptic claim (stale)

`intelligence_integration_test.go:22` → `Newintelligence.IntelligenceStore`

## Reality (current tree)

```go
// line 19
store := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "del_case.json"))
// line 22
t.Fatal(err)
```

Zero remaining `Newintelligence` in `*.go` sources.

## Gating tests present

| Test | File |
|------|------|
| TestGlobeMathRuntimeFromShippedJS | globe_math_runtime_test.go |
| TestComputeGlobeFocusRotationEuropeNotAfrica | gating_focus_math_test.go |
| TestSphereBuildDocumentExportHTMLAndMarkdown | sphere_export_runtime_test.go |
| TestGlobalWatchChromeAndSphereAndNeuralCoreStructure | frontend_static_test.go |
| TestGlobalWatchExtendedCommandsNavigateRegionTimeReport | intelligence_integration_test.go |
| TestOSINTFrontendGlobeNoCDNAndInlinedMath | frontend_static_test.go |

## Run

```bat
scripts\run_gating_tests.bat
```
