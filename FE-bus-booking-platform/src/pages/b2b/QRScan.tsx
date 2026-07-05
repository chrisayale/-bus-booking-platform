import { useState, useEffect, useRef, useCallback } from "react"
import { Html5Qrcode } from "html5-qrcode"
import { api } from "../../api/client"
import { CheckCircle, XCircle, Camera, Smartphone, ScanLine, RotateCcw, Upload, ArrowRight, User, Armchair } from "lucide-react"

// Try to decode a QR token — supports both new base64url and legacy hex encoding.
function decodeQRToken(token: string): { token: string; journey_id: string } | null {
  try {
    // Try base64url decode first (new format)
    const raw = atob(token.replace(/-/g, "+").replace(/_/g, "/"))
    const data = JSON.parse(raw)
    if (data.journey_id) {
      return { token, journey_id: data.journey_id }
    }
  } catch { /* Not base64 — try legacy hex */ }
  try {
    // Legacy hex format: try hex decode -> JSON parse
    const raw = token.match(/.{1,2}/g)?.map((b) => String.fromCharCode(parseInt(b, 16))).join("")
    if (raw) {
      const data = JSON.parse(raw)
      if (data.journey_id) {
        return { token, journey_id: data.journey_id }
      }
    }
  } catch { /* Not parseable, return unknown journey */ }
  return null
}

