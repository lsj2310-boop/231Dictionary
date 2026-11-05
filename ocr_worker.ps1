# 231Dictionary v2.0 - Windows UI Automation + Windows OCR / optional Tesseract
# Local screen crops are NEVER uploaded. The caller deletes each crop when done.
# This script is compatible with Windows PowerShell 5.1; no additional PS modules needed.
$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
$script:lastOcrFailure = ''
$script:winReady = $false
$script:uiaReady = $false
$script:winError = ''
$script:tesseract = ''

try {
    Add-Type -AssemblyName UIAutomationClient
    Add-Type -AssemblyName UIAutomationTypes
    Add-Type -AssemblyName WindowsBase
    $script:uiaReady = $true
} catch { $script:uiaReady = $false }

try {
    $candidatePaths = @(
        (Join-Path $env:ProgramFiles 'Tesseract-OCR\tesseract.exe'),
        (Join-Path ${env:ProgramFiles(x86)} 'Tesseract-OCR\tesseract.exe'),
        (Join-Path $env:LOCALAPPDATA 'Programs\Tesseract-OCR\tesseract.exe')
    )
    $found = Get-Command tesseract.exe -ErrorAction SilentlyContinue
    if ($null -ne $found) { $script:tesseract = [string]$found.Source }
    if (-not $script:tesseract) {
        foreach ($candidate in $candidatePaths) {
            if ($candidate -and (Test-Path -LiteralPath $candidate -PathType Leaf)) {
                $script:tesseract = $candidate; break
            }
        }
    }
} catch { $script:tesseract = '' }

