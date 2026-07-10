@echo off
echo ===================================================
echo  VGT AETHEL - BUILD SYSTEM (WAILS + CGO)
echo ===================================================
echo.
echo Aktiviere CGO fuer Sherpa-ONNX...
set CGO_ENABLED=1

echo Bereinige go.mod dependencies...
go mod tidy

echo Starte Wails Build...
wails build

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo [FEHLER] Wails Build ist fehlgeschlagen.
    pause
    exit /b %ERRORLEVEL%
)

echo.
echo ===================================================
echo  BUILD ERFOLGREICH ABGESCHLOSSEN!
echo  Die aethel.exe befindet sich im build/bin-Ordner.
echo ===================================================
pause
