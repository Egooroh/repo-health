import { Route, Routes } from 'react-router-dom'
import AppLayout from './layout/AppLayout'
import LeaderboardPage from './pages/LeaderboardPage'
import MethodologyPage from './pages/MethodologyPage'
import RepositoriesPage from './pages/RepositoriesPage'
import RepositoryDetailPage from './pages/RepositoryDetailPage'

export default function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<RepositoriesPage />} />
        <Route path="repositories" element={<RepositoriesPage />} />
        <Route path="repositories/:id" element={<RepositoryDetailPage />} />
        <Route path="leaderboard" element={<LeaderboardPage />} />
        <Route path="methodology" element={<MethodologyPage />} />
      </Route>
    </Routes>
  )
}