try {
    Add-Type -AssemblyName System.Runtime.WindowsRuntime
    $null = [Windows.Storage.StorageFile,Windows.Storage,ContentType=WindowsRuntime]
    $null = [Windows.Storage.Streams.IRandomAccessStream,Windows.Storage.Streams,ContentType=WindowsRuntime]
    $null = [Windows.Storage.FileAccessMode,Windows.Storage,ContentType=WindowsRuntime]
    $null = [Windows.Graphics.Imaging.BitmapDecoder,Windows.Graphics.Imaging,ContentType=WindowsRuntime]
    $null = [Windows.Graphics.Imaging.SoftwareBitmap,Windows.Graphics.Imaging,ContentType=WindowsRuntime]
    $null = [Windows.Media.Ocr.OcrEngine,Windows.Foundation,ContentType=WindowsRuntime]
    $null = [Windows.Media.Ocr.OcrResult,Windows.Foundation,ContentType=WindowsRuntime]
    $null = [Windows.Globalization.Language,Windows.Foundation,ContentType=WindowsRuntime]
    $script:asTask = [System.WindowsRuntimeSystemExtensions].GetMethods() |
        Where-Object { $_.Name -eq 'AsTask' -and $_.IsGenericMethod -and
            $_.GetParameters().Count -eq 1 -and
            $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1' } |
        Select-Object -First 1
    if ($null -eq $script:asTask) { throw 'WinRT AsTask converter not found' }
    $lang = [Windows.Globalization.Language]::new('en-US')
    $script:engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromLanguage($lang)
    if ($null -eq $script:engine) {
        $script:engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()
    }
    if ($null -eq $script:engine) { throw 'English Windows OCR language data unavailable' }
    $script:winReady = $true
} catch { $script:winError = [string]$_.Exception.Message }

function Await-WinRT($operation,[type]$resultType) {
    $method = $script:asTask.MakeGenericMethod($resultType)
    $task = $method.Invoke($null,@($operation))
    $task.GetAwaiter().GetResult()
}

function Normalize-Word([string]$s) {
    if ([string]::IsNullOrWhiteSpace($s)) { return '' }
    $s = ($s.Trim() -replace '^[^A-Za-z]+', '' -replace '[^A-Za-z]+$', '')
    if ($s -cmatch '^[A-Za-z][A-Za-z''-]{1,38}$') { return $s }
    return ''
}

# Pick only the word directly underneath (or very close to) the pointer.
# The original release required an unrealistic <=6px match and often returned nothing.
function Pick-BestWord($candidates,[double]$cx,[double]$cy) {
    $bestWord = ''; $bestDist = 999999.0; $num = 0
    foreach ($item in $candidates) {
        $w = Normalize-Word ([string]$item.text)
        if (-not $w) { continue }
        $num++
        $left = [double]$item.x
        $top = [double]$item.y
        $right = $left + [double]$item.w
        $bottom = $top + [double]$item.h
        $dx = [Math]::Max(0.0,[Math]::Max($left-$cx,$cx-$right))
        $dy = [Math]::Max(0.0,[Math]::Max($top-$cy,$cy-$bottom))
        $distance = [Math]::Sqrt($dx*$dx+$dy*$dy)
        if ($distance -lt $bestDist) { $bestDist = $distance; $bestWord = $w }
    }
    # 2x upscaled screenshot: 20px ~= 10 physical screen pixels.
    if ($bestDist -le 20.0) { return @{word=$bestWord; distance=[Math]::Round($bestDist,1); count=$num} }
    return @{word='';distance=[Math]::Round($bestDist,1);count=$num}
}

function Read-WindowsOCR([string]$path,[double]$cx,[double]$cy) {
    $script:lastOcrFailure = ''
    if (-not $script:winReady) { return @{word=''; count=0; distance=0} }
    $stream = $null; $bitmap = $null
    try {
        $file = Await-WinRT ([Windows.Storage.StorageFile]::GetFileFromPathAsync($path)) ([Windows.Storage.StorageFile])
        $stream = Await-WinRT ($file.OpenAsync([Windows.Storage.FileAccessMode]::Read)) ([Windows.Storage.Streams.IRandomAccessStream])
        $decoder = Await-WinRT ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream)) ([Windows.Graphics.Imaging.BitmapDecoder])
        $bitmap = Await-WinRT ($decoder.GetSoftwareBitmapAsync()) ([Windows.Graphics.Imaging.SoftwareBitmap])
        $result = Await-WinRT ($script:engine.RecognizeAsync($bitmap)) ([Windows.Media.Ocr.OcrResult])
        $candidates = New-Object System.Collections.ArrayList
        foreach ($line in $result.Lines) {
            foreach ($item in $line.Words) {
                $bounds = $item.BoundingRect
                [void]$candidates.Add(@{text=[string]$item.Text; x=$bounds.X; y=$bounds.Y; w=$bounds.Width; h=$bounds.Height})
            }
        }
        return Pick-BestWord $candidates $cx $cy
    } catch {
        $script:lastOcrFailure = [string]$_.Exception.Message
        return @{word=''; count=0; distance=0}
    } finally {
        if ($null -ne $bitmap) { try { $bitmap.Dispose() } catch {} }
        if ($null -ne $stream) { try { $stream.Dispose() } catch {} }
    }
}

function Read-Tesseract([string]$path,[double]$cx,[double]$cy) {
    if (-not $script:tesseract) { return @{word=''; count=0; distance=0} }
    try {
        # Sparse text is common in PPT slides; --psm 11 works better than --psm 6.
        $rows = & $script:tesseract $path stdout -l eng --psm 11 tsv 2>$null
        $candidates = New-Object System.Collections.ArrayList
        foreach ($row in ($rows | Select-Object -Skip 1)) {
            $f = ([string]$row).Split([char]9)
            if ($f.Count -lt 12 -or $f[0] -ne '5') { continue }
            $score = 0.0
            if (-not [double]::TryParse($f[10],[System.Globalization.NumberStyles]::Float,
                  [System.Globalization.CultureInfo]::InvariantCulture,[ref]$score) -or $score -lt 15) { continue }
            [void]$candidates.Add(@{text=$f[11];x=[double]$f[6];y=[double]$f[7];w=[double]$f[8];h=[double]$f[9]})
        }
        return Pick-BestWord $candidates $cx $cy
    } catch {
        $script:lastOcrFailure = 'Tesseract: ' + [string]$_.Exception.Message
        return @{word=''; count=0; distance=0}
    }
}