export default function QRScan() {
  const [qrToken, setQrToken] = useState("")
  const [journeyId, setJourneyId] = useState("")
  const [result, setResult] = useState<{ success: boolean; message: string } | null>(null)
  const [loading, setLoading] = useState(false)
  const [scanning, setScanning] = useState(false)
  const [scanFeedback, setScanFeedback] = useState<string | null>(null)
  const [scanStatus, setScanStatus] = useState<"inactive" | "waiting" | "scanning">("inactive")
  const [scannedPassenger, setScannedPassenger] = useState<{ name: string; seat: string } | null>(null)
  const scannerRef = useRef<Html5Qrcode | null>(null)
  const processingRef = useRef(false)
  const scanFailedRef = useRef(false)
  const decodeAndScanRef = useRef<((text: string) => void) | null>(null)
  const scannerDivId = "qr-scanner-element"

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (scannerRef.current) {
        scannerRef.current.stop().catch(() => {})
      }
    }
  }, [])

  const resetScan = useCallback(() => {
    setScannedPassenger(null)
    setResult(null)
    setQrToken("")
    setJourneyId("")
  }, [])

  const stopScanner = useCallback(async () => {
    if (scannerRef.current) {
      try {
        await scannerRef.current.stop()
      } catch { /* scanner may already be stopped */ }
      scannerRef.current = null
    }
    setScanning(false)
    setScanFeedback(null)
    setScanStatus("inactive")
  }, [])

  const handleScanWithData = useCallback(async (token: string, jId: string) => {
    processingRef.current = true
    setLoading(true)
    setResult(null)
    try {
      const res = await api.post<{ message: string; passenger: string; seat: string }>("/scan", {
        qr_token: token,
        journey_id: jId,
      })
      setScannedPassenger({ name: res.passenger, seat: res.seat })
      setResult({ success: true, message: `${res.passenger} - Seat ${res.seat}: ${res.message}` })
      setScanStatus("scanning")
      // Clear manual entry fields to prevent re-submitting an already-validated ticket
      setQrToken("")
      setJourneyId("")
      // Only stop scanner on SUCCESS — user can restart manually for next ticket
      stopScanner()
    } catch (err) {
      const msg = (err as Error).message || "Unknown error scanning ticket"
      setResult({ success: false, message: msg })
      // Keep scanner running on failure so user can retry without restarting camera
      setScanStatus("waiting")
      setScanFeedback("Scan failed — adjust the QR code and try again")
    } finally {
      setLoading(false)
      processingRef.current = false
    }
  }, [stopScanner])

  // decodeAndScan: decode token, extract journey_id, send to backend
  const decodeAndScan = useCallback((decodedText: string) => {
    if (processingRef.current) return

    setScanStatus("scanning")

    // Reset the scan-failed flag so user gets feedback on next failure
    scanFailedRef.current = false

    const decoded = decodeQRToken(decodedText)
    if (decoded) {
      setScanFeedback("QR code detected! Validating...")
      setQrToken(decoded.token)
      setJourneyId(decoded.journey_id)
      handleScanWithData(decoded.token, decoded.journey_id)
    } else {
      // Cannot decode journey_id — use raw token with manually entered journey
      setScanFeedback("QR format not recognized — using manual entry fallback")
      setQrToken(decodedText)
      handleScanWithData(decodedText, journeyId)
    }
  }, [journeyId, handleScanWithData])

  // Keep ref in sync so scanner always calls the latest decodeAndScan
  useEffect(() => {
    decodeAndScanRef.current = decodeAndScan
  }, [decodeAndScan])

  const startScanner = async () => {
    // Prevent starting multiple scanners
    if (scanning || processingRef.current) return

    // Reset scan-failed flag for the new session
    scanFailedRef.current = false

    setScanning(true)
    setResult(null)
    setScanFeedback("Starting camera...")

    try {
      // Clean up any existing scanner first
      if (scannerRef.current) {
        try {
          await scannerRef.current.stop()
        } catch { /* scanner may already be stopped */ }
      }

      const scanner = new Html5Qrcode(scannerDivId)
      scannerRef.current = scanner

      const qrConfig = {
        fps: 10,
        // Responsive qrbox — use up to 80% of container, capped at 300px
        qrbox: (viewfinderWidth: number, viewfinderHeight: number) => ({
          width: Math.min(300, viewfinderWidth * 0.8),
          height: Math.min(300, viewfinderHeight * 0.8),
        }),
      }

      await scanner.start(
        { facingMode: "environment" },
        qrConfig,
        (decodedText) => {
          decodeAndScanRef.current?.(decodedText)
        },
        () => {
          // This fires on every frame where a QR code isn't decoded (up to 10/sec)
          // Only show once per scan session to avoid UI flicker
          if (!scanFailedRef.current) {
            scanFailedRef.current = true
            setScanFeedback("Looking for QR code — point your camera at a ticket")
          }
        }
      )
      setScanStatus("waiting")
      setScanFeedback("Camera ready — point at a QR ticket")
    } catch (err) {
      setResult({ success: false, message: "Camera not available or permission denied. " + (err as Error).message })
      setScanning(false)
      setScanFeedback(null)
      setScanStatus("inactive")
      scannerRef.current = null
    }
  }

  const handleManualScan = async (e: React.FormEvent) => {
    e.preventDefault()
    if (processingRef.current) return
    processingRef.current = true
    setLoading(true)
    setResult(null)
    try {
      const res = await api.post<{ message: string; passenger: string; seat: string }>("/scan", {
        qr_token: qrToken,
        journey_id: journeyId,
      })
      setScannedPassenger({ name: res.passenger, seat: res.seat })
      setResult({ success: true, message: `${res.passenger} - Seat ${res.seat}: ${res.message}` })
      // Clear fields after successful manual validation to prevent re-submitting the same ticket
      setQrToken("")
      setJourneyId("")
    } catch (err) {
      const msg = (err as Error).message || "Unknown error"
      setResult({ success: false, message: msg })
    } finally {
      setLoading(false)
      processingRef.current = false
    }
  }

  // Ring color based on scanStatus
  const ringClass = scanStatus === "scanning"
    ? "ring-4 ring-green-500 ring-offset-2"
    : scanStatus === "waiting"
      ? "ring-4 ring-red-500 ring-offset-2"
      : ""

  // Corner bracket color
  const bracketColor = scanStatus === "scanning" ? "border-green-500" : scanStatus === "waiting" ? "border-red-500" : "border-travel-500"

  // Scan line gradient
  const lineGradient = scanStatus === "scanning"
    ? "from-transparent via-green-400 to-transparent"
    : scanStatus === "waiting"
      ? "from-transparent via-red-400 to-transparent"
      : "from-transparent via-travel-400 to-transparent"

  // Scan line shadow
  const lineShadow = scanStatus === "scanning"
    ? "shadow-lg shadow-green-400/50"
    : scanStatus === "waiting"
      ? "shadow-lg shadow-red-400/50"
      : "shadow-lg shadow-travel-300/50"

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">QR Ticket Scanner</h1>

      <div className="grid md:grid-cols-2 gap-8">
        {/* Camera Scanner */}
        <div className="bg-white p-6 rounded-xl border border-gray-200">
          <div className="flex items-center gap-2 mb-4">
            <Camera className="h-5 w-5 text-travel-600" />
            <h2 className="text-lg font-semibold">Camera Scanner</h2>
          </div>

          {/* Status indicator bar */}
          {scanning && (
            <div className={`mb-3 flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium ${
              scanStatus === "scanning"
                ? "bg-green-50 text-green-700"
                : scanStatus === "waiting"
                  ? "bg-red-50 text-red-700"
                  : "bg-gray-50 text-gray-500"
            }`}>
              <span className={`relative flex h-2 w-2 ${scanStatus === "scanning" ? "animate-ping" : ""}`}>
                <span className={`absolute inline-flex h-full w-full rounded-full opacity-75 ${
                  scanStatus === "scanning" ? "bg-green-400" : scanStatus === "waiting" ? "bg-red-400" : "bg-gray-400"
                }`} />
              </span>
              <span className={`relative inline-flex h-2 w-2 rounded-full ${
                scanStatus === "scanning" ? "bg-green-500" : scanStatus === "waiting" ? "bg-red-500" : "bg-gray-400"
              }`} />
              {scanStatus === "scanning" ? "Scanning..." : scanStatus === "waiting" ? "Waiting for QR code..." : ""}
            </div>
          )}

          {/* Camera view with dynamic ring */}
          <div className={`relative w-full aspect-square bg-gray-100 rounded-xl overflow-hidden transition-all duration-300 ${ringClass}`}>
            {/* Success overlay — shown after successful scan */}
            {!scanning && scannedPassenger && (
              <div className="absolute inset-0 z-20 flex flex-col items-center justify-center bg-gradient-to-br from-green-50 via-emerald-50 to-teal-50 animate-fade-in">
                {/* Success checkmark circle */}
                <div className="w-24 h-24 bg-gradient-to-br from-green-400 to-emerald-500 rounded-full flex items-center justify-center mb-5 shadow-xl shadow-green-200/60 animate-bounce-gentle">
                  <CheckCircle className="h-12 w-12 text-white" />
                </div>
                <h3 className="text-2xl font-bold text-green-700 mb-1">Check-In Successful!</h3>
                <div className="mt-4 p-4 bg-white/80 backdrop-blur-sm rounded-xl border border-green-200 shadow-sm w-64 text-center">
                  <div className="flex items-center justify-center gap-2 mb-2">
                    <User className="h-4 w-4 text-green-600" />
                    <span className="text-base font-semibold text-gray-800">{scannedPassenger.name}</span>
                  </div>
                  <div className="flex items-center justify-center gap-2">
                    <Armchair className="h-4 w-4 text-green-600" />
                    <span className="text-base font-semibold text-gray-800">Seat {scannedPassenger.seat}</span>
                  </div>
                </div>
              </div>
            )}

            {/* Placeholder overlay — shown when NOT scanning and no success */}
            {!scanning && !scannedPassenger && (
              <div className="absolute inset-0 flex items-center justify-center z-10 bg-gray-50/80">
                <div className="text-center text-gray-400 p-4">
                  <Smartphone className="h-16 w-16 mx-auto mb-2" />
                  <p className="text-sm">Point your camera at the QR ticket</p>
                </div>
              </div>
            )}

            {/* Scanner frame overlay — shown when scanning */}
            {scanning && (
              <>
                {/* Corner brackets */}
                <div className="absolute inset-0 z-10 pointer-events-none">
                  {/* Top-left */}
                  <div className={`absolute top-4 left-4 w-8 h-8 border-t-2 border-l-2 ${bracketColor} rounded-tl-lg transition-colors duration-300`} />
                  {/* Top-right */}
                  <div className={`absolute top-4 right-4 w-8 h-8 border-t-2 border-r-2 ${bracketColor} rounded-tr-lg transition-colors duration-300`} />
                  {/* Bottom-left */}
                  <div className={`absolute bottom-4 left-4 w-8 h-8 border-b-2 border-l-2 ${bracketColor} rounded-bl-lg transition-colors duration-300`} />
                  {/* Bottom-right */}
                  <div className={`absolute bottom-4 right-4 w-8 h-8 border-b-2 border-r-2 ${bracketColor} rounded-br-lg transition-colors duration-300`} />

                  {/* Scanning line animation */}
                  <div className={`absolute left-6 right-6 h-0.5 bg-gradient-to-r ${lineGradient} animate-scan-line ${lineShadow} transition-all duration-300`} />
                </div>

                {/* Feedback text */}
                {scanFeedback && (
                  <div className="absolute bottom-4 left-4 right-4 z-20 text-center">
                    <span className="inline-block bg-black/60 backdrop-blur-sm text-white text-xs px-3 py-1.5 rounded-full">
                      <ScanLine className="h-3 w-3 inline mr-1 -mt-0.5 animate-pulse" />
                      {scanFeedback}
                    </span>
                  </div>
                )}
              </>
            )}

            {/* Scanner element - ALWAYS empty of React children (Html5Qrcode manages its own DOM) */}
            <div id={scannerDivId} className="w-full h-full" />
          </div>

          <button
            onClick={scanning ? stopScanner : startScanner}
            className={`mt-4 w-full py-3 rounded-lg text-sm font-medium transition ${
              scanning
                ? "bg-red-500 text-white hover:bg-red-600"
                : "bg-travel-600 text-white hover:bg-travel-700"
            }`}
          >
            {scanning ? "Stop Camera" : "Start Camera"}
          </button>

          {/* Error result — only shown when scan failed */}
          {result && !result.success && (
            <div className="mt-4 p-4 rounded-lg flex items-start gap-3 bg-red-50 text-red-700">
              <XCircle className="h-5 w-5 mt-0.5 shrink-0" />
              <div className="text-sm flex-1">
                <p>{result.message}</p>
                <button
                  onClick={() => setResult(null)}
                  className="mt-1 text-xs text-red-500 hover:text-red-700 flex items-center gap-1"
                >
                  <RotateCcw className="h-3 w-3" /> Dismiss & try again
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Manual Entry */}
        <div className="bg-white p-6 rounded-xl border border-gray-200">
          {scannedPassenger ? (
            /* Success state — show passenger info & next-ticket button */
            <div className="text-center py-6 animate-fade-in">
              <div className="w-20 h-20 bg-gradient-to-br from-green-400 to-emerald-500 rounded-full flex items-center justify-center mx-auto mb-4 shadow-lg shadow-green-200/50">
                <CheckCircle className="h-10 w-10 text-white" />
              </div>
              <h3 className="text-xl font-bold text-green-700 mb-1">✓ Check-In Successful</h3>
              <div className="mt-4 p-4 bg-green-50 rounded-xl border border-green-100">
                <div className="flex items-center gap-3 mb-3">
                  <User className="h-5 w-5 text-green-600" />
                  <span className="text-lg font-semibold text-gray-800">{scannedPassenger.name}</span>
                </div>
                <div className="flex items-center gap-3">
                  <Armchair className="h-5 w-5 text-green-600" />
                  <span className="text-lg font-semibold text-gray-800">Seat {scannedPassenger.seat}</span>
                </div>
              </div>
              <button
                onClick={resetScan}
                className="mt-6 inline-flex items-center gap-2 bg-gradient-to-r from-travel-500 to-rose-500 text-white px-6 py-3 rounded-xl font-medium
                  hover:from-travel-600 hover:to-rose-600 transition-all duration-200 shadow-sm shadow-travel-200/50 active:scale-95"
              >
                <ArrowRight className="h-4 w-4" />
                Scan Next Ticket
              </button>
            </div>
          ) : (
            <>
              <h2 className="text-lg font-semibold mb-4">Manual Entry</h2>
              <form onSubmit={handleManualScan} className="space-y-4">
                <div>
                  <label htmlFor="qr-token" className="text-sm font-medium text-gray-700">QR Token</label>
                  <input
                    id="qr-token"
                    name="qr_token"
                    required
                    value={qrToken}
                    onChange={(e) => setQrToken(e.target.value)}
                    className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-travel-500 font-mono text-sm"
                    placeholder="Paste QR token here..."
                  />
                </div>
                <div>
                  <label htmlFor="journey-id" className="text-sm font-medium text-gray-700">Journey ID</label>
                  <input
                    id="journey-id"
                    name="journey_id"
                    required
                    value={journeyId}
                    onChange={(e) => setJourneyId(e.target.value)}
                    className="mt-1 w-full px-3 py-2 border border-gray-300 rounded-lg outline-none focus:ring-2 focus:ring-travel-500"
                    placeholder="Journey UUID"
                  />
                </div>
                <button
                  type="submit"
                  disabled={loading}
                  className="w-full bg-travel-600 text-white py-2.5 rounded-lg font-medium hover:bg-travel-700 transition disabled:opacity-50"
                >
                  {loading ? "Validating..." : "Validate Ticket"}
                </button>
              </form>

              {!result && !loading && (
                <div className="mt-6 p-4 bg-travel-50 rounded-lg text-sm text-travel-700">
                  <p className="font-medium mb-1">Tip:</p>
                  <p>Scan a QR ticket using your camera, upload a QR image, or paste the token manually to validate passenger check-in.</p>
                </div>
              )}
            </>
          )}

          {/* Upload QR Image */}
          <div className="mt-6 pt-6 border-t border-gray-200">
            <div className="flex items-center gap-2 mb-3">
              <Upload className="h-4 w-4 text-travel-600" />
              <h3 className="text-sm font-semibold">Upload QR Image</h3>
            </div>
            <p className="text-xs text-gray-500 mb-3">Take a screenshot of a QR ticket and upload it to scan</p>
            <label className="flex items-center justify-center gap-2 w-full px-4 py-3 border-2 border-dashed border-gray-300 rounded-lg cursor-pointer hover:border-travel-400 hover:bg-travel-50/30 transition-colors text-sm text-gray-600">
              <Upload className="h-4 w-4" />
              Choose QR image...
              <input
                type="file"
                accept="image/*"
                className="hidden"
                disabled={loading}
                onChange={async (e) => {
                  const file = e.target.files?.[0]
                  if (!file) return

                  setResult(null)
                  setScanFeedback("Decoding QR image...")

                  try {
                    const scanner = new Html5Qrcode("qr-file-scanner")
                    const decodedText = await scanner.scanFile(file, false)
                    scanner.clear()

                    // Process the decoded text through the same decodeAndScan flow
                    // handleScanWithData manages its own loading state internally
                    decodeAndScanRef.current?.(decodedText)
                  } catch {
                    setResult({ success: false, message: "Could not read QR code from image. Try a clearer picture." })
                    setScanFeedback(null)
                  } finally {
                    // Reset the input so the same file can be selected again
                    e.target.value = ""
                  }
                }}
              />
            </label>
            {/* Hidden element for file scanner - Html5Qrcode needs a DOM element */}
            <div id="qr-file-scanner" className="hidden" />
          </div>
        </div>
      </div>
    </div>
  )
}
