import { useEffect, useState } from 'react'
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
} from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import api from '../services/api'

const { Title } = Typography

const PermissionManagement = () => {
  const [permissions, setPermissions] = useState([])
  const [users, setUsers] = useState([])
  const [productionLines, setProductionLines] = useState([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [currentPermission, setCurrentPermission] = useState<any>(null)
  const [form] = Form.useForm()

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      const [permissionsRes, usersRes, linesRes] = await Promise.all([
        api.get('/permissions'),
        api.get('/users'),
        api.get('/production-lines'),
      ])
      setPermissions(permissionsRes.data)
      setUsers(usersRes.data)
      setProductionLines(linesRes.data)
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = () => {
    setCurrentPermission(null)
    form.resetFields()
    form.setFieldsValue({
      can_view: true,
      can_download: false,
      can_upload: false,
      can_manage: false,
    })
    setModalVisible(true)
  }

  const handleEdit = (record: any) => {
    setCurrentPermission(record)
    form.setFieldsValue(record)
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/permissions/${id}`)
      message.success('删除成功')
      loadData()
    } catch (error) {
      console.error('Failed to delete:', error)
    }
  }

  const handleSubmit = async (values: any) => {
    try {
      if (currentPermission) {
        await api.put(`/permissions/${currentPermission.id}`, values)
        message.success('更新成功')
      } else {
        await api.post('/permissions', values)
        message.success('创建成功')
      }
      setModalVisible(false)
      loadData()
    } catch (error) {
      console.error('Failed to submit:', error)
    }
  }

  const columns = [
    {
      title: '用户',
      dataIndex: ['user', 'name'],
      key: 'user',
      render: (text: string, record: any) => `${text} (${record.user.employee_id})`,
    },
    {
      title: '生产线',
      dataIndex: ['production_line', 'name'],
      key: 'production_line',
    },
    {
      title: '查看',
      dataIndex: 'can_view',
      key: 'can_view',
      render: (value: boolean) => <Switch checked={value} disabled />,
    },
    {
      title: '下载',
      dataIndex: 'can_download',
      key: 'can_download',
      render: (value: boolean) => <Switch checked={value} disabled />,
    },
    {
      title: '上传',
      dataIndex: 'can_upload',
      key: 'can_upload',
      render: (value: boolean) => <Switch checked={value} disabled />,
    },
    {
      title: '管理',
      dataIndex: 'can_manage',
      key: 'can_manage',
      render: (value: boolean) => <Switch checked={value} disabled />,
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: any) => (
        <Space>
          <Button type="primary" size="small" onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm title="确定删除?" onConfirm={() => handleDelete(record.id)}>
            <Button danger icon={<DeleteOutlined />} size="small">删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Title level={2}>权限管理</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          新建权限
        </Button>
      </div>
      <Table
        columns={columns}
        dataSource={permissions}
        rowKey="id"
        loading={loading}
      />

      <Modal
        title={currentPermission ? '编辑权限' : '新建权限'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="user_id" label="用户" rules={[{ required: true }]}>
            <Select
              showSearch
              optionFilterProp="children"
              disabled={!!currentPermission}
            >
              {users.map((user: any) => (
                <Select.Option key={user.id} value={user.id}>
                  {user.name} ({user.employee_id})
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="production_line_id" label="生产线" rules={[{ required: true }]}>
            <Select disabled={!!currentPermission}>
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
          <Form.Item name="can_download" label="下载权限" valuePropName="checked">
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
  )
}

export default PermissionManagement
