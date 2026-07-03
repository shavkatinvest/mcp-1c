@echo off
chcp 65001 >nul
setlocal

:: Проверяет исходники расширения (extension/src) через BSL Language Server
:: (https://github.com/1c-syntax/bsl-language-server) — находит синтаксические
:: ошибки, устаревшие конструкции и проблемы качества кода в .bsl-модулях
:: без запуска Конфигуратора.
::
:: При первом запуске сам скачивает bsl-language-server в tools-external/
:: (см. scripts/setup-bsl-language-server.ps1).
::
:: Использование:
::   scripts\lint-bsl.cmd
::   scripts\lint-bsl.cmd console
::   scripts\lint-bsl.cmd json tools-external\bsl-report

set "SCRIPT_DIR=%~dp0"
set "PROJECT_DIR=%SCRIPT_DIR%.."
set "SRC_DIR=%PROJECT_DIR%\extension\src"
set "BSL_LS=%PROJECT_DIR%\tools-external\bsl-language-server\bsl-language-server\bsl-language-server.exe"

set "REPORTER=%~1"
if "%REPORTER%"=="" set "REPORTER=console"
set "OUT_DIR=%~2"
if "%OUT_DIR%"=="" set "OUT_DIR=%PROJECT_DIR%\tools-external\bsl-report"

if not exist "%BSL_LS%" (
    echo BSL Language Server не найден, устанавливаю...
    powershell -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT_DIR%setup-bsl-language-server.ps1"
    if errorlevel 1 exit /b 1
)

echo Анализ %SRC_DIR% ^(reporter=%REPORTER%^)...
"%BSL_LS%" analyze -s "%SRC_DIR%" -r %REPORTER% -o "%OUT_DIR%"
