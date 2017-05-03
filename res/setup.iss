#define appId = "@APP_ID@"
#define appName "@APP_NAME@"
#define appVersion "@APP_VERSION@"
#define appPublisher "Cr@zy"
#define appURL "@APP_URL@"
#define currentYear GetDateTimeString('yyyy', '', '');

[Setup]
AppId={{2A8FE78D-9B0D-49A1-9DFA-8CCDD10336BC}
AppName={#appName}
AppVersion={#appVersion}
;AppVerName={#appName} {#appVersion}
AppPublisher={#appPublisher}
AppPublisherURL={#appURL}
AppSupportURL={#appURL}
AppUpdatesURL={#appURL}

WizardImageFile=setup.bmp
WizardSmallImageFile=setup-mini.bmp
DisableWelcomePage=no
ShowLanguageDialog=yes
LicenseFile=license.txt
SetupIconFile=slack-portable.ico

Compression=lzma/max
SolidCompression=yes

DefaultDirName={sd}\{#appId}
CreateAppDir=yes
Uninstallable=no

VersionInfoCompany={#appPublisher}
VersionInfoCopyright={#appPublisher} {#currentYear}
VersionInfoProductName={#appName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Files]
Source: "src\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[InstallDelete]
Type: filesandordirs; Name: "{app}\app-*"
Type: filesandordirs; Name: "{app}\logs\*"
Type: filesandordirs; Name: "{app}\packages"
Type: filesandordirs; Name: "{app}\SquirrelSetup.log"

[Run]
Filename: {app}\{#appId}.exe; Description: Run {#appName}; Flags: nowait postinstall skipifsilent

[Code]

var
  IsUpgrade: Boolean;

function isEmptyDir(dirName: String): Boolean;
var
  FindRec: TFindRec;
  FileCount: Integer;
begin
  Result := False;
  if FindFirst(dirName + '\*', FindRec) then begin
    try
      repeat
        if (FindRec.Name <> '.') and (FindRec.Name <> '..') then begin
          FileCount := 1;
          break;
        end;
      until not FindNext(FindRec);
    finally
      FindClose(FindRec);
      if FileCount = 0 then Result := True;
    end;
  end;
end;

function InitializeSetup: Boolean;
begin
  Result := True;
  IsUpgrade := False;
end;

function NextButtonClick(PageId: Integer): Boolean;
begin
  Result := True;
  if (PageId = wpSelectDir) then begin
    if DirExists(ExpandConstant('{app}')) and not isEmptyDir(ExpandConstant('{app}')) then begin
      IsUpgrade := True;
    end;
    if IsUpgrade and not FileExists(ExpandConstant('{app}\{#appId}.exe')) then begin
      MsgBox(ExpandConstant('The selected dir is not empty or you are performing an upgrade of {#appName} but {#appId}.exe is not found. Please select a correct folder.'), mbError, MB_OK);
      Result := False;
      exit;
    end;
  end;
end;
