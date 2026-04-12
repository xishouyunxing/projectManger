import { useState, useEffect } from 'react';
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
import GlobalSearch from './GlobalSearch';

const { Header, Content } = AntLayout;
const { Title } = Typography;

const Layout = () => {
  const [profileModalVisible, setProfileModalVisible] = useState(false);
  const [searchVisible, setSearchVisible] = useState(false);
  const [passwordForm] = Form.useForm();
  const [passwordLoading, setPasswordLoading] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout, isAdmin } = useAuth();

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

  const menuItems = [
    {
      key: '/dashboard',
      icon: <DashboardOutlined />,
      label: '仪表盘',
    },
    {
      key: '/programs',
      icon: <FileTextOutlined />,
      label: '程序管理',
    },
    {
      key: '/vehicle-models',
      icon: <CarOutlined />,
      label: '车型管理',
    },
    {
      key: '/production-lines',
      icon: <SettingOutlined />,
      label: '生产线管理',
      requiresAdmin: true,
    },
    {
      key: '/users',
      icon: <UserOutlined />,
      label: '用户管理',
      requiresAdmin: true,
    },
    {
      key: '/permissions',
      icon: <LockOutlined />,
      label: '权限管理',
      requiresAdmin: true,
    },
    {
      key: '/system-management',
      icon: <ControlOutlined />,
      label: '系统管理',
      requiresAdmin: true,
    },
  ]
    .filter((item) => !item.requiresAdmin || isAdmin)
    .map(({ requiresAdmin: _requiresAdmin, ...item }) => item);

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
      message.error(error.response?.data?.message || '密码修改失败');
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
        style={{
          margin: '16px',
          padding: 0,
          minHeight: 280,
        }}
      >
        <Outlet />
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
            {user?.role === 'admin' ? '管理员' : '普通用户'}
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

      <GlobalSearch
        open={searchVisible}
        onClose={() => setSearchVisible(false)}
      />
    </AntLayout>
  );
};

export default Layout;
