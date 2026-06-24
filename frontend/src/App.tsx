import { useEffect } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useAland } from './stores/alandStore'
import { Overlook } from './components/Overlook'
import { TribeView } from './components/TribeView'
import { Loader2 } from 'lucide-react'

function App() {
  const view = useAland(s => s.view)
  const boot = useAland(s => s.boot)
  const booted = useAland(s => s.booted)
  const booting = useAland(s => s.booting)

  useEffect(() => {
    boot()
  }, [boot])

  return (
    <div className="relative w-full h-full overflow-hidden bg-land">
      {/* 俯瞰：始终渲染在底层，做推镜效果时不会被卸载 */}
      <Overlook />

      {/* 部落视图：覆盖在俯瞰之上，推镜转场进入 */}
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

      {/* 启动加载 */}
      {!booted && (
        <div className="draggable absolute inset-0 z-50 flex items-center justify-center bg-land-1/80 backdrop-blur-sm font-mono text-xs uppercase tracking-widest text-ink-dim">
          <Loader2 className="mr-2 h-3 w-3 animate-spin" />
          {booting ? 'Awakening the land…' : 'Standby'}
        </div>
      )}
    </div>
  )
}

export default App
