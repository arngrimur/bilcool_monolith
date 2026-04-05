import { RouterProvider } from 'react-router-dom'
import { Toaster } from './components/ui/toaster'
import { router } from './router'
import { useThemeSync } from './hooks/useThemeSync'

function App() {
  useThemeSync()
  return (
    <>
      <RouterProvider router={router} />
      <Toaster />
    </>
  )
}

export default App
