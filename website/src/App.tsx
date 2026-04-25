import { useState } from "react"
import { AdminApp } from "@/components/AdminApp"
import { CDKeyRedeemer } from "@/components/CDKeyRedeemer"
import { Header } from "@/components/Header"
import { ImageUpload } from "@/components/ImageUpload"
import { PromptForm } from "@/components/PromptForm"
import { ResultGallery } from "@/components/ResultGallery"
import { SlotMachine } from "@/components/SlotMachine"
import { TaskStatusIndicator } from "@/components/TaskStatus"
import { useBootstrap } from "@/hooks/useBootstrap"
import { useSession } from "@/hooks/useSession"
import { useTaskPolling } from "@/hooks/useTaskPolling"
import { createTask } from "@/lib/api"

function PublicApp() {
  const { loading: sessionLoading } = useSession()
  const { bootstrap, loading: bootstrapLoading, error: bootstrapError, refresh } = useBootstrap(!sessionLoading)
  const [taskId, setTaskId] = useState<string | null>(null)
  const [uploadRefIds, setUploadRefIds] = useState<Set<string>>(new Set())
  const { task, result, error: taskError, dismiss } = useTaskPolling(taskId)

  const activeCredit = bootstrap?.activeCredit ?? null
  const templates = bootstrap?.promptTemplates ?? []
  const isGenerating = task && !["TASK_STATUS_COMPLETED", "TASK_STATUS_FAILED"].includes(task.status ?? "")

  const handleGenerate = async (input: { templateId: string; slotValues: Record<string, string> }) => {
    if (!activeCredit?.creditId) return
    try {
      const res = await createTask({
        creditId: activeCredit.creditId,
        templateId: input.templateId,
        slotValues: input.slotValues,
        referenceImageIds: Array.from(uploadRefIds),
      })
      if (res.taskId) setTaskId(res.taskId)
      setUploadRefIds(new Set())
      refresh()
    } catch (e) {
      alert((e as Error).message)
    }
  }

  const handleRefUpload = (refId: string) => {
    setUploadRefIds((prev) => new Set(prev).add(refId))
  }

  const handleBackToMain = () => {
    dismiss()
    setTaskId(null)
    refresh()
  }

  if (sessionLoading || bootstrapLoading) {
    return (
      <div className="flex min-h-svh items-center justify-center">
        <p className="text-muted-foreground">Loading...</p>
      </div>
    )
  }

  return (
    <div className="flex min-h-svh flex-col">
      <Header hasActiveCredit={Boolean(activeCredit)} />
      <main className="mx-auto w-full max-w-2xl flex-1 space-y-6 p-6">
        <SlotMachine spinning={Boolean(isGenerating)} />

        {(bootstrapError || taskError) && (
          <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
            {bootstrapError || taskError}
          </div>
        )}

        {result && task && (
          <ResultGallery task={task} result={result} onBack={handleBackToMain} />
        )}

        {task && !result && (
          <TaskStatusIndicator task={task} error={taskError} />
        )}

        {!isGenerating && !result && (
          <div className="space-y-6">
            <CDKeyRedeemer activeCredit={activeCredit} onRedeemed={refresh} />
            {activeCredit && (
              <div className="space-y-4">
                <ImageUpload
                  onUpload={handleRefUpload}
                  disabled={false}
                  maxImages={bootstrap?.maxReferenceImages ?? 0}
                  maxBytes={bootstrap?.maxReferenceImageBytes ?? 10 * 1024 * 1024}
                />
                <PromptForm
                  templates={templates}
                  onSubmit={handleGenerate}
                  disabled={!activeCredit}
                />
              </div>
            )}
          </div>
        )}
      </main>
    </div>
  )
}

export function App() {
  return window.location.pathname.startsWith("/admin") ? <AdminApp /> : <PublicApp />
}

export default App
