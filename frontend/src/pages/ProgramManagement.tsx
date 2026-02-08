import { useEffect, useState } from 'react'
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Upload,
  message,
  Typography,
  Tag,
  Popconfirm,
  Collapse,
  Timeline,
  Badge,
  Tooltip,
} from 'antd'
import {
  PlusOutlined,
  UploadOutlined,
  DownloadOutlined,
  DeleteOutlined,
  EyeOutlined,
  FileOutlined,
  HistoryOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons'
import api from '../services/api'

const { Title } = Typography
const { TextArea } = Input
const { Panel } = Collapse

const ProgramManagement = () => {
  const [programs, setPrograms] = useState([])
  const [productionLines, setProductionLines] = useState([])
  const [vehicleModels, setVehicleModels] = useState([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [uploadModalVisible, setUploadModalVisible] = useState(false)
  const [fileModalVisible, setFileModalVisible] = useState(false)
  const [currentProgram, setCurrentProgram] = useState<any>(null)
  const [versions, setVersions] = useState<any[]>([])
  const [form] = Form.useForm()
  const [uploadForm] = Form.useForm()

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      const [programsRes, linesRes, modelsRes] = await Promise.all([
        api.get('/programs'),
        api.get('/production-lines'),
        api.get('/vehicle-models'),
      ])
      setPrograms(programsRes.data)
      setProductionLines(linesRes.data)
      setVehicleModels(modelsRes.data)
    } catch (error) {
      console.error('Failed to load data:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleAdd = () => {
    setCurrentProgram(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (record: any) => {
    setCurrentProgram(record)
    form.setFieldsValue(record)
    setModalVisible(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/programs/${id}`)
      message.success('删除成功')
      loadData()
    } catch (error) {
      console.error('Failed to delete:', error)
    }
  }

  const handleSubmit = async (values: any) => {
    try {
      if (currentProgram) {
        await api.put(`/programs/${currentProgram.id}`, values)
        message.success('更新成功')
      } else {
        await api.post('/programs', values)
        message.success('创建成功')
      }
      setModalVisible(false)
      loadData()
    } catch (error) {
      console.error('Failed to submit:', error)
    }
  }

  const handleUpload = (record: any) => {
    setCurrentProgram(record)
    uploadForm.resetFields()
    uploadForm.setFieldValue('program_id', record.id)
    setUploadModalVisible(true)
  }

  const handleReupload = (record: any) => {
    setCurrentProgram(record)
    uploadForm.resetFields()
    uploadForm.setFieldValue('program_id', record.id)
    uploadForm.setFieldValue('version', record.version || 'v1.0.0')
    setUploadModalVisible(true)
  }

  const handleUploadSubmit = async (values: any) => {
    console.log('上传数据:', values)
    
    const formData = new FormData()
    
    // 支持多文件上传 - 检查后端期望的字段名
    if (values.file && values.file.length > 0) {
      values.file.forEach((fileObj: any) => {
        console.log('添加文件:', fileObj.originFileObj.name)
        formData.append('files', fileObj.originFileObj)
      })
    }
    
    formData.append('program_id', values.program_id)
    formData.append('version', values.version)
    formData.append('description', values.description || '')

    console.log('FormData内容:')
    for (let [key, value] of formData.entries()) {
      console.log(key, value)
    }

    try {
      const response = await api.post('/files/upload', formData)
      
      const { isNewVersion } = response.data
      if (isNewVersion) {
        message.success('新版本文件上传成功')
      } else {
        message.success('文件重新上传成功')
      }
      
      setUploadModalVisible(false)
      loadData()
    } catch (error) {
      console.error('Failed to upload:', error)
      if (error.response?.data?.error) {
        message.error(`上传失败: ${error.response.data.error}`)
      } else {
        message.error('上传失败，请稍后重试')
      }
    }
  }

  const handleViewFiles = async (record: any) => {
    setCurrentProgram(record)
    setLoading(true)
    try {
      const response = await api.get(`/files/program/${record.id}`)
      setVersions(response.data.versions || [])
      setFileModalVisible(true)
    } catch (error) {
      console.error('Failed to load files:', error)
      message.error('加载文件列表失败')
    } finally {
      setLoading(false)
    }
  }

  const downloadWithAuth = async (url: string, fallbackName: string) => {
    const response = await api.get(url, { responseType: 'blob' })
    const blob = new Blob([response.data])

    const contentDisposition = response.headers['content-disposition']
    let filename = fallbackName
    if (contentDisposition) {
      const match = /filename\*=UTF-8''([^;]+)|filename="?([^";]+)"?/i.exec(contentDisposition)
      const encodedName = match?.[1] || match?.[2]
      if (encodedName) {
        filename = decodeURIComponent(encodedName)
      }
    }

    const urlObject = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = urlObject
    link.download = filename
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(urlObject)
  }

  const handleDownload = async (record: any) => {
    try {
      const response = await api.get(`/files/program/${record.id}`)
      if (response.data.versions.length === 0) {
        message.warning('该程序暂无上传的文件')
        return
      }
      
      // 获取最新版本的所有文件
      const latestVersion = response.data.versions[0]
      if (latestVersion.files.length > 0) {
        if (latestVersion.files.length === 1) {
          // 如果只有一个文件，直接下载
          const file = latestVersion.files[0]
          await downloadWithAuth(`/files/download/${file.id}`, file.file_name)
        } else {
          // 如果有多个文件，打包下载最新版本
          await downloadWithAuth(
            `/files/download/program/${record.id}/latest`,
            `${record.code || record.id}_${latestVersion.version}.zip`
          )
          message.success('正在打包下载最新版本的所有文件...')
        }
      } else {
        message.warning('该程序暂无可用文件')
      }
    } catch (error) {
      console.error('Failed to download:', error)
      message.error('下载失败')
    }
  }

  const renderVersionFiles = (version: any) => {
    if (!version.files || version.files.length === 0) {
      return <p style={{ color: '#999', textAlign: 'center' }}>此版本暂无文件</p>
    }

    return (
      <Table
        dataSource={version.files}
        rowKey="id"
        pagination={false}
        size="small"
        columns={[
          {
            title: '文件名',
            dataIndex: 'file_name',
            key: 'file_name',
            render: (text: string, record: any) => (
              <Space>
                <FileOutlined style={{ color: '#1890ff' }} />
                {text}
              </Space>
            ),
          },
          {
            title: '大小',
            dataIndex: 'file_size',
            key: 'file_size',
            width: 100,
            render: (size: number) => `${(size / 1024).toFixed(2)} KB`,
          },
          {
            title: '上传时间',
            dataIndex: 'created_at',
            key: 'created_at',
            width: 150,
            render: (time: string) => new Date(time).toLocaleString(),
          },
          {
            title: '上传者',
            dataIndex: ['uploader', 'name'],
            key: 'uploader',
            width: 100,
          },
          {
            title: '操作',
            key: 'action',
            width: 80,
            render: (_: any, record: any) => (
              <Button
                size="small"
                type="link"
                icon={<DownloadOutlined />}
                onClick={async () => {
                  try {
                    await downloadWithAuth(`/files/download/${record.id}`, record.file_name)
                  } catch (error) {
                    console.error('Failed to download file:', error)
                    message.error('下载失败')
                  }
                }}
              >
                下载
              </Button>
            ),
          },
        ]}
      />
    )
  }

  const columns = [
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
      title: '车型',
      dataIndex: ['vehicle_model', 'name'],
      key: 'vehicle_model',
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
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: any) => (
        <Space>
          <Button 
            icon={<EyeOutlined />} 
            size="small" 
            onClick={() => handleViewFiles(record)}
          >
            查看
          </Button>
          <Button 
            icon={<UploadOutlined />} 
            size="small" 
            onClick={() => handleUpload(record)}
          >
            上传
          </Button>
          {record.version && (
            <Button 
              icon={<UploadOutlined />} 
              size="small" 
              type="dashed"
              onClick={() => handleReupload(record)}
            >
              重传当前版本
            </Button>
          )}
          <Button 
            icon={<DownloadOutlined />} 
            size="small" 
            onClick={() => handleDownload(record)}
          >
            下载
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
        <Title level={2}>程序管理</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          新建程序
        </Button>
      </div>
      <Table
        columns={columns}
        dataSource={programs}
        rowKey="id"
        loading={loading}
      />

      <Modal
        title={currentProgram ? '编辑程序' : '新建程序'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="程序名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="code" label="程序编号" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="production_line_id" label="生产线" rules={[{ required: true }]}>
            <Select>
              {productionLines.map((line: any) => (
                <Select.Option key={line.id} value={line.id}>
                  {line.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="vehicle_model_id" label="车型">
            <Select allowClear>
              {vehicleModels.map((model: any) => (
                <Select.Option key={model.id} value={model.id}>
                  {model.name}
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

      <Modal
        title="上传程序文件"
        open={uploadModalVisible}
        onCancel={() => setUploadModalVisible(false)}
        onOk={() => uploadForm.submit()}
      >
        <Form form={uploadForm} layout="vertical" onFinish={handleUploadSubmit}>
          <Form.Item name="program_id" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="version" label="版本号" rules={[{ required: true }]} extra="如需重新上传当前版本，请输入相同版本号">
            <Input placeholder="例如: v1.0.0" />
          </Form.Item>
          <Form.Item
            name="file"
            label="选择文件"
            valuePropName="fileList"
            getValueFromEvent={(e) => (Array.isArray(e) ? e : e?.fileList)}
            rules={[{ required: true, message: '请选择文件' }]}
          >
            <Upload beforeUpload={() => false} multiple>
              <Button icon={<UploadOutlined />}>选择文件（可多选）</Button>
            </Upload>
          </Form.Item>
          <Form.Item name="description" label="变更说明">
            <TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 文件版本管理模态框 */}
      <Modal
        title={
          <Space>
            <HistoryOutlined />
            {`${currentProgram?.name} - 版本文件管理`}
          </Space>
        }
        open={fileModalVisible}
        onCancel={() => setFileModalVisible(false)}
        footer={null}
        width={1000}
        style={{ top: 20 }}
      >
        {loading ? (
          <div style={{ textAlign: 'center', padding: '40px' }}>加载中...</div>
        ) : versions.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <p>暂无版本信息</p>
            <Button 
              type="primary" 
              icon={<UploadOutlined />}
              onClick={() => {
                setFileModalVisible(false)
                handleUpload(currentProgram)
              }}
            >
              上传第一个版本
            </Button>
          </div>
        ) : (
          <Collapse 
            defaultActiveKey={versions.length > 0 ? [versions[0].version] : []}
            ghost
          >
            {versions.map((version: any) => (
              <Panel
                key={version.version}
                header={
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Space>
                      <Badge 
                        status={version.is_current ? 'processing' : 'default'} 
                        text={version.is_current ? '当前版本' : '历史版本'} 
                      />
                      <Tag color="blue">{version.version}</Tag>
                      <Badge count={version.file_count || 0} style={{ backgroundColor: '#52c41a' }}>
                        <FileOutlined />
                      </Badge>
                    </Space>
                    <Space>
                      <Tooltip title="版本说明">
                        {version.change_log && (
                          <span style={{ color: '#666', fontSize: '12px', maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {version.change_log}
                          </span>
                        )}
                      </Tooltip>
                      <span style={{ color: '#999', fontSize: '12px' }}>
                        {new Date(version.created_at).toLocaleDateString()}
                      </span>
                    </Space>
                  </div>
                }
                extra={
                  <Space>
                    <Button 
                      size="small" 
                      type="primary"
                      icon={<DownloadOutlined />}
                      onClick={async () => {
                        if (version.files && version.files.length > 0) {
                          try {
                            if (version.files.length === 1) {
                              // 单个文件直接下载
                              const file = version.files[0]
                              await downloadWithAuth(`/files/download/${file.id}`, file.file_name)
                            } else {
                              // 多个文件打包下载
                              await downloadWithAuth(
                                `/files/download/version/${version.version}?program_id=${currentProgram.id}`,
                                `${currentProgram.code || currentProgram.id}_${version.version}.zip`
                              )
                              message.success(`正在打包下载版本 ${version.version} 的所有文件...`)
                            }
                          } catch (error) {
                            console.error('Failed to download version files:', error)
                            message.error('下载失败')
                          }
                        } else {
                          message.warning('该版本暂无文件')
                        }
                      }}
                    >
                      批量下载
                    </Button>
                    {!version.is_current && (
                      <Button 
                        size="small" 
                        type="link"
                        onClick={() => {
                          Modal.confirm({
                            title: '确认激活版本',
                            content: `确定要激活版本 ${version.version} 吗？这将设为当前版本。`,
                            onOk: async () => {
                              try {
                                await api.put(`/versions/${version.id}/activate`)
                                message.success('版本激活成功')
                                handleViewFiles(currentProgram)
                              } catch (error) {
                                message.error('激活失败')
                              }
                            }
                          })
                        }}
                      >
                        激活
                      </Button>
                    )}
                  </Space>
                }
              >
                <div style={{ marginBottom: '16px' }}>
                  {version.change_log && (
                    <div style={{ 
                      background: '#f6f8fa', 
                      padding: '12px', 
                      borderRadius: '6px', 
                      marginBottom: '16px',
                      border: '1px solid #e1e4e8'
                    }}>
                      <div style={{ fontWeight: 'bold', marginBottom: '8px', color: '#586069' }}>
                        <ClockCircleOutlined style={{ marginRight: '8px' }} />
                        版本说明：
                      </div>
                      <div style={{ color: '#24292e', lineHeight: '1.5' }}>
                        {version.change_log}
                      </div>
                    </div>
                  )}
                  <div style={{ marginBottom: '8px' }}>
                    <Space>
                      <span>上传者：</span>
                      <Tag>{version.uploader?.name}</Tag>
                      <span>创建时间：</span>
                      <span>{new Date(version.created_at).toLocaleString()}</span>
                    </Space>
                  </div>
                  {renderVersionFiles(version)}
                </div>
              </Panel>
            ))}
          </Collapse>
        )}
      </Modal>
    </div>
  )
}

export default ProgramManagement