function Read-UIA([double]$sx,[double]$sy) {
    if (-not $script:uiaReady) { return '' }
    try {
        $p = New-Object System.Windows.Point($sx,$sy)
        $el = [System.Windows.Automation.AutomationElement]::FromPoint($p)
        $walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker
        for ($level=0; $level -lt 7 -and $null -ne $el; $level++) {
            try {
                $pat = $null
                if ($el.TryGetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern,[ref]$pat)) {
                    $range = $pat.RangeFromPoint($p)
                    if ($null -ne $range) {
                        $range.ExpandToEnclosingUnit([System.Windows.Automation.TextUnit]::Word)
                        $w = Normalize-Word ($range.GetText(-1))
                        if ($w) { return $w }
                    }
                }
            } catch {}
            $el = $walker.GetParent($el)
        }
    } catch {}
    return ''
}

function Emit-Answer($answer) {
    [Console]::Out.WriteLine(($answer | ConvertTo-Json -Compress -Depth 4))
    [Console]::Out.Flush()
}

# Direct-text selection only: UI Automation first, Ctrl+C text selection second.
# Clipboard fallback restores the original OLE IDataObject. No screen capture/OCR.
$script:selectionWinReady = $false
$script:lastSelectionError = ''
try {
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
using System.Runtime.InteropServices.ComTypes;
public static class SelectionWin32 {
    [DllImport("ole32.dll", PreserveSig=true)] public static extern int OleGetClipboard(out IDataObject data);
    [DllImport("ole32.dll", PreserveSig=true)] public static extern int OleSetClipboard(IDataObject data);
    [DllImport("user32.dll")] public static extern uint GetClipboardSequenceNumber();
    [DllImport("user32.dll", SetLastError=true)] public static extern uint EnumClipboardFormats(uint format);
    [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
    [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hwnd, out uint processId);
    [DllImport("user32.dll")] static extern void keybd_event(byte key, byte scan, uint flags, UIntPtr extra);
    public static void SendCtrlC() {
        keybd_event(0x11, 0, 0, UIntPtr.Zero);
        keybd_event(0x43, 0, 0, UIntPtr.Zero);
        keybd_event(0x43, 0, 2, UIntPtr.Zero);
        keybd_event(0x11, 0, 2, UIntPtr.Zero);
    }
}
'@ -ErrorAction Stop
    $script:selectionWinReady = $true
} catch { $script:lastSelectionError = [string]$_.Exception.Message }

# UIA selection is non-destructive; some viewers do not implement TextPattern.
function Read-SelectedText([double]$sx,[double]$sy) {
    if (-not $script:uiaReady) { return '' }
    try {
        $pt = New-Object System.Windows.Point($sx,$sy)
        $elements = @([System.Windows.Automation.AutomationElement]::FocusedElement,
                      [System.Windows.Automation.AutomationElement]::FromPoint($pt))
        foreach ($initial in $elements) {
            $el=$initial
            $walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker
            for ($i=0; $i -lt 13 -and $null -ne $el; $i++) {
                try {
                    $pattern = $null
                    if ($el.TryGetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern,[ref]$pattern)) {
                        $selected=$pattern.GetSelection()
                        if ($null -ne $selected) {
                            $parts=New-Object 'System.Collections.Generic.List[string]'
                            foreach ($r in $selected) {
                                if ($null -ne $r) {
                                    $t=[string]$r.GetText(14000)
                                    if (-not [string]::IsNullOrWhiteSpace($t)) { $parts.Add($t) }
                                }
                            }
                            if ($parts.Count -gt 0) { return ($parts -join "`n") }
                        }
                    }
                } catch {}
                $el=$walker.GetParent($el)
            }
        }
    } catch { $script:lastSelectionError = "UIA: " + [string]$_.Exception.Message }
    return ''
}

