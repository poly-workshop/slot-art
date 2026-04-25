import { useState, useEffect } from "react"

const REELS = ["🎨", "🖼️", "✨", "🎰", "🌈", "🎭", "🌟", "💫", "🎪"]

interface SlotMachineProps {
  spinning: boolean
}

export function SlotMachine({ spinning }: SlotMachineProps) {
  const [reels, setReels] = useState([REELS[0], REELS[1], REELS[2]])

  useEffect(() => {
    if (!spinning) return
    const interval = setInterval(() => {
      setReels([
        REELS[Math.floor(Math.random() * REELS.length)],
        REELS[Math.floor(Math.random() * REELS.length)],
        REELS[Math.floor(Math.random() * REELS.length)],
      ])
    }, 100)
    return () => clearInterval(interval)
  }, [spinning])

  return (
    <div className="flex items-center justify-center gap-4 py-8">
      {reels.map((r, i) => (
        <div
          key={i}
          className="w-20 h-20 border-2 border-primary rounded-lg flex items-center justify-center text-4xl bg-card shadow-lg"
        >
          {r}
        </div>
      ))}
    </div>
  )
}
