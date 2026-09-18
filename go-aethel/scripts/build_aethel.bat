@echo off
echo ===================================================
echo  VGT AETHEL - BUILD SYSTEM (WAILS + CGO)
echo ===================================================
echo.
echo Aktiviere CGO fuer Sherpa-ONNX...
set CGO_ENABLED=1
set GOTOOLCHAIN=go1.26.5+auto

echo Bereinige go.mod dependencies...
go mod tidy

echo Starte archivierenden, nicht-destruktiven Wails Build...
echo [SCHUTZ] Vorhandene EXE-Dateien werden vor Wails ausserhalb von build\bin gespiegelt.
echo [SCHUTZ] build\bin\vgt_workspace wird weder bereinigt noch verschoben.
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build_preserve.ps1" -OutputName "AETHEL.exe"

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [FEHLER] Wails Build ist fehlgeschlagen.
    pause
    exit /b %ERRORLEVEL%
)

echo Kopiere native Runtime-Abhaengigkeiten...
for %%F in (onnxruntime.dll sherpa-onnx-c-api.dll sherpa-onnx-cxx-api.dll) do (
    if not exist "%%F" (
        echo [FEHLER] Erforderliche Runtime-Datei fehlt: %%F
        exit /b 1
    )
    copy /Y "%%F" "build\bin\%%F" >nul
)

echo.
echo ===================================================
echo  BUILD ERFOLGREICH ABGESCHLOSSEN!
echo  Die aethel.exe befindet sich im build/bin-Ordner.
echo ===================================================
pause
