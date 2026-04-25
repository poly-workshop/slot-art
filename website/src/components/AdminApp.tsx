import { useEffect, useMemo, useState } from "react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  adminCreatePromptTemplate,
  adminDeletePromptTemplate,
  adminGenerateCDKeys,
  adminGetDashboard,
  adminListCDKeys,
  adminListCredits,
  adminListPromptTemplates,
  adminListSessions,
  adminListTasks,
  adminRevokeCDKey,
  adminSetPromptTemplateEnabled,
  adminUpdatePromptTemplate,
  getAdminToken,
  setAdminToken,
} from "@/lib/api"
import type {
  AdminCDKey,
  AdminCredit,
  AdminDashboard,
  AdminPromptTemplate,
  AdminPromptTemplateField,
  AdminSession,
  AdminTask,
  GeneratedCDKey,
} from "@/types/api"

const DEFAULT_FIELDS = JSON.stringify([
  {
    name: "subject",
    label: "Subject",
    type: "PROMPT_TEMPLATE_FIELD_TYPE_TEXT",
    required: true,
    maxLength: 80,
    placeholder: "cute cat, red sports car..."
  },
  {
    name: "style",
    label: "Style",
    type: "PROMPT_TEMPLATE_FIELD_TYPE_SELECT",
    required: true,
    options: ["pixel art", "watercolor", "cinematic", "3D render"]
  }
], null, 2)

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : "-"
}

function DashboardCard({ label, value }: { label: string; value?: number }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardDescription>{label}</CardDescription>
        <CardTitle className="text-2xl">{value ?? 0}</CardTitle>
      </CardHeader>
    </Card>
  )
}

