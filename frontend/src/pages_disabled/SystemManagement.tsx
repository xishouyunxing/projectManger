import { useEffect, useState } from 'react'
import {
  Card,
  Button,
  Table,
  Space,
  Modal,
  message,
  Typography,
  Tag,
  Progress,
  Divider,
  Popconfirm,
  Upload,
  Alert,
} from 'antd'
import {
  DatabaseOutlined,
  FileOutlined,
  CloudDownloadOutlined,
  DeleteOutlined,
  ReloadOutlined,
  UploadOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons'
import api from '../services/api'
import { useAuth } from '../contexts/AuthContext'

const { Title, Text } = Typography

interface BackupInfo {
  name: string
  path: string
  size: number
  created_at: string
  type: 'database' | 'files' | 'full'
}

const SystemManagement = () => {
  const [backups, setBackups] = useState<BackupInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [operationLoading, setOperationLoading] = useState('')
  const [confirmModalVisible, setConfirmModalVisible] = useState(false)
  const [currentOperation, setCurrentOperation] = useState<any>(null)

  const { isAdmin } = useAuth()

  useEffect(() => {
    loadBackups()
  }, [])

  const loadBackups = async () => {
    setLoading(true)
    try {
      const response = await api.get('/backup')
      setBackups(response.data.backups || [])
    } catch (error) {
      console.error('Failed to load backups:', error)
      message.error('加载备份列表失败')
    } finally {
      setLoading(false)
    }
  }

  const createBackup = async (type: string) => {
    setOperationLoading(type)
    try {
      const endpoint = type === 'database' ? '/backup/database' : 
                      type === 'files' ? '/backup/files' : 
                      '/backup/full'
      
      const response = await api.post(endpoint)
      message.success(response.data.message)
      
      // 刷新备份列表
      await loadBackups()
    } catch (error: any) {
      console.error(`Failed to create ${type} backup:`, error)
      message.error(error.response?.data?.error || `创建${type}备份失败`)
    } finally {
      setOperationLoading('')
    }
  }

  const deleteBackup = async (backupName: string) => {
    try {
      await api.delete(`/backup/${backupName}`)
      message.success('备份删除成功')
      await loadBackups()
    } catch (error: any) {
      console.error('Failed to delete backup:', error)
      message.error(error.response?.data?.error || '删除备份失败')
    }
  }

  const downloadBackup = async (backupName: string) => {
    try {
      const response = await api.get(`/backup/download/${backupName}`, {
        responseType: 'blob'
      })
      
      // 创建下载链接
      const url = window.URL.createObjectURL(new Blob([response.data]))
      const link = document.createElement('a')
      link.href = url
      link.setAttribute('download', backupName)
      document.body.appendChild(link)
      link.click()
      link.remove()
      window.URL.revokeObjectURL(url)
    } catch (error: any) {
      console.error('Failed to download backup:', error)
      message.error('下载备份失败')
    }
  }

  const confirmRestore = (backupName: string, type: 'database' | 'files') => {
    setCurrentOperation({ backupName, type })
    setConfirmModalVisible(true)
  }

  const executeRestore = async () => {
    if (!currentOperation) return
    
    setOperationLoading(`restore-${currentOperation.type}`)
    try {
      const endpoint = currentOperation.type === 'database' 
        ? `/backup/restore/database/${currentOperation.backupName}`
        : `/backup/restore/files/${currentOperation.backupName}`
      
      const response = await api.post(endpoint)
      
      Modal.success({
        title: '恢复成功',
        content: (
          <div>
            <p>{response.data.message}</p>
            {response.data.rollback_backup && (
              <p>回滚备份已创建：{response.data.rollback_backup}</p>
            )}
          </div>
        ),
      })
      
      setConfirmModalVisible(false)
      await loadBackups()
    } catch (error: any) {
      console.error('Failed to restore:', error)
      message.error(error.response?.data?.error || '恢复失败')
    } finally {
      setOperationLoading('')
      setCurrentOperation(null)
    }
  }

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('zh-CN')
  }

  const getBackupTypeTag = (type: string) => {
    const config = {
      database: { color: 'blue', text: '数据库' },
      files: { color: 'green', text: '文件系统' },
      full: { color: 'purple', text: '完整备份' },
    }
    const { color, text } = config[type as keyof typeof config] || { color: 'default', text: '未知' }
    return <Tag color={color}>{text}</Tag>
  }

  const columns = [
    {
      title: '备份名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <Text strong>{text}</Text>,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => getBackupTypeTag(type),
    },
    {
      title: '文件大小',
      dataIndex: 'size',
      key: 'size',
      render: (size: number) => formatFileSize(size),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => formatDate(date),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_: any, record: BackupInfo) => (
        <Space>
          <Button
            type="link"
            icon={<CloudDownloadOutlined />}
            onClick={() => downloadBackup(record.name)}
          >
            下载
          </Button>
          <Popconfirm
            title="确定要删除这个备份吗？"
            onConfirm={() => deleteBackup(record.name)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
          {record.type === 'database' || record.type === 'full' ? (
            <Button
              type="link"
              icon={<ReloadOutlined />}
              onClick={() => confirmRestore(record.name, 'database')}
              loading={operationLoading === 'restore-database'}
            >
              恢复数据库
            </Button>
          ) : null}
          {record.type === 'files' || record.type === 'full' ? (
            <Button
              type="link"
              icon={<ReloadOutlined />}
              onClick={() => confirmRestore(record.name, 'files')}
              loading={operationLoading === 'restore-files'}
            >
              恢复文件
            </Button>
          ) : null}
        </Space>
      ),
    },
  ]

  if (!isAdmin) {
    return (
      <div style={{ padding: '24px' }}>
        <Alert
          message="权限不足"
          description="只有系统管理员才能访问系统管理功能。"
          type="error"
          showIcon
        />
      </div>
    )
  }

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>系统管理</Title>
      
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* 备份操作区域 */}
        <Card title="备份操作" extra={
          <Button onClick={loadBackups} loading={loading}>
            刷新列表
          </Button>
        }>
          <Space size="large" wrap>
            <Button
              type="primary"
              icon={<DatabaseOutlined />}
              onClick={() => createBackup('database')}
              loading={operationLoading === 'database'}
            >
              备份数据库
            </Button>
            <Button
              type="primary"
              icon={<FileOutlined />}
              onClick={() => createBackup('files')}
              loading={operationLoading === 'files'}
            >
              备份文件系统
            </Button>
            <Button
              type="primary"
              danger
              icon={<CloudDownloadOutlined />}
              onClick={() => createBackup('full')}
              loading={operationLoading === 'full'}
            >
              创建完整备份
            </Button>
          </Space>
          
          <Divider />
          
          <Alert
            message="备份说明"
            description={
              <ul>
                <li>数据库备份：导出SQLite数据库文件</li>
                <li>文件系统备份：压缩uploads目录下的所有文件</li>
                <li>完整备份：包含数据库和文件系统的完整备份</li>
                <li>恢复操作会自动创建回滚点，请谨慎操作</li>
              </ul>
            }
            type="info"
            showIcon
          />
        </Card>

        {/* 备份列表 */}
        <Card title="备份列表" extra={
          <Text type="secondary">
            共 {backups.length} 个备份文件
          </Text>
        }>
          <Table
            columns={columns}
            dataSource={backups}
            rowKey="name"
            loading={loading}
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total) => `共 ${total} 条记录`,
            }}
          />
        </Card>
      </Space>

      {/* 恢复确认对话框 */}
      <Modal
        title={
          <span>
            <ExclamationCircleOutlined style={{ color: '#faad14', marginRight: 8 }} />
            确认恢复操作
          </span>
        }
        open={confirmModalVisible}
        onOk={executeRestore}
        onCancel={() => {
          setConfirmModalVisible(false)
          setCurrentOperation(null)
        }}
        confirmLoading={operationLoading?.startsWith('restore-')}
        okText="确认恢复"
        cancelText="取消"
      >
        <div>
          <Alert
            message="警告"
            description={
              currentOperation && (
                <div>
                  <p>您即将恢复备份文件：{currentOperation.backupName}</p>
                  <p>恢复类型：{currentOperation.type === 'database' ? '数据库' : '文件系统'}</p>
                  <p>此操作会覆盖当前数据，系统会自动创建回滚备份。</p>
                  <p style={{ color: '#ff4d4f', fontWeight: 'bold' }}>
                    请确认您已保存当前工作，并了解此操作的风险！
                  </p>
                </div>
              )
            }
            type="warning"
            showIcon
          />
        </div>
      </Modal>
    </div>
  )
}

export default SystemManagement