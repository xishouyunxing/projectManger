import { Suspense, lazy } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { AuthProvider } from './contexts/AuthContext';
import { ThemeProvider } from './contexts/ThemeContext';
import PrivateRoute from './components/PrivateRoute';
import PermissionRoute from './components/PermissionRoute';

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
                      <PermissionRoute permission="page:user_management">
                        <UserManagement />
                      </PermissionRoute>
                    }
                  />
                  <Route
                    path="production-lines"
                    element={
                      <PermissionRoute permission="page:production_lines">
                        <ProductionLineManagement />
                      </PermissionRoute>
                    }
                  />
                  <Route
                    path="vehicle-models"
                    element={
                      <PermissionRoute permission="page:vehicle_models">
                        <VehicleModelManagement />
                      </PermissionRoute>
                    }
                  />
                  <Route
                    path="permissions"
                    element={
                      <PermissionRoute permission="page:permissions">
                        <PermissionManagement />
                      </PermissionRoute>
                    }
                  />
                  <Route path="file-ignore-list" element={<FileIgnoreList />} />
                  <Route
                    path="system-management"
                    element={
                      <PermissionRoute permission="page:system_management">
                        <SystemManagement />
                      </PermissionRoute>
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
