import { useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import type { PromptTemplateSummary } from "@/types/api"

interface PromptFormProps {
  templates: PromptTemplateSummary[]
  onSubmit: (input: { templateId: string; slotValues: Record<string, string> }) => void
  disabled: boolean
}

function emptyValues(template?: PromptTemplateSummary) {
  const values: Record<string, string> = {}
  for (const field of template?.fields ?? []) {
    if (field.name) values[field.name] = ""
  }
  return values
}

export function PromptForm({ templates, onSubmit, disabled }: PromptFormProps) {
  const [templateId, setTemplateId] = useState(templates[0]?.templateId ?? "")
  const selected = useMemo(
    () => templates.find((template) => template.templateId === templateId) ?? templates[0],
    [templateId, templates]
  )
  const [values, setValues] = useState<Record<string, string>>(() => emptyValues(selected))

  const selectTemplate = (nextId: string) => {
    const next = templates.find((template) => template.templateId === nextId)
    setTemplateId(nextId)
    setValues(emptyValues(next))
  }

  const updateValue = (name: string, value: string) => {
    setValues((prev) => ({ ...prev, [name]: value }))
  }

  const canSubmit = Boolean(selected?.templateId) && (selected?.fields ?? []).every((field) => {
    if (!field.required || !field.name) return true
    return Boolean(values[field.name]?.trim())
  })

  if (templates.length === 0) {
    return (
      <div className="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">
        No prompt templates are enabled yet. Ask an admin to create one.
      </div>
    )
  }

  return (
    <div className="space-y-4 rounded-xl border bg-card p-4">
      <div className="space-y-2">
        <label className="text-sm font-medium">Prompt template</label>
        <select
          className="h-9 w-full rounded-md border bg-background px-3 text-sm"
          value={selected?.templateId ?? ""}
          disabled={disabled}
          onChange={(e) => selectTemplate(e.target.value)}
        >
          {templates.map((template) => (
            <option key={template.templateId} value={template.templateId}>
              {template.name} v{template.version}
            </option>
          ))}
        </select>
        {selected?.description && (
          <p className="text-xs text-muted-foreground">{selected.description}</p>
        )}
      </div>

      {(selected?.fields ?? []).map((field) => {
        if (!field.name) return null
        const value = values[field.name] ?? ""
        const label = field.label || field.name
        const commonLabel = (
          <label className="text-sm font-medium">
            {label}{field.required ? " *" : ""}
          </label>
        )

        if (field.type === "PROMPT_TEMPLATE_FIELD_TYPE_SELECT") {
          return (
            <div key={field.name} className="space-y-2">
              {commonLabel}
              <select
                className="h-9 w-full rounded-md border bg-background px-3 text-sm"
                value={value}
                disabled={disabled}
                onChange={(e) => updateValue(field.name!, e.target.value)}
              >
                <option value="">Select...</option>
                {(field.options ?? []).map((option) => (
                  <option key={option} value={option}>{option}</option>
                ))}
              </select>
              {field.helpText && <p className="text-xs text-muted-foreground">{field.helpText}</p>}
            </div>
          )
        }

        if (field.type === "PROMPT_TEMPLATE_FIELD_TYPE_TEXTAREA") {
          return (
            <div key={field.name} className="space-y-2">
              {commonLabel}
              <Textarea
                value={value}
                placeholder={field.placeholder}
                maxLength={field.maxLength || undefined}
                disabled={disabled}
                onChange={(e) => updateValue(field.name!, e.target.value)}
              />
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>{field.helpText}</span>
                {field.maxLength ? <span>{value.length}/{field.maxLength}</span> : null}
              </div>
            </div>
          )
        }

        return (
          <div key={field.name} className="space-y-2">
            {commonLabel}
            <Input
              type={field.type === "PROMPT_TEMPLATE_FIELD_TYPE_NUMBER" ? "number" : "text"}
              value={value}
              placeholder={field.placeholder}
              maxLength={field.maxLength || undefined}
              disabled={disabled}
              onChange={(e) => updateValue(field.name!, e.target.value)}
            />
            {field.helpText && <p className="text-xs text-muted-foreground">{field.helpText}</p>}
          </div>
        )
      })}

      <div className="flex justify-end">
        <Button
          onClick={() => selected?.templateId && onSubmit({ templateId: selected.templateId, slotValues: values })}
          disabled={disabled || !canSubmit}
        >
          Generate
        </Button>
      </div>
    </div>
  )
}
