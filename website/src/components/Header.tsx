import { Badge } from "@/components/ui/badge"

interface HeaderProps {
  hasActiveCredit: boolean
}

export function Header({ hasActiveCredit }: HeaderProps) {
  return (
    <header className="flex items-center justify-between p-4 border-b">
      <h1 className="text-xl font-bold tracking-tight">Slot Art</h1>
      <div className="flex items-center gap-3">
        <a href="/admin" className="text-sm text-muted-foreground hover:text-foreground">Admin</a>
        {hasActiveCredit && <Badge variant="default">Credit active</Badge>}
      </div>
    </header>
  )
}
