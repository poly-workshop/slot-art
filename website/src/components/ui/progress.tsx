import * as React from "react"

import { cn } from "@/lib/utils"

function Progress({ className, value = 0, ...props }: React.ComponentProps<"div"> & { value?: number }) {
  const pct = Math.min(Math.max(Number(value) || 0, 0), 100)
  return (
    <div
      className={cn("relative h-2 w-full overflow-hidden rounded-full bg-primary/20", className)}
      {...props}
    >
      <div
        className="h-full bg-primary transition-all"
        style={{ transform: `translateX(-${100 - pct}%)` }}
      />
    </div>
  )
}

export { Progress }
