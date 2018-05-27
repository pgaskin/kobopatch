@ECHO off

CD %~dp0

REM ----------  set project name
REM if projectname is empty, the filename of the current batch file is used instead
SET projectname=4.8.11073
REM SET projectname=
IF "X%projectname%"=="X" (SET projectname=%~n0)

SETLOCAL enableextensions
echo off
set str1="%~dp0"
if not x%str1:)=%==x%str1% GOTO errorINPATHNAME
ENDLOCAL



SETLOCAL EnableDelayedExpansion
SET the7zipexe=%CD%\tools\7za920\7za.exe
SET thepatchexe=%CD%\tools\pa32lsb.exe
SET currentpath=%CD%
SET sourcepath=%CD%\%projectname%_source
SET targetpath=%CD%\%projectname%_target
SET thekobopath=%targetpath%\usr\local\Kobo

SET debuglog=false
IF "%debuglog%"=="true" (
SET debugfile=%currentpath%\debug.log
ECHO project name = %projectname% > !debugfile!
ECHO the7zipexe=%CD%\tools\7za920\7za.exe >> !debugfile!
ECHO thepatchexe=%CD%\tools\pa32lsb.exe >> !debugfile!
ECHO currentpath=%CD% >> !debugfile!
ECHO sourcepath=%CD%\%projectname%_source  >> !debugfile!
ECHO targetpath=%CD%\%projectname%_target  >> !debugfile!
ECHO thekobopath=%targetpath%\usr\local\Kobo  >> !debugfile!
)

IF NOT EXIST "%sourcepath%\" (
GOTO errorNOSOURCEPATH
)
cd "%sourcepath%"

REM ---------- check for arguments
FOR %%i IN (%*) DO (
  IF "%%i"=="restore" (SET restore=true) 
)

REM -------------  check whether all files to patch are available
FOR %%i IN ("%sourcepath%\*.patch") DO (
  CALL:getfiletopatch "%%~ni"
  IF NOT !existsfiletopath!==true (
   ECHO ERROR Cannot find or extract "%%~ni"
   GOTO errorMISSINGSOURCEFILE     
  )
)

REM ------------ clean up 
DEL /s /f /q "%targetpath%" 2> nul
RD /S /Q "%targetpath%" 2> nul
MD "%thekobopath%"

REM ---------- copy files in order to restore original state
IF "!restore!"=="true" (
 FOR %%i IN ("%sourcepath%\*.patch") DO (
  SET patchfile=%%~i
  SET inputfile=%sourcepath%\%%~ni
  COPY "!inputfile!" "%thekobopath%\"
  IF ERRORLEVEL 1 (
     SET errorf=%%i
    GOTO errorCOPY
  )
 GOTO afterPatching
 )
)

REM ------------- patch files
FOR %%i IN ("%sourcepath%\*.patch") DO (
  SET patchfile=%%~i
  SET inputfile=%sourcepath%\%%~ni
  SET outputfile=%thekobopath%\%%~ni
if exist "!inputfile!" (
  ECHO //////////////////////////////////////////////
  ECHO //                    Patching start
  ECHO //////////////////////////////////////////////
  "%thepatchexe%" -p "!patchfile!" -i "!inputfile!" -o "!outputfile!"
  IF ERRORLEVEL 1 (
     SET errorf=%%i
    GOTO errorPATCH
  )
  ECHO //////////////////////////////////////////////
  ECHO //                    Patching end
  ECHO //////////////////////////////////////////////
)
)
:afterPatching

CD "%targetpath%"

REM ------------- compress, if there is something worth compressing
set _TMP=
for /f "delims=" %%a in ('dir /b "%thekobopath%"') do set _TMP=%%a
IF {%_TMP%}=={} (
       SET KoboRootCreated=false
) ELSE (
 REM --- clean up
 IF EXIST KoboRoot.tar DEL KoboRoot.tar
 IF EXIST KoboRoot.tgz DEL KoboRoot.tgz
 REM ------------- make tar
 "%the7zipexe%" a KoboRoot.tar -ttar  * -r
 IF ERRORLEVEL 1 GOTO errorTAR
 REM ------------- make tgz
 "%the7zipexe%" a KoboRoot.tgz -tgzip  KoboRoot.tar
 IF ERRORLEVEL 1 GOTO errorTGZ
 SET KoboRootCreated=true
)

REM ------------- clean up
IF EXIST KoboRoot.tar DEL KoboRoot.tar
RD /S /Q usr

IF %KoboRootCreated%==false (GOTO errorNORESULT)
GOTO byby     

REM ------------- handle errors

:errorINPATHNAME
ECHO The path name must not contain ")"!
SET problems=true
GOTO byby



:errorNORESULT
ECHO No result!
SET problems=true
GOTO byby

:errorMISSINGSOURCEFILE
ECHO Did you copy kobo-update-X.X.X.zip to %sourcepath%?
SET problems=true
GOTO byby

:errorNOSOURCEPATH
ECHO ERROR: cannot find %sourcepath%
SET problems=true
GOTO byby

:errorUNPACK
ECHO ERROR: problems while unpacking
ECHO Did you copy kobo-update-X.X.X.zip to %sourcepath%?
SET problems=true
GOTO byby


:errorPATCH
ECHO ERROR processing %errorf%
SET problems=true
GOTO byby

:errorCOPY
ECHO ERROR cannot copy %errorf%
SET problems=true
GOTO byby

:errorTAR
ECHO ERROR while creating KoboRoot.tar
SET problems=true
GOTO byby

:errorTGZ
ECHO ERROR while creating KoboRoot.tgz
SET problems=true
GOTO byby

:byby
CD "%currentpath%"
IF  "%problems%" NEQ "true" (
ECHO -------------------------------------------------------
ECHO SUMMARY: Everything seems to be O.K.!
ECHO -------------------------------------------------------
)
PAUSE
GOTO:eof

REM --------- Function
:getfiletopatch
IF EXIST %~1 (
        SET existsfiletopath=true
        GOTO:eof 
)
IF EXIST KoboRoot.tar GOTO untarFiletopatch
IF EXIST KoboRoot.tgz GOTO ungzipKoboRoot
"%the7zipexe%" e -y *.zip KoboRoot.tgz
IF NOT EXIST KoboRoot.tgz (
 SET existsfiletopath=false
 GOTO:eof 
)
:ungzipKoboRoot
"%the7zipexe%" e -y KoboRoot.tgz
DEL KoboRoot.tgz
:untarFiletopatch
"%the7zipexe%" e -r -y -ttar KoboRoot.tar %~1
IF EXIST %~1 (
 SET existsfiletopath=true
) ELSE (
 SET existsfiletopath=false
)
GOTO:eof
