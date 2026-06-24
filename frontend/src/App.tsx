import { useEffect } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useAland } from './stores/alandStore'
import { Overlook } from './components/Overlook'
import { TribeView } from './components/TribeView'
import { Spotlight } from './components/Spotlight'
import { Forge } from './components/Forge'
import { CapabilityMatrix } from './components/CapabilityMatrix'
import { Loader2 } from 'lucide-react'
import { TooltipProvider } from './components/ui'

function App() {
  const view = useAland(s => s.view)
  const boot = useAland(s => s.boot)
  const booted = useAland(s => s.booted)
  const booting = useAland(s => s.booting)
  const forgeOpen = useAland(s => s.forgeOpen)
  const setForgeOpen = useAland(s => s.setForgeOpen)
  const matrixOpen = useAland(s => s.matrixOpen)
  const setMatrixOpen = useAland(s => s.setMatrixOpen)

  useEffect(() => {
    boot()
  }, [boot])

  return (
    <TooltipProvider delayDuration={300}>
      <div className="relative w-full h-full overflow-hidden bg-land">
        {/* 俯瞰 */}
        <Overlook
          onOpenForge={() => setForgeOpen(true)}
          onOpenMatrix={() => setMatrixOpen(true)}
          onOpenSpotlight={() => useAland.getState().setSpotlight(true)}
        />

        {/* 部落视图 */}
        <AnimatePresence>
          {view === 'tribe' && (
            <motion.div
              key="tribe"
              initial={{ opacity: 0, scale: 0.92 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.92 }}
              transition={{ type: 'spring', stiffness: 120, damping: 20 }}
              className="absolute inset-0 z-10"
            >
              <TribeView />
            </motion.div>
          )}
        </AnimatePresence>

        {/* Forge */}
        <Forge open={forgeOpen} onOpenChange={setForgeOpen} />

        {/* Capability Matrix */}
        <CapabilityMatrix open={matrixOpen} onOpenChange={setMatrixOpen} />

        {/* Spotlight */}
        <Spotlight />

        {/* 启动加载 */}
        {!booted && (
          <div className="draggable absolute inset-0 z-40 flex flex-col items-center justify-center bg-land-1/80 backdrop-blur-sm font-mono text-xs uppercase tracking-widest text-ink-dim gap-2">
            <div className="flex items-center">
              <Loader2 className="mr-2 h-3 w-3 animate-spin" />
              {booting ? 'Awakening the land…' : 'Wails runtime not detected'}
            </div>
          </div>
        )}
      </div>
    </TooltipProvider>
  )
}

export default App
