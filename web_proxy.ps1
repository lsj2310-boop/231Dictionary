# Windows PowerShell 5.1: fallback through the current user's WinINET proxy.
# Input is a single UTF-8 JSON packet on stdin. Never write source text to disk.
$ErrorActionPreference = 'Stop'
try {
  $raw = [Console]::In.ReadToEnd()
  $packet = $raw | ConvertFrom-Json
  $headers = @{}
  foreach ($prop in $packet.headers.PSObject.Properties) { $headers[$prop.Name] = [string]$prop.Value }
  if ($headers.ContainsKey('Content-Type')) { $contentType = $headers['Content-Type'];$headers.Remove('Content-Type') } else { $contentType = 'application/x-www-form-urlencoded' }
  $body = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String([string]$packet.body))
  $params = @{Uri=[string]$packet.url;Method=[string]$packet.method;Headers=$headers;TimeoutSec=8;UseBasicParsing=$true}
  if ($packet.method -eq 'POST') { $params.Body = $body;$params.ContentType=$contentType }
  $response=Invoke-WebRequest @params
  $data= [Text.Encoding]::UTF8.GetBytes([string]$response.Content)
  [Console]::Out.WriteLine((@{status=[int]$response.StatusCode;base64=[Convert]::ToBase64String($data);error=''} | ConvertTo-Json -Compress))
} catch {
  $status=0; $content=''; $errorText=$_.Exception.Message
  if ($_.Exception.Response) {
    try { $status=[int]$_.Exception.Response.StatusCode; $stream=$_.Exception.Response.GetResponseStream(); if($stream){ $reader=New-Object IO.StreamReader($stream);$content=$reader.ReadToEnd() } } catch {}
  }
  $data=[Text.Encoding]::UTF8.GetBytes($content)
  [Console]::Out.WriteLine((@{status=$status;base64=[Convert]::ToBase64String($data);error=$errorText} | ConvertTo-Json -Compress))
}
