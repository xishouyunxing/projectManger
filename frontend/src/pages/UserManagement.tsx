import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  message,
  Typography,
  Popconfirm,
  DatePicker,
  ConfigProvider,
  Tooltip,
} from 'antd';
import { PlusOutlined, DeleteOutlined, LockOutlined, UserOutlined, SearchOutlined, EditOutlined } from '@ant-design/icons';
import api, { extractListData, extractPagedListData } from '../services/api';
import { useAuth } from '../contexts/AuthContext';

const { Title } = Typography;

const UserManagement = () => {
  const [searchParams] = useSearchParams();
  const selectedUserId = Number(searchParams.get('id') || 0);
  const [users, setUsers] = useState<any[]>([]);
  const [departments, setDepartments] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [tablePagination, setTablePagination] = useState({ current: 1, pageSize: 20, total: 0 });
  const [modalVisible, setModalVisible] = useState(false);
  const [currentUser, setCurrentUser] = useState<any>(null);
  const [form] = Form.useForm();
  const { isAdmin } = useAuth();

  // 筛选相关状态
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterDepartment, setFilterDepartment] = useState<number | null>(null);
  const [filterRole, setFilterRole] = useState<string | null>(null);
  const [filterStatus, setFilterStatus] = useState<string | null>(null);
  const [filterDateRange, setFilterDateRange] = useState<[string | null, string | null]>([null, null]);

  useEffect(() => {
    loadData();
  }, []);

  useEffect(() => {
    const keyword = searchParams.get('keyword');
    if (keyword) {
      setSearchKeyword(keyword);
    }
  }, [searchParams]);

  const buildUserQueryParams = (page = tablePagination.current, pageSize = tablePagination.pageSize) => ({
    page,
    page_size: pageSize,
    ...(searchKeyword ? { keyword: searchKeyword } : {}),
    ...(filterDepartment ? { department_id: filterDepartment } : {}),
    ...(filterRole ? { role: filterRole } : {}),
    ...(filterStatus ? { status: filterStatus } : {}),
    ...(filterDateRange[0] ? { date_from: filterDateRange[0] } : {}),
    ...(filterDateRange[1] ? { date_to: filterDateRange[1] } : {}),
  });

  const loadData = async (page = tablePagination.current, pageSize = tablePagination.pageSize) => {
    setLoading(true);
    try {
      const [usersRes, departmentsRes] = await Promise.all([
        api.get('/users', { params: buildUserQueryParams(page, pageSize) }),
        api.get('/departments'),
      ]);
      const usersPaged = extractPagedListData(usersRes.data);
      setUsers(usersPaged.items);
      setTablePagination({
        current: usersPaged.page || page,
        pageSize: usersPaged.pageSize || pageSize,
        total: usersPaged.total,
      });
      setDepartments(extractListData(departmentsRes.data));
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadUsers = async (page = tablePagination.current, pageSize = tablePagination.pageSize) => {
    setLoading(true);
    try {
      const response = await api.get('/users', { params: buildUserQueryParams(page, pageSize) });
      const usersPaged = extractPagedListData(response.data);
      const fallbackToPreviousPage = page > 1 && usersPaged.items.length === 0;
      if (fallbackToPreviousPage) {
        await loadUsers(page - 1, pageSize);
        return;
      }
      setUsers(usersPaged.items);
      setTablePagination({
        current: usersPaged.page || page,
        pageSize: usersPaged.pageSize || pageSize,
        total: usersPaged.total,
      });
    } catch (error) {
      console.error('Failed to load users:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = () => {
    setCurrentUser(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: any) => {
    setCurrentUser(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/users/${id}`);
      message.success('删除成功');
      loadUsers(tablePagination.current, tablePagination.pageSize);
    } catch (error) {
      console.error('Failed to delete:', error);
    }
  };

  const handleResetPassword = async (id: number) => {
    try {
      await api.put(`/users/${id}/reset-password`);
      message.success('密码已重置，请通过安全渠道告知用户新密码');
    } catch (error: any) {
      console.error('Failed to reset password:', error);
      message.error(error.response?.data?.error || '重置密码失败');
    }
  };

  const handleSubmit = async (values: any) => {
    try {
      if (currentUser) {
        await api.put(`/users/${currentUser.id}`, values);
        message.success('更新成功');
      } else {
        await api.post('/users', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      loadUsers(tablePagination.current, tablePagination.pageSize);
    } catch (error) {
      console.error('Failed to submit:', error);
    }
  };

  // 重置筛选
  const handleResetFilter = () => {
    setSearchKeyword('');
    setFilterDepartment(null);
    setFilterRole(null);
    setFilterStatus(null);
    setFilterDateRange([null, null]);
  };

  useEffect(() => {
    loadUsers(1, tablePagination.pageSize);
  }, [searchKeyword, filterDepartment, filterRole, filterStatus, filterDateRange]);

  // 筛选后的用户列表
  const sortedUsers = [...users].sort((a: any, b: any) => {
    if (selectedUserId) {
      if (a.id === selectedUserId) return -1;
      if (b.id === selectedUserId) return 1;
    }
    return 0;
  });

  const columns = [
    {
      title: '工号',
      dataIndex: 'employee_id',
      key: 'employee_id',
      render: (text: string) => (
        <span style={{ color: '#2D3335', fontSize: '14px', fontWeight: 700, fontFamily: 'Inter, sans-serif' }}>
          {text}
        </span>
      ),
    },
    {
      title: '姓名',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <span style={{ color: '#5A6062', fontSize: '14px', fontWeight: 500, fontFamily: 'Inter, sans-serif' }}>
          {text}
        </span>
      ),
    },
    {
      title: '部门',
      dataIndex: ['department', 'name'],
      key: 'department',
      render: (text: string) => (
        <div style={{ background: '#EBEEF0', borderRadius: '4px', display: 'inline-block', padding: '2px 8px' }}>
          <span style={{ color: '#2D3335', fontSize: '12px', fontWeight: 700, letterSpacing: '0.6px' }}>
            {text || '-'}
          </span>
        </div>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => {
        const isAdmin = role === 'admin';
        return (
          <div style={{ 
            background: isAdmin ? 'rgba(255, 77, 79, 0.20)' : 'rgba(61, 137, 255, 0.20)', 
            borderRadius: '9999px', 
            display: 'inline-flex', 
            alignItems: 'center',
            padding: '2px 10px',
            gap: '6px'
          }}>
            <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: isAdmin ? '#F53F3F' : '#005BC1' }}></div>
            <span style={{ color: isAdmin ? '#F53F3F' : '#005BC1', fontSize: '11px', fontWeight: 700 }}>
              {isAdmin ? '管理员' : '普通用户'}
            </span>
          </div>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const isActive = status === 'active';
        return (
          <div style={{ 
            background: isActive ? 'rgba(61, 137, 255, 0.20)' : 'rgba(222, 204, 253, 0.40)', 
            borderRadius: '9999px', 
            display: 'inline-flex', 
            alignItems: 'center',
            padding: '2px 10px',
            gap: '6px'
          }}>
            <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: isActive ? '#005BC1' : '#50426B' }}></div>
            <span style={{ color: isActive ? '#005BC1' : '#50426B', fontSize: '11px', fontWeight: 700 }}>
              {isActive ? '正常' : '禁用'}
            </span>
          </div>
        );
      },
    },
    {
      title: '操作',
      key: 'action',
      align: 'right' as const,
      render: (_: any, record: any) => (
        <Space size="small">
          {isAdmin && (
            <>
              <Tooltip title="编辑">
                <Button
                  type="text"
                  icon={<EditOutlined style={{ color: '#5A6062' }} />}
                  onClick={() => handleEdit(record)}
                  style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
                />
              </Tooltip>
              <Popconfirm
                title="确定重置密码?"
                onConfirm={() => handleResetPassword(record.id)}
              >
                <Tooltip title="重置密码">
                  <Button 
                    type="text"
                    icon={<LockOutlined style={{ color: '#5A6062' }} />} 
                    style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
                  />
                </Tooltip>
              </Popconfirm>
              <Popconfirm
                title="确定删除?"
                onConfirm={() => handleDelete(record.id)}
              >
                <Tooltip title="删除">
                  <Button 
                    type="text"
                    icon={<DeleteOutlined style={{ color: '#A83836' }} />} 
                    style={{ width: '32px', height: '32px', borderRadius: '4px', background: 'rgba(168, 56, 54, 0.05)' }}
                  />
                </Tooltip>
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className="management-page">
      {/* 顶部标题区 */}
      <div className="management-page-header">
        <div>
          <div className="management-page-breadcrumb">
            <span>系统</span>
            <span style={{ margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>/</span>
            <span className="active">用户管理</span>
          </div>
          <Title level={2} className="management-page-title">
            用户管理
          </Title>
        </div>
        {isAdmin && (
          <Button 
            type="primary" 
            icon={<PlusOutlined />} 
            onClick={handleAdd}
            style={{
              background: 'linear-gradient(176deg, #005BC1 0%, #3D89FF 100%)',
              border: 'none',
              boxShadow: '0px 4px 6px -4px rgba(0, 91, 193, 0.10), 0px 10px 15px -3px rgba(0, 91, 193, 0.10)',
              borderRadius: '8px',
              height: '44px',
              padding: '0 24px',
              fontWeight: 600,
              fontSize: '16px'
            }}
          >
            新建用户
          </Button>
        )}
      </div>

      {/* 搜索/筛选区域 */}
      <ConfigProvider
        theme={{
          components: {
            Input: {
              controlHeight: 36,
              borderRadius: 8,
              colorBorder: 'transparent',
              colorPrimaryHover: 'transparent',
              controlOutline: 'none',
            },
            Select: {
              controlHeight: 36,
              borderRadius: 8,
              colorBorder: 'transparent',
              colorPrimaryHover: 'transparent',
              controlOutline: 'none',
            },
            DatePicker: {
              controlHeight: 36,
              borderRadius: 8,
              colorBorder: 'transparent',
              colorPrimaryHover: 'transparent',
              controlOutline: 'none',
            }
          }
        }}
      >
        <div className="management-filter-panel">
          <div className="management-filter-field flex">
            <div className="management-filter-label">搜索工号/姓名</div>
            <Input 
              placeholder="搜索工号/姓名" 
              value={searchKeyword} 
              onChange={(e) => setSearchKeyword(e.target.value)}
            />
          </div>
          <div className="management-filter-field">
            <div className="management-filter-label">部门</div>
            <Select 
              placeholder="全部" 
              value={filterDepartment} 
              onChange={setFilterDepartment}
              allowClear
              style={{ width: '100%' }}
            >
              {departments.map((dept: any) => (
                <Select.Option key={dept.id} value={dept.id}>
                  {dept.name}
                </Select.Option>
              ))}
            </Select>
          </div>
          <div className="management-filter-field">
            <div className="management-filter-label">角色</div>
            <Select 
              placeholder="全部" 
              value={filterRole} 
              onChange={setFilterRole}
              allowClear
              style={{ width: '100%' }}
            >
              <Select.Option value="admin">管理员</Select.Option>
              <Select.Option value="user">普通用户</Select.Option>
            </Select>
          </div>
          <div className="management-filter-field">
            <div className="management-filter-label">状态</div>
            <Select 
              placeholder="全部" 
              value={filterStatus} 
              onChange={setFilterStatus}
              allowClear
              style={{ width: '100%' }}
            >
              <Select.Option value="active">正常</Select.Option>
              <Select.Option value="inactive">禁用</Select.Option>
            </Select>
          </div>
          <div className="management-filter-field">
            <div className="management-filter-label">创建日期</div>
            <DatePicker.RangePicker 
              style={{ width: '100%' }}
              onChange={(_, dateStrings) => {
                setFilterDateRange([dateStrings[0] || null, dateStrings[1] || null]);
              }}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <Button 
              icon={<SearchOutlined />}
              style={{ height: '40px', width: '115px', borderRadius: '8px', background: '#DEE3E6', color: '#2D3335', fontWeight: 700, border: 'none' }}
            >
              查询
            </Button>
            <Button 
              type="text"
              onClick={handleResetFilter} 
              style={{ height: '40px', color: '#005BC1', fontWeight: 700, letterSpacing: '1.2px', border: 'none', background: 'transparent' }}
            >
              重置
            </Button>
          </div>
        </div>
      </ConfigProvider>

      <div className="management-table-card">
        <Table
          className="custom-table"
          columns={columns}
          dataSource={sortedUsers}
          rowKey="id"
          loading={loading}
          pagination={{
            current: tablePagination.current,
            pageSize: tablePagination.pageSize,
            total: tablePagination.total,
            onChange: (page, pageSize) => loadUsers(page, pageSize),
            showTotal: (total, range) => `显示第 ${range[0]} 至 ${range[1]} 条，共 ${total} 条记录`,
            style: { padding: '16px 24px', margin: 0, background: 'rgba(241, 244, 245, 0.50)' }
          }}
          locale={{
            emptyText: (
              <div style={{ padding: '40px 0' }}>
                <UserOutlined style={{ fontSize: '48px', color: '#d9d9d9', marginBottom: '16px' }} />
                <div style={{ color: '#999', marginBottom: '16px' }}>
                  暂无用户数据
                </div>
                {isAdmin && (
                  <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
                    创建第一个用户
                  </Button>
                )}
              </div>
            ),
          }}
        />
      </div>

      <Modal
        title={currentUser ? '编辑用户' : '新建用户'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            name="employee_id"
            label="工号"
            rules={[{ required: true }]}
          >
            <Input disabled={!!currentUser} />
          </Form.Item>
          <Form.Item name="name" label="姓名" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="department_id" label="部门">
            <Select allowClear placeholder="请选择部门">
              {departments.map((dept: any) => (
                <Select.Option key={dept.id} value={dept.id}>
                  {dept.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="role" label="角色" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="user">普通用户</Select.Option>
              <Select.Option value="admin">管理员</Select.Option>
            </Select>
          </Form.Item>
          {!currentUser && (
            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true }]}
            >
              <Input.Password />
            </Form.Item>
          )}
          <Form.Item name="status" label="状态" initialValue="active">
            <Select>
              <Select.Option value="active">正常</Select.Option>
              <Select.Option value="inactive">禁用</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default UserManagement;