# Text fallback for PPT/PDF software that doesn't expose selection to UIA.
# Preserve ALL existing clipboard formats via original OLE IDataObject object,
# not via plain-text backup (which would destroy copied PowerPoint objects).
function Read-SelectedTextViaCopy([int64]$expectedHwnd) {
    if (-not $script:selectionWinReady) { return '' }
    $old = $null
    $changed=$false
    try {
        $hwnd=[SelectionWin32]::GetForegroundWindow()
        if ($expectedHwnd -le 0 -or $hwnd.ToInt64() -ne $expectedHwnd) {
            $script:lastSelectionError = 'Focus moved away from the originally selected window. Copy canceled.'
            return ''
        }
        $pidNumber=[uint32]0
        [void][SelectionWin32]::GetWindowThreadProcessId($hwnd,[ref]$pidNumber)
        if ($pidNumber -eq 0) { return '' }
        $processName=([Diagnostics.Process]::GetProcessById([int]$pidNumber)).ProcessName.ToLowerInvariant()
        $allow=@('powerpnt','acrord32','acrobat','foxitreader','foxitpdfreader','foxitpdfeditor',
                 'pdfxedit','pdfxcview','sumatrapdf','msedge','chrome','chromium',
                 'whale','naverwhale','brave','brave-browser','aside','aside-browser','asidebrowser',
                 'firefox','waterfox','opera','vivaldi','arc','zen','sider','thorium','winword','excel',
                 'soffice.bin','wps','et','hwp','hword')
        if ($allow -notcontains $processName) { return '' }
        $hr=[SelectionWin32]::OleGetClipboard([ref]$old)
        if ($hr -ne 0) {
            # An empty clipboard has nothing to preserve. For non-empty/locked
            # clipboards, refuse to copy rather than destroying user data.
            $f=[SelectionWin32]::EnumClipboardFormats(0)
            $winErr=[Runtime.InteropServices.Marshal]::GetLastWin32Error()
            if ($f -ne 0 -or $winErr -ne 0) {
                $script:lastSelectionError = "Failed to preserve clipboard (HRESULT $hr)"
                return ''
            }
            $old=$null
        }
        $before=[SelectionWin32]::GetClipboardSequenceNumber()
        [SelectionWin32]::SendCtrlC()
        $candidate=''
        for ($try=0; $try -lt 12; $try++) {
            [Threading.Thread]::Sleep(65)
            if ([SelectionWin32]::GetClipboardSequenceNumber() -ne $before) {
                $changed=$true
                try {
                    if ([System.Windows.Forms.Clipboard]::ContainsText()) {
                        $candidate=[System.Windows.Forms.Clipboard]::GetText([System.Windows.Forms.TextDataFormat]::UnicodeText)
                        if ($candidate) { break }
                    }
                } catch {}
            }
        }
        if (-not $changed) { return '' }
        if ([string]::IsNullOrWhiteSpace($candidate)) { return '' }
        return $candidate
    } catch { $script:lastSelectionError = "Failed to copy selected text: " + [string]$_.Exception.Message; return '' }
    finally {
        if ($changed) {
            try {
                $restore=[SelectionWin32]::OleSetClipboard($old)
                if ($restore -ne 0) {
                    $script:lastSelectionError="Failed to restore original clipboard (HRESULT $restore)"
                }
            } catch { $script:lastSelectionError="Original clipboard restoration error: " + [string]$_.Exception.Message }
        }
        if ($null -ne $old) {
            try { [void][Runtime.InteropServices.Marshal]::ReleaseComObject($old) } catch {}
        }
    }
}

