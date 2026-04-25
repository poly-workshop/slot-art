import { useEffect, useState } from "react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { Task, TaskResult } from "@/types/api"

const RESULT_TTL_MS = 2 * 60 * 60 * 1000

function formatTimeRemaining(ms: number): string {
  const totalSec = Math.floor(ms / 1000)
  const h = Math.floor(totalSec / 3600)
  const m = Math.floor((totalSec % 3600) / 60)
  const s = totalSec % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

interface ResultGalleryProps {
  task: Task
  result: TaskResult
  onBack: () => void
}

export function ResultGallery({ task, result, onBack }: ResultGalleryProps) {
  const [remaining, setRemaining] = useState(0)

  useEffect(() => {
    const generatedAt = result.generatedAt ? Date.parse(result.generatedAt) : Date.now()
    const expiresAt = generatedAt + RESULT_TTL_MS
    const tick = () => setRemaining(Math.max(0, expiresAt - Date.now()))
    tick()
    const interval = window.setInterval(tick, 1000)
    return () => window.clearInterval(interval)
  }, [result.generatedAt])

  if (!result.images?.length) return null

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">Generated Images</h3>
        <div className="flex items-center gap-3">
          {remaining > 0 ? (
            <Badge variant="secondary">{formatTimeRemaining(remaining)} remaining</Badge>
          ) : (
            <Badge variant="destructive">Expired</Badge>
          )}
          <Button variant="ghost" size="sm" onClick={onBack}>Back</Button>
        </div>
      </div>

      <div className="rounded-lg bg-muted p-3">
        <p className="text-sm font-medium">Template</p>
        <div className="mt-2 flex flex-wrap gap-2">
          {task.templateName && <Badge variant="outline">{task.templateName}</Badge>}
          <Badge variant="secondary">{result.images.length} images</Badge>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        {result.images.map((img) => (
          <div key={img.imageId} className="space-y-2">
            <div className="aspect-square overflow-hidden rounded-lg border bg-muted">
              {img.downloadUrl ? (
                <img src={img.downloadUrl} alt="generated" className="h-full w-full object-cover" />
              ) : (
                <div className="flex h-full items-center justify-center text-xs text-muted-foreground">No URL</div>
              )}
            </div>
            <Button
              variant="outline"
              size="sm"
              className="w-full"
              disabled={!img.downloadUrl}
              onClick={() => img.downloadUrl && window.open(img.downloadUrl, "_blank")}
            >
              Download
            </Button>
          </div>
        ))}
      </div>
    </div>
  )
}
