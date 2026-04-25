import { useState, useEffect, useCallback } from "react"
import { initSession } from "@/lib/api"

export function useSession() {
  const [uid, setUid] = useState<string | null>(localStorage.getItem("slot_uid"))
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const initialize = useCallback(async () => {
    try {
      const fingerprint = [
        navigator.userAgent,
        screen.width,
        screen.height,
        navigator.language,
        Intl.DateTimeFormat().resolvedOptions().timeZone,
      ].join("|")
      const res = await initSession(fingerprint)
      const nextUID = res.uid ?? null
      setUid(nextUID)
      if (nextUID) localStorage.setItem("slot_uid", nextUID)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    initialize()
  }, [initialize])

  return { uid, loading, error, refresh: initialize }
}
