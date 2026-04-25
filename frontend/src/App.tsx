import { Suspense, lazy } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { AuthProvider } from './contexts/AuthContext';
import { ThemeProvider } from './contexts/ThemeContext';
import PrivateRoute from './components/PrivateRoute';
import AdminRoute from './components/AdminRoute';

const Layout = lazy(() => import('./components/Layout'));
const Login = lazy(() => import('./pages/Login'));
const Dashboard = lazy(() => import('./pages/Dashboard'));
const ProgramManagement = lazy(() => import('./pages/ProgramManagement'));
const UserManagement = lazy(() => import('./pages/UserManagement'));
const ProductionLineManagement = lazy(
  () => import('./pages/ProductionLineManagement'),
);
const VehicleModelManagement = lazy(
  () => import('./pages/VehicleModelManagement'),
);
const PermissionManagement = lazy(() => import('./pages/PermissionManagement'));
const SystemManagement = lazy(() => import('./pages/SystemManagement'));
const FileIgnoreList = lazy(() => import('./pages/FileIgnoreList'));
const ProgramMatrixPreview = lazy(() => import('./pages/ProgramMatrixPreview'));

// 设置dayjs语言为中文
dayjs.locale('zh-cn');

function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <ThemeProvider>
        <AuthProvider>
          <Suspense
            fallback={
              <div style={{ padding: 40, textAlign: 'center' }}>
                页面加载中...
              </div>
            }
          >
            <BrowserRouter
              future={{
                v7_startTransition: true,
                v7_relativeSplatPath: true,
              }}
            >
              <Routes>
                <Route path="/login" element={<Login />} />
                <Route
                  path="/"
                  element={
                    <PrivateRoute>
                      <Layout />
                    </PrivateRoute>
                  }
                >
                  <Route index element={<Navigate to="/dashboard" replace />} />
                  <Route path="dashboard" element={<Dashboard />} />
                  <Route path="programs" element={<ProgramManagement />} />
                  <Route
                    path="users"
                    element={
                      <AdminRoute>
                        <UserManagement />
                      </AdminRoute>
                    }
                  />
                  <Route
                    path="production-lines"
                    element={
                      <AdminRoute>
                        <ProductionLineManagement />
                      </AdminRoute>
                    }
                  />
                  <Route
                    path="vehicle-models"
                    element={
                      <AdminRoute>
                        <VehicleModelManagement />
                      </AdminRoute>
                    }
                  />
                  <Route
                    path="permissions"
                    element={
                      <AdminRoute>
                        <PermissionManagement />
                      </AdminRoute>
                    }
                  />
                  <Route path="file-ignore-list" element={<FileIgnoreList />} />
                  <Route
                    path="system-management"
                    element={
                      <AdminRoute>
                        <SystemManagement />
                      </AdminRoute>
                    }
                  />
                  <Route
                    path="program-matrix"
                    element={<ProgramMatrixPreview />}
                  />
                </Route>
              </Routes>
            </BrowserRouter>
          </Suspense>
        </AuthProvider>
      </ThemeProvider>
    </ConfigProvider>
  );
}

export default App;
