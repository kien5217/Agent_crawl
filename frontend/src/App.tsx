import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import DocumentsPage from './pages/DocumentsPage'
import DocumentDetailPage from './pages/DocumentDetailPage'
import TopicsPage from './pages/TopicsPage'
import WorkflowsPage from './pages/WorkflowsPage'
import WorkflowStepsPage from './pages/WorkflowStepsPage'
import HealthPage from './pages/HealthPage'
import LabelingPage from './pages/LabelingPage'
import SourcesPage from './pages/SourcesPage'
import DashboardPage from './pages/DashboardPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Navigate to="/documents" replace />} />
          <Route path="documents" element={<DocumentsPage />} />
          <Route path="documents/:id" element={<DocumentDetailPage />} />
          <Route path="topics" element={<TopicsPage />} />
          <Route path="workflows" element={<WorkflowsPage />} />
          <Route path="workflows/:id/steps" element={<WorkflowStepsPage />} />
          <Route path="health" element={<HealthPage />} />
          <Route path="labeling" element={<LabelingPage />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="sources" element={<SourcesPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
