import { Suspense, lazy, useState, useEffect, useRef } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  Layout as AntLayout,
  Menu,
  Avatar,
  Dropdown,
  Typography,
  Space,
  Button,
  Modal,
  Form,
  Input,
  message,
  Divider,
  Tooltip,
} from 'antd';
import {
  DashboardOutlined,
  FileTextOutlined,
  CarOutlined,
  SettingOutlined,
  UserOutlined,
  LockOutlined,
  LogoutOutlined,
  ControlOutlined,
  KeyOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { useAuth } from '../contexts/AuthContext';
import api from '../services/api';

const GlobalSearch = lazy(() => import('./GlobalSearch'));

const { Header, Content } = AntLayout;
const { Title } = Typography;

const Layout = () => {
  const [profileModalVisible, setProfileModalVisible] = useState(false);
  const [searchVisible, setSearchVisible] = useState(false);
  const [routeTransitioning, setRouteTransitioning] = useState(false);
  const prefetchedRoutesRef = useRef(new Set<string>());
  const [passwordForm] = Form.useForm();
  const [passwordLoading, setPasswordLoading] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout, isAdmin, hasPermission } = useAuth();

  // 全局快捷键 Ctrl+K
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        setSearchVisible(true);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  useEffect(() => {
    setRouteTransitioning(true);
    const timer = window.setTimeout(() => setRouteTransitioning(false), 150);
    return () => window.clearTimeout(timer);
  }, [location.pathname]);

  const routePrefetchers: Record<string, () => Promise<unknown>> = {
    '/dashboard': () => import('../pages/Dashboard'),
    '/programs': () => import('../pages/ProgramManagement'),
    '/vehicle-models': () => import('../pages/VehicleModelManagement'),
    '/production-lines': () => import('../pages/ProductionLineManagement'),
    '/users': () => import('../pages/UserManagement'),
    '/permissions': () => import('../pages/PermissionManagement'),
    '/system-management': () => import('../pages/SystemManagement'),
  };

  const prefetchRoute = (routePath: string) => {
    if (prefetchedRoutesRef.current.has(routePath)) {
      return;
    }
    const loader = routePrefetchers[routePath];
    if (!loader) {
      return;
    }
    prefetchedRoutesRef.current.add(routePath);
    void loader().catch(() => {
      prefetchedRoutesRef.current.delete(routePath);
    });
  };

  const routeLabel = (path: string, text: string) => (
    <span onMouseEnter={() => prefetchRoute(path)}>{text}</span>
  );

  const menuItems = [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: routeLabel('/dashboard', '仪表盘'),
    },
    {
      key: '/programs',
      icon: <FileTextOutlined />,
      label: routeLabel('/programs', '程序管理'),
      permission: 'page:programs',
    },
    {
      key: '/vehicle-models',
      icon: <CarOutlined />,
      label: routeLabel('/vehicle-models', '车型管理'),
      permission: 'page:vehicle_models',
    },
    {
      key: '/production-lines',
      icon: <SettingOutlined />,
      label: routeLabel('/production-lines', '生产线管理'),
      permission: 'page:production_lines',
    },
    {
      key: '/users',
      icon: <UserOutlined />,
      label: routeLabel('/users', '用户管理'),
      permission: 'page:user_management',
    },
    {
      key: '/permissions',
      icon: <LockOutlined />,
      label: routeLabel('/permissions', '权限管理'),
      permission: 'page:permissions',
    },
    {
      key: '/system-management',
      icon: <ControlOutlined />,
      label: routeLabel('/system-management', '系统管理'),
      permission: 'page:system_management',
    },
  ]
    .filter((item) => !item.permission || isAdmin || hasPermission(item.permission))
    .map(({ permission: _permission, ...item }) => item);

  const roleDisplayMap: Record<string, string> = {
    admin: '管理员',
    system_admin: '系统管理员',
    line_admin: '产线管理员',
    engineer: '工程师',
    operator: '操作员',
    viewer: '查看者',
  };

  const userMenuItems: MenuProps['items'] = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人信息',
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      danger: true,
    },
  ];

  const handleMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') {
      logout();
      navigate('/login');
    } else {
      prefetchRoute(key);
      navigate(key);
    }
  };

  const handleUserMenuClick = ({ key }: { key: string }) => {
    if (key === 'logout') {
      logout();
      navigate('/login');
    } else if (key === 'profile') {
      setProfileModalVisible(true);
    }
  };

  const handlePasswordChange = async (values: any) => {
    try {
      setPasswordLoading(true);
      await api.put(`/users/${user?.id}/password`, {
        old_password: values.oldPassword,
        new_password: values.newPassword,
      });
      message.success('密码修改成功');
      setProfileModalVisible(false);
      passwordForm.resetFields();
    } catch (error: any) {
      message.error(error.response?.data?.error || error.response?.data?.message || '密码修改失败');
    } finally {
      setPasswordLoading(false);
    }
  };

  return (
    <AntLayout style={{ minHeight: '100vh', background: '#f7f8fa' }}>
      <Header
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          background: '#ffffff',
          padding: '0 24px',
          boxShadow: '0 1px 4px rgba(0, 0, 0, 0.08)',
          position: 'sticky',
          top: 0,
          zIndex: 100,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '32px' }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              cursor: 'pointer',
            }}
            onClick={() => navigate('/dashboard')}
          >
            <div
              style={{
                width: '32px',
                height: '32px',
                background: 'linear-gradient(135deg, #165dff 0%, #0e42d2 100%)',
                borderRadius: '8px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <ControlOutlined style={{ color: '#fff', fontSize: '18px' }} />
            </div>
            <span
              style={{
                fontSize: '16px',
                fontWeight: 600,
                color: '#1d2129',
              }}
            >
              程序管理系统
            </span>
          </div>

          <Menu
            mode="horizontal"
            selectedKeys={[location.pathname]}
            items={menuItems}
            onClick={handleMenuClick}
            className="app-top-menu"
            style={{
              border: 'none',
              background: 'transparent',
              flex: 1,
              minWidth: 0,
            }}
          />
        </div>

        <Space size="middle">
          <Tooltip title="搜索 (Ctrl+K)">
            <Button
              type="text"
              icon={<SearchOutlined />}
              onClick={() => setSearchVisible(true)}
              style={{
                width: '36px',
                height: '36px',
                borderRadius: '8px',
              }}
            />
          </Tooltip>

          <Dropdown
            menu={{
              items: userMenuItems,
              onClick: handleUserMenuClick,
            }}
            placement="bottomRight"
            arrow
          >
            <div className="user-avatar-area" style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}>
              <Avatar
                size="small"
                style={{
                  background: '#165dff',
                }}
              >
                {user?.name?.charAt(0) || 'U'}
              </Avatar>
              <span style={{ fontSize: '14px', color: '#1d2129' }}>
                {user?.name || '用户'}
              </span>
            </div>
          </Dropdown>
        </Space>
      </Header>

      <Content
        className="app-content-shell"
        style={{
          margin: '16px',
          padding: 0,
          minHeight: 280,
        }}
      >
        <div className={`app-route-content${routeTransitioning ? ' is-transitioning' : ''}`}>
          <Outlet />
        </div>
      </Content>

      <Modal
        title={
          <Space>
            <KeyOutlined />
            个人信息
          </Space>
        }
        open={profileModalVisible}
        onCancel={() => {
          setProfileModalVisible(false);
          passwordForm.resetFields();
        }}
        footer={null}
        width={400}
      >
        <div style={{ marginBottom: 24 }}>
          <div style={{ marginBottom: 12 }}>
            <strong>姓名:</strong> {user?.name}
          </div>
          <div style={{ marginBottom: 12 }}>
            <strong>工号:</strong> {user?.employee_id}
          </div>
          <div style={{ marginBottom: 12 }}>
            <strong>部门:</strong> {user?.department?.name || '-'}
          </div>
          <div>
            <strong>角色:</strong>{' '}
            {roleDisplayMap[user?.role || ''] || user?.role || '-'}
          </div>
        </div>

        <Divider />
        <Title level={5}>修改密码</Title>
        <Form
          form={passwordForm}
          onFinish={handlePasswordChange}
          layout="vertical"
        >
          <Form.Item
            name="oldPassword"
            label="原密码"
            rules={[{ required: true, message: '请输入原密码' }]}
          >
            <Input.Password placeholder="请输入原密码" />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码至少6位' },
            ]}
          >
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="确认密码"
            dependencies={['newPassword']}
            rules={[
              { required: true, message: '请确认新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password placeholder="请再次输入新密码" />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setProfileModalVisible(false)}>
                取消
              </Button>
              <Button
                type="primary"
                htmlType="submit"
                loading={passwordLoading}
              >
                确认修改
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {searchVisible ? (
        <Suspense fallback={null}>
          <GlobalSearch
            open={searchVisible}
            onClose={() => setSearchVisible(false)}
          />
        </Suspense>
      ) : null}
    </AntLayout>
  );
};

export default Layout;
