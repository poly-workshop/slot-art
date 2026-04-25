import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import type { Task } from "@/types/api"

interface TaskStatusProps {
  task: Task
  error?: string | null
}

function progressFor(status?: string) {
  switch (status) {
    case "TASK_STATUS_PENDING":
      return 15
    case "TASK_STATUS_PROCESSING":
      return 65
    case "TASK_STATUS_COMPLETED":
      return 100
    case "TASK_STATUS_FAILED":
      return 100
    default:
      return 5
  }
}

function labelFor(status?: string) {
  switch (status) {
    case "TASK_STATUS_PENDING":
      return "Queuing your task..."
    case "TASK_STATUS_PROCESSING":
      return "Generating images, please wait..."
    case "TASK_STATUS_COMPLETED":
      return "Generation completed."
    case "TASK_STATUS_FAILED":
      return "Generation failed."
    default:
      return "Waiting..."
  }
}

export function TaskStatusIndicator({ task, error }: TaskStatusProps) {
  return (
    <div className="space-y-3 text-center">
      <Progress value={progressFor(task.status)} className="w-full" />
      <p className={task.status === "TASK_STATUS_FAILED" ? "text-sm text-destructive" : "text-sm text-muted-foreground"}>
        {task.status === "TASK_STATUS_FAILED" ? (task.errorMessage || error || "Generation failed") : labelFor(task.status)}
      </p>
      <div className="flex flex-wrap justify-center gap-2">
        {task.templateName && <Badge variant="outline">{task.templateName}</Badge>}
        <Badge variant="secondary">{task.imageCountRequested ?? 0} requested</Badge>
        <Badge variant="secondary">{task.imageCountGenerated ?? 0} generated</Badge>
      </div>
    </div>
  )
}
