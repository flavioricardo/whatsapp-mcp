# Watchdog for the WhatsApp bridge on Windows: restarts it when it dies, logs its output
# to logs\bridge-YYYY-MM-DD.log (its own events to logs\watchdog.log) and touches
# logs\heartbeat.txt every minute while it runs.
# Install/uninstall as a logon task: see "Keep the bridge running on Windows" in the README.
param(
    [string]$Dir = $PSScriptRoot,              # folder with the exe and store\ (paths are relative)
    [string]$Exe = 'whatsapp-bridge-bg.exe'
)

$logs = Join-Path $Dir 'logs'
$heartbeat = Join-Path $logs 'heartbeat.txt'
New-Item -ItemType Directory -Force $logs | Out-Null

# One watchdog only: two bridges on the same device kick each other off WhatsApp
$mutex = New-Object System.Threading.Mutex($false, 'Global\whatsapp-bridge-watchdog')
try { if (-not $mutex.WaitOne(0)) { exit 0 } } catch [System.Threading.AbandonedMutexException] {}

# Own file: while a bridge runs, cmd's redirect keeps its log locked against other writers
function Write-Log($msg) { Add-Content (Join-Path $logs 'watchdog.log') "$(Get-Date -Format s) $msg" }

while ($true) {
    # ponytail: file is named after the day the bridge started, not rotated while it runs
    $log = Join-Path $logs "bridge-$(Get-Date -Format yyyy-MM-dd).log"

    # A bridge started by hand is waited on, not duplicated
    $p = Get-Process -Name whatsapp-bridge, whatsapp-bridge-bg -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($p) {
        Write-Log "bridge already running (pid $($p.Id)), waiting for it to exit"
    } else {
        Write-Log 'starting bridge'
        # cmd does the redirect: raw bytes, stdout and stderr in one file (PowerShell 5.1 would re-encode)
        $p = Start-Process cmd.exe -ArgumentList "/c `"`"$Dir\$Exe`" >> `"$log`" 2>&1`"" `
            -WorkingDirectory $Dir -WindowStyle Hidden -PassThru
    }
    # Without a handle WaitForExit throws and the loop below would spin with a fake heartbeat
    # (exe missing mid-rebuild, or a bridge started elevated); Handle also keeps ExitCode readable
    try { $null = $p.Handle } catch {
        Write-Log "can't start or watch the bridge: $_"
        Start-Sleep 60
        continue
    }

    # ponytail: says the process is alive, not that it is connected to WhatsApp
    do {
        Set-Content $heartbeat (Get-Date -Format s) -ErrorAction SilentlyContinue  # a reader may hold it open
    } until ($p.WaitForExit(60000))

    Write-Log "bridge exited (code $($p.ExitCode)), restarting in 15s"
    Start-Sleep 15
}
