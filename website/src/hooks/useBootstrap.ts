import { useCallback, useEffect, useState } from "react"
import { getBootstrap, type BootstrapResponse } from "@/lib/api"

export function useBootstrap(enabled = true) {
  const [bootstrap, setBootstrap] = useState<BootstrapResponse | null>(null)
  const [loading, setLoading] = useState(enabled)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    if (!enabled) return
    setLoading(true)
    setError(null)
    try {
      setBootstrap(await getBootstrap())
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [enabled])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { bootstrap, loading, error, refresh }
}
