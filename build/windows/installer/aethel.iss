; STATUS: DIAMANT VGT SUPREME
#define AppName "VGT AETHEL"
#define AppVersion "0.9.0-alpha"
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
Source: "..\..\bin\*.dll"; DestDir: "{app}"; Flags: ignoreversion skipifsourcedoesntexist

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExeName}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExeName}"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Desktop-Verknüpfung erstellen"; GroupDescription: "Zusätzliche Symbole:"

[Run]
Filename: "{app}\{#AppExeName}"; Description: "{#AppName} starten"; Flags: nowait postinstall skipifsilent
