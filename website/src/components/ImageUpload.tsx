import { useCallback, useState } from "react"
import { uploadReferenceImage } from "@/lib/api"

interface ImageUploadProps {
  onUpload: (refId: string, preview: string) => void
  disabled: boolean
  maxImages: number
  maxBytes: number
}

export function ImageUpload({ onUpload, disabled, maxImages, maxBytes }: ImageUploadProps) {
  const [previews, setPreviews] = useState<{ refId: string; preview: string }[]>([])
  const [dragOver, setDragOver] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleFile = useCallback(async (file: File) => {
    setError(null)
    if (disabled || maxImages <= 0) return
    if (previews.length >= maxImages) {
      setError(`Only ${maxImages} reference images are allowed.`)
      return
    }
    if (!["image/png", "image/jpeg", "image/webp"].includes(file.type)) {
      setError("Only PNG, JPEG and WebP are supported.")
      return
    }
    if (file.size > maxBytes) {
      setError(`Image must be smaller than ${Math.round(maxBytes / 1024 / 1024)}MB.`)
      return
    }

    const preview = URL.createObjectURL(file)
    try {
      const res = await uploadReferenceImage(file)
      const refId = res.referenceImage?.referenceImageId
      if (!refId) throw new Error("Upload response is missing reference image ID")
      setPreviews((prev) => [...prev, { refId, preview }])
      onUpload(refId, preview)
    } catch (e) {
      URL.revokeObjectURL(preview)
      setError((e as Error).message)
    }
  }, [disabled, maxBytes, maxImages, onUpload, previews.length])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    for (const file of Array.from(e.dataTransfer.files)) {
      handleFile(file)
    }
  }, [handleFile])

  if (maxImages <= 0) {
    return (
      <div className="rounded-lg border border-dashed p-4 text-center text-sm text-muted-foreground">
        Reference images are disabled for the current OpenAI Image2 configuration.
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div
        className={`cursor-pointer rounded-lg border-2 border-dashed p-6 text-center transition-colors ${
          dragOver ? "border-primary bg-primary/5" : "border-muted"
        } ${disabled ? "pointer-events-none opacity-50" : ""}`}
        onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
        onClick={() => {
          if (disabled) return
          const input = document.createElement("input")
          input.type = "file"
          input.accept = "image/png,image/jpeg,image/webp"
          input.multiple = true
          input.onchange = () => {
            for (const file of Array.from(input.files || [])) {
              handleFile(file)
            }
          }
          input.click()
        }}
      >
        <p className="text-sm text-muted-foreground">
          Drop reference images here or click to upload
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          PNG, JPEG, WebP · max {Math.round(maxBytes / 1024 / 1024)}MB · {previews.length}/{maxImages}
        </p>
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}
      {previews.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {previews.map((preview) => (
            <img
              key={preview.refId}
              src={preview.preview}
              alt="reference"
              className="h-16 w-16 rounded border object-cover"
            />
          ))}
        </div>
      )}
    </div>
  )
}
