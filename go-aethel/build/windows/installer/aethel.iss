; STATUS: DIAMANT VGT SUPREME
#define AppName "VGT AETHEL"
#ifndef AppVersion
  #define AppVersion "1.0.0-beta.2"
#endif
#define AppPublisher "VisionGaiaTechnology"
#define AppExeName "AETHEL.exe"

[Setup]
AppId={{CE51F42C-038E-4B40-A8D5-4398587BC604}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
DefaultDirName={autopf}\VisionGaiaTechnology\AETHEL
DefaultGroupName={#AppName}
UninstallDisplayIcon={app}\{#AppExeName}
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=lowest
OutputDir=..\..\bin\installer
OutputBaseFilename=AETHEL-{#AppVersion}-windows-x64-setup
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
SetupIconFile=..\icon.ico
CloseApplications=yes
RestartApplications=no

[Files]
Source: "..\..\bin\AETHEL.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\bin\onnxruntime.dll"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\bin\sherpa-onnx-c-api.dll"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\bin\sherpa-onnx-cxx-api.dll"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExeName}"; WorkingDir: "{app}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExeName}"; WorkingDir: "{app}"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Desktop-Verknüpfung erstellen"; GroupDescription: "Zusätzliche Symbole:"

[Run]
Filename: "{app}\{#AppExeName}"; WorkingDir: "{app}"; Description: "{#AppName} starten"; Flags: nowait postinstall skipifsilent
