@echo off
setlocal
cd /d "%~dp0.."
set GOEXE=C:\Program Files\Go\bin\go.exe
if not exist "%GOEXE%" set GOEXE=go
set LOG=%LOCALAPPDATA%\Temp\grok-goal-46f92dc627d7\implementer\go_test_run_live.log
if not exist "%LOCALAPPDATA%\Temp\grok-goal-46f92dc627d7\implementer" mkdir "%LOCALAPPDATA%\Temp\grok-goal-46f92dc627d7\implementer"

echo === compile package main tests === > "%LOG%"
"%GOEXE%" test -c -o NUL . >> "%LOG%" 2>&1
if errorlevel 1 (
  echo COMPILE FAILED >> "%LOG%"
  type "%LOG%"
  exit /b 1
)
echo COMPILE OK >> "%LOG%"

echo. >> "%LOG%"
echo === gated package main tests === >> "%LOG%"
"%GOEXE%" test . -count=1 -run "TestGlobeMathRuntimeFromShippedJS|TestSphereBuildDocumentExportHTMLAndMarkdown|TestGlobalWatchChromeAndSphereAndNeuralCoreStructure|TestGlobalWatchExtendedCommandsNavigateRegionTimeReport|TestOSINTFrontend" -v >> "%LOG%" 2>&1
set RC1=%ERRORLEVEL%

echo. >> "%LOG%"
echo === USGS intelligence test === >> "%LOG%"
"%GOEXE%" test ./intelligence/ -count=1 -run "TestUSGSEarthquake" -v >> "%LOG%" 2>&1
set RC2=%ERRORLEVEL%

echo. >> "%LOG%"
echo exit_main=%RC1% exit_usgs=%RC2% >> "%LOG%"
type "%LOG%"
exit /b %RC1%
