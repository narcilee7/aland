import {useEffect} from 'react'
import {AnimatePresence, motion} from 'framer-motion'
import {useAland} from './stores/alandStore'
import {Overlook} from './components/Overlook'
import {TribeView} from './components/TribeView'

function App() {
  const view = useAland(s => s.view)
  const boot = useAland(s => s.boot)
  const booted = useAland(s => s.booted)
  const booting = useAland(s => s.booting)

  useEffect(() => {
    boot()
  }, [boot])

  return (
    <div style={{position: 'relative', width: '100%', height: '100%', overflow: 'hidden'}}>
      {/* 俯瞰：始终渲染在底层，做推镜效果时不会被卸载 */}
      <Overlook />

      {/* 部落视图：覆盖在俯瞰之上，推镜转场进入 */}
      <AnimatePresence>
        {view === 'tribe' && (
          <motion.div
            key="tribe"
            initial={{opacity: 0, scale: 0.92}}
            animate={{opacity: 1, scale: 1}}
            exit={{opacity: 0, scale: 0.92}}
            transition={{type: 'spring', stiffness: 120, damping: 20}}
            style={{position: 'absolute', inset: 0, zIndex: 10}}
          >
            <TribeView />
          </motion.div>
        )}
      </AnimatePresence>

      {/* 启动加载 */}
      {!booted && (
        <div
          className="draggable"
          style={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: 'rgba(10,14,26,0.8)',
            zIndex: 100,
            fontFamily: 'var(--font-mono)',
            fontSize: 12,
            color: 'var(--aland-text-dim)',
            letterSpacing: 2,
          }}
        >
          {booting ? 'AWAKENING THE LAND…' : 'STANDBY'}
        </div>
      )}
    </div>
  )
}

export default App
