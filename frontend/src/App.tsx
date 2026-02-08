import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import Login from './pages/Login'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import ProgramManagement from './pages/ProgramManagement'
import UserManagement from './pages/UserManagement'
import ProductionLineManagement from './pages/ProductionLineManagement'
import VehicleModelManagement from './pages/VehicleModelManagement'
import PermissionManagement from './pages/PermissionManagement'
import SystemManagement from './pages/SystemManagement'
import { AuthProvider } from './contexts/AuthContext'
import { ThemeProvider } from './contexts/ThemeContext'
import PrivateRoute from './components/PrivateRoute'

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter future={{ 
          v7_startTransition: true, 
          v7_relativeSplatPath: true 
        }}>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/" element={<PrivateRoute><Layout /></PrivateRoute>}>
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<Dashboard />} />
              <Route path="programs" element={<ProgramManagement />} />
              <Route path="users" element={<UserManagement />} />
              <Route path="production-lines" element={<ProductionLineManagement />} />
              <Route path="vehicle-models" element={<VehicleModelManagement />} />
              <Route path="permissions" element={<PermissionManagement />} />
              <Route path="system-management" element={<SystemManagement />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  )
}

export default App
