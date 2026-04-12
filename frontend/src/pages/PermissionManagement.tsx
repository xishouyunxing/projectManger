import { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Select,
  Switch,
  message,
  Typography,
  Popconfirm,
  Tabs,
  ConfigProvider,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  TeamOutlined,
  UserOutlined,
  EditOutlined,
} from '@ant-design/icons';
import api from '../services/api';

const { Title } = Typography;

const PermissionManagement = () => {
  const [permissions, setPermissions] = useState([]);
  const [deptPermissions, setDeptPermissions] = useState([]);
  const [users, setUsers] = useState([]);
  const [departments, setDepartments] = useState([]);
  const [productionLines, setProductionLines] = useState([]);
  const [loading, setLoading] = useState(false);
  const [deptLoading, setDeptLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [deptModalVisible, setDeptModalVisible] = useState(false);
  const [currentPermission, setCurrentPermission] = useState<any>(null);
  const [currentDeptPermission, setCurrentDeptPermission] = useState<any>(null);
  const [selectedDepartment, setSelectedDepartment] = useState<number | null>(
    null,
  );
  const [form] = Form.useForm();
  const [deptForm] = Form.useForm();

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [permissionsRes, usersRes, linesRes, departmentsRes] =
        await Promise.all([
          api.get('/permissions'),
          api.get('/users'),
          api.get('/production-lines'),
          api.get('/departments'),
        ]);
      setPermissions(permissionsRes.data);
      setUsers(usersRes.data);
      setProductionLines(linesRes.data);
      setDepartments(departmentsRes.data);
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadDeptPermissions = async (departmentId?: number) => {
    setDeptLoading(true);
    try {
      const url = departmentId
        ? `/department-permissions?department_id=${departmentId}`
        : '/department-permissions';
      const response = await api.get(url);
      setDeptPermissions(response.data);
    } catch (error) {
      console.error('Failed to load department permissions:', error);
    } finally {
      setDeptLoading(false);
    }
  };

  const handleDepartmentChange = (value: number | null) => {
    setSelectedDepartment(value);
    loadDeptPermissions(value || undefined);
  };

  const handleAdd = () => {
    setCurrentPermission(null);
    form.resetFields();
    form.setFieldsValue({
      can_view: true,
      can_download: false,
      can_upload: false,
      can_manage: false,
    });
    setModalVisible(true);
  };

  const handleEdit = (record: any) => {
    setCurrentPermission(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/permissions/${id}`);
      message.success('删除成功');
      loadData();
    } catch (error) {
      console.error('Failed to delete:', error);
    }
  };

  const handleSubmit = async (values: any) => {
    try {
      if (currentPermission) {
        await api.put(`/permissions/${currentPermission.id}`, values);
        message.success('更新成功');
      } else {
        await api.post('/permissions', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      loadData();
    } catch (error) {
      console.error('Failed to submit:', error);
    }
  };

  const handleDeptAdd = () => {
    setCurrentDeptPermission(null);
    deptForm.resetFields();
    deptForm.setFieldsValue({
      can_view: true,
      can_download: false,
      can_upload: false,
      can_manage: false,
    });
    setDeptModalVisible(true);
  };

  const handleDeptEdit = (record: any) => {
    setCurrentDeptPermission(record);
    deptForm.setFieldsValue(record);
    setDeptModalVisible(true);
  };

  const handleDeptDelete = async (id: number) => {
    try {
      await api.delete(`/department-permissions/${id}`);
      message.success('删除成功');
      loadDeptPermissions(selectedDepartment || undefined);
    } catch (error) {
      console.error('Failed to delete:', error);
    }
  };

  const handleDeptSubmit = async (values: any) => {
    try {
      if (currentDeptPermission) {
        await api.put(
          `/department-permissions/${currentDeptPermission.id}`,
          values,
        );
        message.success('更新成功');
      } else {
        await api.post('/department-permissions', values);
        message.success('创建成功');
      }
      setDeptModalVisible(false);
      loadDeptPermissions(selectedDepartment || undefined);
    } catch (error) {
      console.error('Failed to submit:', error);
    }
  };

  const filteredPermissions = selectedDepartment
    ? permissions.filter((p: any) => {
        const user = users.find((u: any) => u.id === p.user_id) as any;
        return user?.department_id === selectedDepartment;
      })
    : permissions;

  const columns = [
    {
      title: '用户',
      dataIndex: ['user', 'name'],
      key: 'user',
      render: (text: string, record: any) => (
        <Space>
          <UserOutlined style={{ color: '#005BC1' }} />
          <span style={{ color: '#2D3335', fontSize: '14px', fontWeight: 700, fontFamily: 'Inter, sans-serif' }}>
            {text}
          </span>
          <span style={{ color: '#5A6062', fontSize: '12px' }}>
            ({record.user.employee_id})
          </span>
          {record.user.department && (
            <div style={{ background: '#EBEEF0', borderRadius: '4px', display: 'inline-block', padding: '2px 8px' }}>
              <span style={{ color: '#2D3335', fontSize: '11px', fontWeight: 600 }}>
                {record.user.department.name}
              </span>
            </div>
          )}
        </Space>
      ),
    },
    {
      title: '生产线',
      dataIndex: ['production_line', 'name'],
      key: 'production_line',
      render: (text: string) => (
        <span style={{ color: '#5A6062', fontSize: '14px', fontWeight: 500, fontFamily: 'Inter, sans-serif' }}>
          {text}
        </span>
      ),
    },
    {
      title: '查看',
      dataIndex: 'can_view',
      key: 'can_view',
      render: (value: boolean) => (
        <Switch checked={value} disabled size="small" />
      ),
    },
    {
      title: '下载',
      dataIndex: 'can_download',
      key: 'can_download',
      render: (value: boolean) => (
        <Switch checked={value} disabled size="small" />
      ),
    },
    {
      title: '上传',
      dataIndex: 'can_upload',
      key: 'can_upload',
      render: (value: boolean) => (
        <Switch checked={value} disabled size="small" />
      ),
    },
    {
      title: '管理',
      dataIndex: 'can_manage',
      key: 'can_manage',
      render: (value: boolean) => (
        <Switch checked={value} disabled size="small" />
      ),
    },
    {
      title: '操作',
      key: 'action',
      align: 'right' as const,
      render: (_: any, record: any) => (
        <Space size="small">
          <Tooltip title="编辑权限">
            <Button
              type="text"
              icon={<EditOutlined style={{ color: '#5A6062' }} />}
              onClick={() => handleEdit(record)}
              style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
            />
          </Tooltip>
          <Popconfirm
            title="确定删除?"
            onConfirm={() => handleDelete(record.id)}
          >
            <Tooltip title="删除权限">
              <Button
                type="text"
                icon={<DeleteOutlined style={{ color: '#A83836' }} />}
                style={{ width: '32px', height: '32px', borderRadius: '4px', background: 'rgba(168, 56, 54, 0.05)' }}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const deptColumns = [
    {
      title: '部门',
      dataIndex: ['department', 'name'],
      key: 'department',
      render: (text: string) => (
        <Space>
          <TeamOutlined style={{ color: '#005BC1' }} />
          <span style={{ color: '#2D3335', fontSize: '14px', fontWeight: 700, fontFamily: 'Inter, sans-serif' }}>
            {text}
          </span>
        </Space>
      ),
    },
    {
      title: '生产线',
      dataIndex: ['production_line', 'name'],
      key: 'production_line',
      render: (text: string) => (
        <span style={{ color: '#5A6062', fontSize: '14px', fontWeight: 500, fontFamily: 'Inter, sans-serif' }}>
          {text}
        </span>
      ),
    },
    {
      title: '查看',
      dataIndex: 'can_view',
      key: 'can_view',
      render: (value: boolean) => (
        <Switch checked={value} disabled size="small" />
      ),
    },
    {
      title: '下载',
      dataIndex: 'can_download',
      key: 'can_download',
      render: (value: boolean) => (
        <Switch checked={value} disabled size="small" />
      ),
    },
    {
      title: '上传',
      dataIndex: 'can_upload',
      key: 'can_upload',
      render: (value: boolean) => (
        <Switch checked={value} disabled size="small" />
      ),
    },
    {
      title: '管理',
      dataIndex: 'can_manage',
      key: 'can_manage',
      render: (value: boolean) => (
        <Switch checked={value} disabled size="small" />
      ),
    },
    {
      title: '操作',
      key: 'action',
      align: 'right' as const,
      render: (_: any, record: any) => (
        <Space size="small">
          <Tooltip title="编辑权限">
            <Button
              type="text"
              icon={<EditOutlined style={{ color: '#5A6062' }} />}
              onClick={() => handleDeptEdit(record)}
              style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
            />
          </Tooltip>
          <Popconfirm
            title="确定删除?"
            onConfirm={() => handleDeptDelete(record.id)}
          >
            <Tooltip title="删除权限">
              <Button
                type="text"
                icon={<DeleteOutlined style={{ color: '#A83836' }} />}
                style={{ width: '32px', height: '32px', borderRadius: '4px', background: 'rgba(168, 56, 54, 0.05)' }}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="management-page">
      <div className="management-page-header">
        <div>
          <div className="management-page-breadcrumb">
            <span>系统</span>
            <span style={{ margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>/</span>
            <span className="active">权限管理</span>
          </div>
          <Title level={2} className="management-page-title">
            权限管理
          </Title>
        </div>
      </div>

      <ConfigProvider
        theme={{
          components: {
            Select: {
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
          <div className="management-filter-field">
            <div className="management-filter-label">按部门筛选</div>
            <Select
              style={{ width: '100%' }}
              placeholder="全部门"
              allowClear
              value={selectedDepartment}
              onChange={handleDepartmentChange}
            >
              {departments.map((dept: any) => (
                <Select.Option key={dept.id} value={dept.id}>
                  {dept.name}
                </Select.Option>
              ))}
            </Select>
          </div>
        </div>
      </ConfigProvider>

      <Tabs
        items={[
          {
            key: 'user',
            label: (
              <span>
                <UserOutlined />
                用户权限
              </span>
            ),
            children: (
              <>
                <div
                  style={{
                    marginBottom: 16,
                    display: 'flex',
                    justifyContent: 'flex-end',
                  }}
                >
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={handleAdd}
                  >
                    新建用户权限
                  </Button>
                </div>
                <div className="management-table-card">
                  <Table
                  columns={columns}
                  dataSource={filteredPermissions}
                  rowKey="id"
                  loading={loading}
                  className="custom-table"
                />
                </div>
              </>
            ),
          },
          {
            key: 'department',
            label: (
              <span>
                <TeamOutlined />
                部门权限
              </span>
            ),
            children: (
              <>
                <div
                  style={{
                    marginBottom: 16,
                    display: 'flex',
                    justifyContent: 'flex-end',
                  }}
                >
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={handleDeptAdd}
                  >
                    新建部门权限
                  </Button>
                </div>
                <div className="management-table-card">
                  <Table
                  columns={deptColumns}
                  dataSource={deptPermissions}
                  rowKey="id"
                  loading={deptLoading}
                  className="custom-table"
                />
                </div>
              </>
            ),
          },
        ]}
        onChange={(key) => {
          if (key === 'department') {
            loadDeptPermissions(selectedDepartment || undefined);
          }
        }}
      />

      {/* 用户权限模态框 */}
      <Modal
        title={currentPermission ? '编辑用户权限' : '新建用户权限'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            name="user_id"
            label="用户"
            rules={[{ required: true, message: '请选择用户' }]}
          >
            <Select
              showSearch
              optionFilterProp="children"
              disabled={!!currentPermission}
              placeholder="请选择用户"
            >
              {users.map((user: any) => (
                <Select.Option key={user.id} value={user.id}>
                  {user.name} ({user.employee_id}) -{' '}
                  {user.department?.name || '无部门'}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="production_line_id"
            label="生产线"
            rules={[{ required: true, message: '请选择生产线' }]}
          >
            <Select disabled={!!currentPermission} placeholder="请选择生产线">
              {productionLines.map((line: any) => (
                <Select.Option key={line.id} value={line.id}>
                  {line.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="can_view" label="查看权限" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            name="can_download"
            label="下载权限"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
          <Form.Item name="can_upload" label="上传权限" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="can_manage" label="管理权限" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* 部门权限模态框 */}
      <Modal
        title={currentDeptPermission ? '编辑部门权限' : '新建部门权限'}
        open={deptModalVisible}
        onCancel={() => setDeptModalVisible(false)}
        onOk={() => deptForm.submit()}
      >
        <Form form={deptForm} layout="vertical" onFinish={handleDeptSubmit}>
          <Form.Item
            name="department_id"
            label="部门"
            rules={[{ required: true, message: '请选择部门' }]}
          >
            <Select disabled={!!currentDeptPermission} placeholder="请选择部门">
              {departments.map((dept: any) => (
                <Select.Option key={dept.id} value={dept.id}>
                  {dept.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="production_line_id"
            label="生产线"
            rules={[{ required: true, message: '请选择生产线' }]}
          >
            <Select
              disabled={!!currentDeptPermission}
              placeholder="请选择生产线"
            >
              {productionLines.map((line: any) => (
                <Select.Option key={line.id} value={line.id}>
                  {line.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="can_view" label="查看权限" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            name="can_download"
            label="下载权限"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
          <Form.Item name="can_upload" label="上传权限" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="can_manage" label="管理权限" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default PermissionManagement;