# Recognize full lines inside the crop (rather than the single nearest word).
function Read-WindowsOCRText([string]$path) {
    $script:lastOcrFailure=''
    if (-not $script:winReady) { return '' }
    $stream=$null;$bitmap=$null
    try {
        $file = Await-WinRT ([Windows.Storage.StorageFile]::GetFileFromPathAsync($path)) ([Windows.Storage.StorageFile])
        $stream = Await-WinRT ($file.OpenAsync([Windows.Storage.FileAccessMode]::Read)) ([Windows.Storage.Streams.IRandomAccessStream])
        $decoder = Await-WinRT ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream)) ([Windows.Graphics.Imaging.BitmapDecoder])
        $bitmap = Await-WinRT ($decoder.GetSoftwareBitmapAsync()) ([Windows.Graphics.Imaging.SoftwareBitmap])
        $result = Await-WinRT ($script:engine.RecognizeAsync($bitmap)) ([Windows.Media.Ocr.OcrResult])
        $lines = New-Object System.Collections.Generic.List[string]
        foreach ($line in $result.Lines) {
            if (-not [string]::IsNullOrWhiteSpace([string]$line.Text)) { $lines.Add(([string]$line.Text).Trim()) }
        }
        return ($lines -join ' ')
    } catch {
        $script:lastOcrFailure = [string]$_.Exception.Message
        return ''
    } finally {
        if ($null -ne $bitmap) { try { $bitmap.Dispose() } catch {} }
        if ($null -ne $stream) { try { $stream.Dispose() } catch {} }
    }
}
function Read-TesseractText([string]$path) {
    if (-not $script:tesseract) { return '' }
    try {
        $r = & $script:tesseract $path stdout -l eng --psm 6 2>$null
        return (($r | Where-Object {$_ -and ([string]$_).Trim()}) -join ' ').Trim()
    } catch { return '' }
}

while ($null -ne ($line = [Console]::ReadLine())) {
    if ([string]::IsNullOrWhiteSpace($line)) { continue }
    try {
        $req = ConvertFrom-Json -InputObject $line
        $avail = 'winOCR=' + $script:winReady + '; tess=' + [bool]($script:tesseract) + '; UIA=' + $script:uiaReady
        if ($req.warmup -eq $true) {
            Emit-Answer @{ok=$true;word='';backend='warmup';available=$avail;ocr=$script:winReady;ocrError=$script:winError}
            continue
        }
        $path = [string]$req.path
        if ([string]$req.mode -eq 'selection') {
            $script:lastSelectionError = ''
            $expectedWindow=[int64]$req.foregroundHwnd
            if ($expectedWindow -le 0 -or [SelectionWin32]::GetForegroundWindow().ToInt64() -ne $expectedWindow) {
                Emit-Answer @{ok=$false;sentence='';backend='selection';error='Selection canceled because the active window changed.'}
                continue
            }
            $sentence=Read-SelectedText ([double]$req.sx) ([double]$req.sy)
            $method='uia-selection'
            if (-not $sentence) {
                $sentence=Read-SelectedTextViaCopy $expectedWindow
                $method='selected-text-copy'
            }
            Emit-Answer @{ok=$true;sentence=[string]$sentence;backend=$method;ocr=$false;
                ocrError='';available=$avail;error=$script:lastSelectionError}
            continue
        }
        $cx = [double]$req.x; $cy = [double]$req.y
        $word = ''; $backend='none'; $detected=0; $distance=0
        if ($path -and (Test-Path -LiteralPath $path -PathType Leaf)) {
            $a = Read-WindowsOCR $path $cx $cy
            $detected = $a.count; $distance = $a.distance
            if ($a.word) { $word=$a.word; $backend='windows-ocr' }
            if (-not $word) {
                $a = Read-Tesseract $path $cx $cy
                if ($a.count -gt $detected) { $detected=$a.count; $distance=$a.distance }
                if ($a.word) { $word=$a.word; $backend='tesseract' }
            }
        }
        if (-not $word -and $req.selftest -ne $true) {
            $word = Read-UIA ([double]$req.sx) ([double]$req.sy)
            if ($word) { $backend='uia' }
        }
        $why = $script:winError
        if ($script:lastOcrFailure) { $why = $script:lastOcrFailure }
        Emit-Answer @{ok=$true;word=[string]$word;backend=$backend;ocr=$script:winReady;
            ocrError=$why;available=$avail;count=$detected;distance=$distance}
    } catch {
        Emit-Answer @{ok=$false;word='';backend='error';error=[string]$_.Exception.Message}
    }
}
