param([Parameter(Mandatory=$true)][Int64]$Hwnd)
# Windows PowerShell 5.1 UI Automation probe. Returns PDF or WEB only.
# Reads only browser accessibility metadata (title/address/PDF-viewer controls);
# never copies selected document text, never stores URLs, never makes requests.
$ErrorActionPreference = 'Stop'
try {
    Add-Type -AssemblyName UIAutomationClient
    Add-Type -AssemblyName UIAutomationTypes
    $root = [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]$Hwnd)
    if ($null -eq $root) { 'WEB'; exit }
    $edits = $root.FindAll([System.Windows.Automation.TreeScope]::Descendants,
        ([System.Windows.Automation.PropertyCondition]::new(
            [System.Windows.Automation.AutomationElement]::ControlTypeProperty,
            [System.Windows.Automation.ControlType]::Edit)))
    $limit=[Math]::Min(90, $edits.Count)
    for($i=0; $i -lt $limit; $i++) {
        $el=$edits.Item($i)
        $name=[string]$el.Current.Name
        # Only read the browser's omnibox, not form fields from web pages.
        if ($name -notmatch '(?i)address|search bar|주소|주소창|주소 및 검색') { continue }
        $pat=$null
        if (-not $el.TryGetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern,[ref]$pat)) { continue }
        $value=[string]$pat.Current.Value
        try {
            $u=[uri]$value
            $path=[uri]::UnescapeDataString($u.AbsolutePath)
            if ($path -match '(?i)\.pdf$') { 'PDF'; exit }
            $q=$u.Query.TrimStart('?') -split '&'
            foreach($pair in $q) {
                $parts=$pair -split '=', 2
                if ($parts.Count -lt 2 -or $parts[0] -notmatch '(?i)^(file|url|document|src|pdf)$') { continue }
                $candidate=[uri]::UnescapeDataString($parts[1])
                $candidate=($candidate -split '[?#]',2)[0]
                if ($candidate -match '(?i)\.pdf$') { 'PDF'; exit }
            }
        } catch {}
    }
    # Explicit built-in PDF viewer accessibility labels, not generic page copy.
    foreach($name in @('PDF Viewer', 'PDF document', 'PDF 뷰어', 'PDF 문서')) {
        $condition=[System.Windows.Automation.PropertyCondition]::new([System.Windows.Automation.AutomationElement]::NameProperty,$name)
        $found=$root.FindFirst([System.Windows.Automation.TreeScope]::Descendants,$condition)
        if($null -ne $found) { 'PDF'; exit }
    }
} catch {}
'WEB'