export function AdminApp() {
  const [token, setToken] = useState(getAdminToken())
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [dashboard, setDashboard] = useState<AdminDashboard | null>(null)
  const [cdkeys, setCDKeys] = useState<AdminCDKey[]>([])
  const [generated, setGenerated] = useState<GeneratedCDKey[]>([])
  const [templates, setTemplates] = useState<AdminPromptTemplate[]>([])
  const [tasks, setTasks] = useState<AdminTask[]>([])
  const [credits, setCredits] = useState<AdminCredit[]>([])
  const [sessions, setSessions] = useState<AdminSession[]>([])

  const [count, setCount] = useState(5)
  const [imageCount, setImageCount] = useState(1)
  const [expiresDays, setExpiresDays] = useState(30)
  const [note, setNote] = useState("")

  const [editId, setEditId] = useState<string | null>(null)
  const [templateName, setTemplateName] = useState("Starter template")
  const [templateDescription, setTemplateDescription] = useState("User-facing template summary")
  const [templateCategory, setTemplateCategory] = useState("default")
  const [templateBody, setTemplateBody] = useState("Create a {{ style }} image of {{ subject }}. High quality, clean composition.")
  const [templateFields, setTemplateFields] = useState(DEFAULT_FIELDS)
  const [templateEnabled, setTemplateEnabled] = useState(true)

  const ready = useMemo(() => token.trim().length > 0, [token])

  const reload = async () => {
    if (!ready) return
    setLoading(true)
    setError(null)
    try {
      setAdminToken(token.trim())
      const [dashboardRes, cdkeyRes, templateRes, taskRes, creditRes, sessionRes] = await Promise.all([
        adminGetDashboard(),
        adminListCDKeys(),
        adminListPromptTemplates(true),
        adminListTasks(),
        adminListCredits(),
        adminListSessions(),
      ])
      setDashboard(dashboardRes)
      setCDKeys(cdkeyRes.cdkeys ?? [])
      setTemplates(templateRes.templates ?? [])
      setTasks(taskRes.tasks ?? [])
      setCredits(creditRes.credits ?? [])
      setSessions(sessionRes.sessions ?? [])
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    reload()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const generateCDKeys = async () => {
    setLoading(true)
    setError(null)
    try {
      const expiresAtUnix = expiresDays > 0 ? String(Math.floor(Date.now() / 1000) + expiresDays * 86400) : undefined
      const res = await adminGenerateCDKeys({ count, imageCount, expiresAtUnix, note })
      setGenerated(res.cdkeys ?? [])
      await reload()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  const resetTemplateForm = () => {
    setEditId(null)
    setTemplateName("Starter template")
    setTemplateDescription("User-facing template summary")
    setTemplateCategory("default")
    setTemplateBody("Create a {{ style }} image of {{ subject }}. High quality, clean composition.")
    setTemplateFields(DEFAULT_FIELDS)
    setTemplateEnabled(true)
  }

  const loadTemplateIntoForm = (template: AdminPromptTemplate) => {
    setEditId(template.templateId ?? null)
    setTemplateName(template.name ?? "")
    setTemplateDescription(template.description ?? "")
    setTemplateCategory(template.category ?? "")
    setTemplateBody(template.templateBody ?? "")
    setTemplateFields(JSON.stringify(template.fields ?? [], null, 2))
    setTemplateEnabled(Boolean(template.enabled))
  }

  const saveTemplate = async () => {
    setLoading(true)
    setError(null)
    try {
      const fields = JSON.parse(templateFields) as AdminPromptTemplateField[]
      const payload: AdminPromptTemplate = {
        templateId: editId ?? undefined,
        name: templateName,
        description: templateDescription,
        category: templateCategory,
        enabled: templateEnabled,
        templateBody,
        fields,
        source: "admin-ui",
      }
      if (editId) {
        await adminUpdatePromptTemplate(payload)
      } else {
        await adminCreatePromptTemplate(payload)
      }
      resetTemplateForm()
      await reload()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-svh bg-background">
      <header className="flex items-center justify-between border-b p-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight">Slot Art Admin</h1>
          <p className="text-sm text-muted-foreground">CDKeys, credits, tasks and prompt templates</p>
        </div>
        <a href="/" className="text-sm text-muted-foreground hover:text-foreground">Public page</a>
      </header>

      <main className="mx-auto max-w-6xl space-y-6 p-6">
        <Card>
          <CardHeader>
            <CardTitle>Admin token</CardTitle>
            <CardDescription>Stored locally and sent as a bearer token to Admin API only.</CardDescription>
          </CardHeader>
          <CardContent className="flex gap-2">
            <Input value={token} onChange={(e) => setToken(e.target.value)} placeholder="ADMIN_TOKEN" type="password" />
            <Button onClick={reload} disabled={!ready || loading}>{loading ? "Loading..." : "Connect"}</Button>
          </CardContent>
        </Card>

        {error && <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}

        <section className="grid gap-4 md:grid-cols-4">
          <DashboardCard label="Pending tasks" value={dashboard?.pendingTasks} />
          <DashboardCard label="Processing tasks" value={dashboard?.processingTasks} />
          <DashboardCard label="Completed tasks" value={dashboard?.completedTasks} />
          <DashboardCard label="Failed tasks" value={dashboard?.failedTasks} />
          <DashboardCard label="Created CDKeys" value={dashboard?.cdkeysCreated} />
          <DashboardCard label="Redeemed CDKeys" value={dashboard?.cdkeysRedeemed} />
          <DashboardCard label="Sessions" value={dashboard?.activeSessions} />
          <DashboardCard label="Enabled templates" value={dashboard?.activePromptTemplates} />
        </section>

        <section className="grid gap-6 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Generate CDKeys</CardTitle>
              <CardDescription>Plaintext keys are shown only once after generation.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="grid grid-cols-3 gap-2">
                <Input type="number" min={1} max={500} value={count} onChange={(e) => setCount(Number(e.target.value))} />
                <Input type="number" min={1} max={4} value={imageCount} onChange={(e) => setImageCount(Number(e.target.value))} />
                <Input type="number" min={0} value={expiresDays} onChange={(e) => setExpiresDays(Number(e.target.value))} />
              </div>
              <p className="text-xs text-muted-foreground">Count · images per key · expires in days (0 means no expiry)</p>
              <Input value={note} onChange={(e) => setNote(e.target.value)} placeholder="Internal note" />
              <Button onClick={generateCDKeys} disabled={!ready || loading}>Generate</Button>
              {generated.length > 0 && (
                <div className="rounded-lg bg-muted p-3 text-xs">
                  <p className="mb-2 font-medium">Plaintext keys</p>
                  <div className="max-h-40 space-y-1 overflow-auto font-mono">
                    {generated.map((item) => <div key={item.cdkey?.cdkeyId}>{item.plaintextKey}</div>)}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{editId ? "Update prompt template" : "Create prompt template"}</CardTitle>
              <CardDescription>Template body is admin-only; public API returns only fields and summary.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <Input value={templateName} onChange={(e) => setTemplateName(e.target.value)} placeholder="Name" />
              <Input value={templateDescription} onChange={(e) => setTemplateDescription(e.target.value)} placeholder="Description" />
              <Input value={templateCategory} onChange={(e) => setTemplateCategory(e.target.value)} placeholder="Category" />
              <Textarea value={templateBody} onChange={(e) => setTemplateBody(e.target.value)} className="min-h-24 font-mono text-xs" />
              <Textarea value={templateFields} onChange={(e) => setTemplateFields(e.target.value)} className="min-h-48 font-mono text-xs" />
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={templateEnabled} onChange={(e) => setTemplateEnabled(e.target.checked)} />
                Enabled
              </label>
              <div className="flex gap-2">
                <Button onClick={saveTemplate} disabled={!ready || loading}>{editId ? "Update" : "Create"}</Button>
                {editId && <Button variant="outline" onClick={resetTemplateForm}>Cancel edit</Button>}
              </div>
            </CardContent>
          </Card>
        </section>

        <section className="grid gap-6 lg:grid-cols-2">
          <Card>
            <CardHeader><CardTitle>Prompt templates</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              {templates.map((template) => (
                <div key={template.templateId} className="rounded-lg border p-3">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="font-medium">{template.name}</p>
                      <p className="text-xs text-muted-foreground">{template.templateId} · v{template.version}</p>
                    </div>
                    <Badge variant={template.enabled ? "default" : "secondary"}>{template.enabled ? "enabled" : "disabled"}</Badge>
                  </div>
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Button size="sm" variant="outline" onClick={() => loadTemplateIntoForm(template)}>Edit</Button>
                    {template.templateId && (
                      <Button size="sm" variant="outline" onClick={async () => { await adminSetPromptTemplateEnabled(template.templateId!, !template.enabled); await reload() }}>
                        {template.enabled ? "Disable" : "Enable"}
                      </Button>
                    )}
                    {template.templateId && (
                      <Button size="sm" variant="destructive" onClick={async () => { await adminDeletePromptTemplate(template.templateId!); await reload() }}>
                        Delete
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>

          <Card>
            <CardHeader><CardTitle>CDKeys</CardTitle></CardHeader>
            <CardContent className="space-y-2">
              {cdkeys.map((cdkey) => (
                <div key={cdkey.cdkeyId} className="flex items-center justify-between rounded-lg border p-3 text-sm">
                  <div>
                    <p className="font-mono">{cdkey.maskedKey}</p>
                    <p className="text-xs text-muted-foreground">{cdkey.cdkeyId} · expires {formatDate(cdkey.expiresAt)}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">{cdkey.status}</Badge>
                    {cdkey.status === "CD_KEY_STATUS_CREATED" && cdkey.cdkeyId && (
                      <Button size="sm" variant="destructive" onClick={async () => { await adminRevokeCDKey(cdkey.cdkeyId!); await reload() }}>
                        Revoke
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        </section>

        <section className="grid gap-6 lg:grid-cols-3">
          <Card>
            <CardHeader><CardTitle>Recent tasks</CardTitle></CardHeader>
            <CardContent className="space-y-2 text-sm">
              {tasks.map((task) => (
                <div key={task.taskId} className="rounded-lg border p-2">
                  <p className="font-mono text-xs">{task.taskId}</p>
                  <p className="text-xs text-muted-foreground">{task.status} · {task.imageCountGenerated ?? 0}/{task.imageCountRequested ?? 0}</p>
                </div>
              ))}
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>Recent credits</CardTitle></CardHeader>
            <CardContent className="space-y-2 text-sm">
              {credits.map((credit) => (
                <div key={credit.creditId} className="rounded-lg border p-2">
                  <p className="font-mono text-xs">{credit.creditId}</p>
                  <p className="text-xs text-muted-foreground">{credit.status} · {credit.remainingImageCount ?? 0}/{credit.imageCount ?? 0}</p>
                </div>
              ))}
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>Recent sessions</CardTitle></CardHeader>
            <CardContent className="space-y-2 text-sm">
              {sessions.map((session) => (
                <div key={session.uid} className="rounded-lg border p-2">
                  <p className="font-mono text-xs">{session.uid}</p>
                  <p className="text-xs text-muted-foreground">last seen {formatDate(session.lastSeen)}</p>
                </div>
              ))}
            </CardContent>
          </Card>
        </section>
      </main>
    </div>
  )
}
