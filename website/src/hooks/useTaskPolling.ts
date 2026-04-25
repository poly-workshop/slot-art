import { useCallback, useEffect, useRef, useState } from "react"
import { getTask } from "@/lib/api"
import type { Task, TaskResult } from "@/types/api"

const STORAGE_KEY = "slot_active_task"
export const RESULT_TTL_MS = 2 * 60 * 60 * 1000

function loadPersistedTask(): { taskId: string; savedAt: number } | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

function persistTaskId(taskId: string) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({ taskId, savedAt: Date.now() }))
}

function clearPersistedTask() {
  localStorage.removeItem(STORAGE_KEY)
}

function isFinished(task: Task) {
  return task.status === "TASK_STATUS_COMPLETED" || task.status === "TASK_STATUS_FAILED"
}

export function useTaskPolling(userTaskId: string | null) {
  const [task, setTask] = useState<Task | null>(null)
  const [result, setResult] = useState<TaskResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const restoredRef = useRef(false)

  const poll = useCallback(async (taskId: string) => {
    let interval = 2000
    let elapsed = 0

    const doPoll = async () => {
      try {
        const data = await getTask(taskId)
        setTask(data)
        if (data.result) setResult(data.result)

        if (data.status === "TASK_STATUS_FAILED") {
          setError(data.errorMessage || "Task failed")
          return
        }
        if (isFinished(data)) return

        elapsed += interval
        if (elapsed > 300000) {
          setError("Task timed out")
          return
        }

        interval = Math.min(interval * 1.5, 10000)
        window.setTimeout(doPoll, interval)
      } catch (e) {
        setError((e as Error).message)
      }
    }

    doPoll()
  }, [])

  useEffect(() => {
    if (restoredRef.current || userTaskId) return
    restoredRef.current = true
    const saved = loadPersistedTask()
    if (saved && Date.now() - saved.savedAt < RESULT_TTL_MS) {
      poll(saved.taskId)
    } else if (saved) {
      clearPersistedTask()
    }
  }, [poll, userTaskId])

  useEffect(() => {
    if (!userTaskId) return
    setTask(null)
    setResult(null)
    setError(null)
    persistTaskId(userTaskId)
    poll(userTaskId)
  }, [poll, userTaskId])

  const dismiss = useCallback(() => {
    clearPersistedTask()
    setTask(null)
    setResult(null)
    setError(null)
  }, [])

  return { task, result, error, dismiss, resultTTL: RESULT_TTL_MS }
}
