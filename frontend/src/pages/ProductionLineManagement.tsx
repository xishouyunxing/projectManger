import { useEffect, useState } from 'react'
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
  Tag,
  Popconfirm,
} from 'antd'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import api from '../services/api'

const { Title } = Typography
const { TextArea } = Input

const ProductionLineManagement = () => {
  const [productionLines, setProductionLines] = useState([])
  const [processes, setProcesses] = useState([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [currentLine, setCurrentLine] = useState<any>(null)
  const [form] = Form.useForm()

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      const [linesRes, processesRes] = await Promise.all([
        api.get('/production-lines'),
        api.get('/processes'),
      ])
      setProductionLines(linesRes.data)
      setProcesses(processesRes.data)
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = () => {
    setCurrentLine(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (record: any) => {
    setCurrentLine(record)
    form.setFieldsValue(record)
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/production-lines/${id}`)
      message.success('删除成功')
      loadData()
    } catch (error) {
      console.error('Failed to delete:', error)
    }
  }

  const handleSubmit = async (values: any) => {
    try {
      if (currentLine) {
        await api.put(`/production-lines/${currentLine.id}`, values)
        message.success('更新成功')
      } else {
        await api.post('/production-lines', values)
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
      title: '生产线名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '生产线编号',
      dataIndex: 'code',
      key: 'code',
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => (
        <Tag color={type === 'upper' ? 'blue' : 'green'}>
          {type === 'upper' ? '上车' : '下车'}
        </Tag>
      ),
    },
    {
      title: '所属工序',
      dataIndex: ['process', 'name'],
      key: 'process',
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
        <Title level={2}>生产线管理</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          新建生产线
        </Button>
      </div>
      <Table
        columns={columns}
        dataSource={productionLines}
        rowKey="id"
        loading={loading}
      />

      <Modal
        title={currentLine ? '编辑生产线' : '新建生产线'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="生产线名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="code" label="生产线编号" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="类型" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="upper">上车</Select.Option>
              <Select.Option value="lower">下车</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="process_id" label="所属工序">
            <Select allowClear>
              {processes.map((process: any) => (
                <Select.Option key={process.id} value={process.id}>
                  {process.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={4} />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="active">
            <Select>
              <Select.Option value="active">活跃</Select.Option>
              <Select.Option value="inactive">停用</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default ProductionLineManagement