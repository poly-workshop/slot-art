import type { components as AdminComponents } from "@/gen/admin-api"
import type { components as SlotComponents } from "@/gen/slot-api"

export type SlotSchemas = SlotComponents["schemas"]
export type AdminSchemas = AdminComponents["schemas"]

export type SessionResponse = SlotSchemas["slot.v1.InitSessionResponse"]
export type BootstrapResponse = SlotSchemas["slot.v1.GetBootstrapResponse"]
export type Credit = SlotSchemas["slot.v1.Credit"]
export type PromptTemplateSummary = SlotSchemas["slot.v1.PromptTemplateSummary"]
export type PromptTemplateField = SlotSchemas["slot.v1.PromptTemplateField"]
export type ReferenceImage = SlotSchemas["slot.v1.ReferenceImage"]
export type CreateTaskResponse = SlotSchemas["slot.v1.CreateTaskResponse"]
export type Task = SlotSchemas["slot.v1.Task"]
export type TaskResult = SlotSchemas["slot.v1.TaskResult"]
export type ResultImage = SlotSchemas["slot.v1.ResultImage"]
export type DownloadUrlResponse = SlotSchemas["slot.v1.GetDownloadUrlResponse"]

export type AdminDashboard = AdminSchemas["slot.v1.GetDashboardResponse"]
export type AdminPromptTemplate = AdminSchemas["slot.v1.AdminPromptTemplate"]
export type AdminPromptTemplateField = AdminSchemas["slot.v1.PromptTemplateField"]
export type AdminTask = AdminSchemas["slot.v1.AdminTask"]
export type AdminCDKey = AdminSchemas["slot.v1.CDKey"]
export type GeneratedCDKey = AdminSchemas["slot.v1.GeneratedCDKey"]
export type AdminCredit = AdminSchemas["slot.v1.Credit"]
export type AdminSession = AdminSchemas["slot.v1.Session"]

const UID_STORAGE_KEY = "slot_uid"
const ADMIN_TOKEN_STORAGE_KEY = "slot_admin_token"

function getUID(): string | null {
  return localStorage.getItem(UID_STORAGE_KEY)
}

function setUID(uid: string) {
  localStorage.setItem(UID_STORAGE_KEY, uid)
}

export function getAdminToken(): string {
  return localStorage.getItem(ADMIN_TOKEN_STORAGE_KEY) ?? ""
}

export function setAdminToken(token: string) {
  localStorage.setItem(ADMIN_TOKEN_STORAGE_KEY, token)
}

function buildQuery(params?: Record<string, string | number | boolean | undefined>) {
  if (!params) return ""
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== "") query.set(key, String(value))
  }
  const raw = query.toString()
  return raw ? `?${raw}` : ""
}

async function fetchJSON<T>(path: string, init: RequestInit = {}, admin = false): Promise<T> {
  const headers: Record<string, string> = {}
  const uid = getUID()
  if (uid) headers["X-Slot-UID"] = uid
  if (admin) {
    const token = getAdminToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }
  if (init.body && !(init.body instanceof FormData)) {
    headers["Content-Type"] = "application/json"
  }
  Object.assign(headers, init.headers)

  const res = await fetch(path, { ...init, headers })
  if (!res.ok) {
    const data = (await res.json().catch(() => ({}))) as { error?: string; message?: string }
    throw new Error(data.message || data.error || `HTTP ${res.status}`)
  }
  if (res.status === 204) return {} as T
  return res.json() as Promise<T>
}

function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(reader.error)
    reader.onload = () => {
      const result = String(reader.result ?? "")
      resolve(result.includes(",") ? result.split(",")[1] : result)
    }
    reader.readAsDataURL(file)
  })
}

export async function initSession(fingerprint?: string) {
  const res = await fetchJSON<SessionResponse>("/api/v1/session", {
    method: "POST",
    body: JSON.stringify({ fingerprint }),
  })
  if (res.uid) setUID(res.uid)
  return res
}

export function getBootstrap() {
  return fetchJSON<BootstrapResponse>("/api/v1/bootstrap")
}

export function redeemCDKey(key: string) {
  return fetchJSON<{ credit?: Credit }>("/api/v1/cdkeys/redeem", {
    method: "POST",
    body: JSON.stringify({ key }),
  })
}

