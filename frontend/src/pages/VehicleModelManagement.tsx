import { useEffect, useState } from 'react'
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  message,
  Typography,
  Tag,
  Popconfirm,
  Drawer,
} from 'antd'
import { PlusOutlined, DeleteOutlined, EyeOutlined, CodeSandboxOutlined } from '@ant-design/icons'
import api from '../services/api'

const { Title } = Typography
const { TextArea } = Input

const VehicleModelManagement = () => {
  const [vehicleModels, setVehicleModels] = useState([])
  const [programs, setPrograms] = useState([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [drawerVisible, setDrawerVisible] = useState(false)
  const [currentModel, setCurrentModel] = useState<any>(null)
  const [form] = Form.useForm()

  useEffect(() => {
    loadVehicleModels()
  }, [])

  const loadVehicleModels = async () => {
    setLoading(true)
    try {
      const response = await api.get('/vehicle-models')
      setVehicleModels(response.data)
    } catch (error) {
      console.error('Failed to load vehicle models:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = () => {
    setCurrentModel(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (record: any) => {
    setCurrentModel(record)
    form.setFieldsValue(record)
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/vehicle-models/${id}`)
      message.success('删除成功')
      loadVehicleModels()
    } catch (error) {
      console.error('Failed to delete:', error)
    }
  }

  const handleSubmit = async (values: any) => {
    try {
      if (currentModel) {
        await api.put(`/vehicle-models/${currentModel.id}`, values)
        message.success('更新成功')
      } else {
        await api.post('/vehicle-models', values)
        message.success('创建成功')
      }
      setModalVisible(false)
      loadVehicleModels()
    } catch (error) {
      console.error('Failed to submit:', error)
    }
  }

  const handleViewPrograms = async (record: any) => {
    setCurrentModel(record)
    setLoading(true)
    try {
      const response = await api.get(`/programs/by-vehicle/${record.id}`)
      setPrograms(response.data)
      setDrawerVisible(true)
    } catch (error) {
      console.error('Failed to load programs:', error)
      message.error('加载程序列表失败')
    } finally {
      setLoading(false)
    }
  }

  const columns = [
    {
      title: '车型名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '车型编号',
      dataIndex: 'code',
      key: 'code',
    },
    {
      title: '系列',
      dataIndex: 'series',
      key: 'series',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'red'}>
          {status === 'active' ? '活跃' : '停用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: any) => (
        <Space>
          <Button 
            icon={<EyeOutlined />} 
            size="small" 
            onClick={() => handleViewPrograms(record)}
          >
            查看程序
          </Button>
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
        <Title level={2}>车型管理</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          新建车型
        </Button>
      </div>
      <Table
        columns={columns}
        dataSource={vehicleModels}
        rowKey="id"
        loading={loading}
      />

      <Modal
        title={currentModel ? '编辑车型' : '新建车型'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="车型名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="code" label="车型编号" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="series" label="系列">
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={`${currentModel?.name} - 程序列表`}
        placement="right"
        onClose={() => setDrawerVisible(false)}
        open={drawerVisible}
        width={800}
      >
        <Table
          dataSource={programs}
          rowKey="id"
          loading={loading}
          pagination={false}
          columns={[
            {
              title: '程序名称',
              dataIndex: 'name',
              key: 'name',
            },
            {
              title: '程序编号',
              dataIndex: 'code',
              key: 'code',
            },
            {
              title: '生产线',
              dataIndex: ['production_line', 'name'],
              key: 'production_line',
            },
            {
              title: '当前版本',
              dataIndex: 'version',
              key: 'version',
              render: (version: string) => version ? <Tag color="blue">{version}</Tag> : '-',
            },
            {
              title: '状态',
              dataIndex: 'status',
              key: 'status',
              render: (status: string) => (
                <Tag color={status === 'active' ? 'green' : 'red'}>
                  {status === 'active' ? '活跃' : '停用'}
                </Tag>
              ),
            },
          ]}
        />
      </Drawer>
    </div>
  )
}

export default VehicleModelManagement