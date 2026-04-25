import { useState } from "react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { redeemCDKey } from "@/lib/api"
import type { Credit } from "@/types/api"

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "No expiry"
}

interface CDKeyRedeemerProps {
  activeCredit?: Credit | null
  onRedeemed: () => void
}

export function CDKeyRedeemer({ activeCredit, onRedeemed }: CDKeyRedeemerProps) {
  const [key, setKey] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleRedeem = async () => {
    if (!key.trim()) return
    setLoading(true)
    setError(null)
    try {
      await redeemCDKey(key.trim())
      setKey("")
      onRedeemed()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  if (activeCredit) {
    return (
      <Card className="border-primary/50">
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>Active credit</span>
            <Badge>{activeCredit.remainingImageCount ?? 0} images</Badge>
          </CardTitle>
          <CardDescription>
            Redeemed from {activeCredit.source ?? "CDKey"} · expires {formatDate(activeCredit.expiresAt)}
          </CardDescription>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          Credit ID: {activeCredit.creditId}
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Redeem CDKey</CardTitle>
        <CardDescription>Enter an admin-generated CDKey to unlock one generation credit.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex gap-2">
          <Input
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="SA-XXXX-XXXX-XXXX-XXXX"
            disabled={loading}
          />
          <Button onClick={handleRedeem} disabled={loading || !key.trim()}>
            {loading ? "Redeeming..." : "Redeem"}
          </Button>
        </div>
        {error && <p className="text-sm text-destructive">{error}</p>}
      </CardContent>
    </Card>
  )
}