export async function uploadReferenceImage(file: File) {
  const data = await fileToBase64(file)
  return fetchJSON<{ referenceImage?: ReferenceImage }>("/api/v1/reference-images", {
    method: "POST",
    body: JSON.stringify({ filename: file.name, contentType: file.type, data }),
  })
}

export function createTask(body: {
  creditId: string
  templateId: string
  slotValues: Record<string, string>
  referenceImageIds: string[]
}) {
  return fetchJSON<CreateTaskResponse>("/api/v1/tasks", {
    method: "POST",
    body: JSON.stringify(body),
  })
}

export async function getTask(taskId: string) {
  const res = await fetchJSON<{ task?: Task }>(`/api/v1/tasks/${encodeURIComponent(taskId)}`)
  if (!res.task) throw new Error("Task not found")
  return res.task
}

export function getDownloadUrl(taskId: string, imageId: string) {
  return fetchJSON<DownloadUrlResponse>(
    `/api/v1/tasks/${encodeURIComponent(taskId)}/images/${encodeURIComponent(imageId)}/download-url`
  )
}

export function adminGetDashboard() {
  return fetchJSON<AdminDashboard>("/api/admin/v1/dashboard", {}, true)
}

export function adminGenerateCDKeys(body: {
  count: number
  imageCount: number
  batchId?: string
  expiresAtUnix?: string
  note?: string
}) {
  return fetchJSON<{ cdkeys?: GeneratedCDKey[] }>(
    "/api/admin/v1/cdkeys/generate",
    { method: "POST", body: JSON.stringify(body) },
    true
  )
}

export function adminListCDKeys(limit = 50) {
  return fetchJSON<{ cdkeys?: AdminCDKey[] }>(
    `/api/admin/v1/cdkeys${buildQuery({ "page.pageSize": limit })}`,
    {},
    true
  )
}

export function adminRevokeCDKey(cdkeyId: string) {
  return fetchJSON<{ cdkey?: AdminCDKey }>(
    `/api/admin/v1/cdkeys/${encodeURIComponent(cdkeyId)}/revoke`,
    { method: "POST", body: JSON.stringify({}) },
    true
  )
}

export function adminListPromptTemplates(includeDisabled = true) {
  return fetchJSON<{ templates?: AdminPromptTemplate[] }>(
    `/api/admin/v1/prompt-templates${buildQuery({ includeDisabled, "page.pageSize": 100 })}`,
    {},
    true
  )
}

export function adminCreatePromptTemplate(template: AdminPromptTemplate) {
  return fetchJSON<{ template?: AdminPromptTemplate }>(
    "/api/admin/v1/prompt-templates",
    { method: "POST", body: JSON.stringify(template) },
    true
  )
}

export function adminUpdatePromptTemplate(template: AdminPromptTemplate) {
  if (!template.templateId) throw new Error("templateId is required")
  return fetchJSON<{ template?: AdminPromptTemplate }>(
    `/api/admin/v1/prompt-templates/${encodeURIComponent(template.templateId)}`,
    { method: "PATCH", body: JSON.stringify(template) },
    true
  )
}

export function adminSetPromptTemplateEnabled(templateId: string, enabled: boolean) {
  return fetchJSON<{ template?: AdminPromptTemplate }>(
    `/api/admin/v1/prompt-templates/${encodeURIComponent(templateId)}/set-enabled`,
    { method: "POST", body: JSON.stringify({ enabled }) },
    true
  )
}

export function adminDeletePromptTemplate(templateId: string) {
  return fetchJSON<Record<string, never>>(
    `/api/admin/v1/prompt-templates/${encodeURIComponent(templateId)}`,
    { method: "DELETE" },
    true
  )
}

export function adminListTasks(limit = 50) {
  return fetchJSON<{ tasks?: AdminTask[] }>(
    `/api/admin/v1/tasks${buildQuery({ "page.pageSize": limit })}`,
    {},
    true
  )
}

export function adminListCredits(limit = 50) {
  return fetchJSON<{ credits?: AdminCredit[] }>(
    `/api/admin/v1/credits${buildQuery({ "page.pageSize": limit })}`,
    {},
    true
  )
}

export function adminListSessions(limit = 50) {
  return fetchJSON<{ sessions?: AdminSession[] }>(
    `/api/admin/v1/sessions${buildQuery({ "page.pageSize": limit })}`,
    {},
    true
  )
}
