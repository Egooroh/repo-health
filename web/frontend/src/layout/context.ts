import { useOutletContext } from 'react-router-dom'

export interface LayoutContext {
  demoMode: boolean
  onOpenAnalysis: () => void
  onToggleDemo: () => void
  // Syncs demo state from a fetched project (e.g. a demo repo opened via a
  // direct link) without re-triggering the /demo seed request.
  onSyncDemo: (demo: boolean) => void
  // Effective personal scope: connected, "Мои анализы" selected, and not
  // currently browsing demo data. Pages use this to pick which endpoint
  // family (public vs /personal/...) their data hooks should hit.
  personalMode: boolean
}

export function useLayoutContext() {
  return useOutletContext<LayoutContext>()
}
