import { useEffect, useState } from 'react'
import { Card, Row, Col, Statistic, Typography, Space, Button, List, Avatar, Tag } from 'antd'
import {
  UserOutlined,
  FileTextOutlined,
  SettingOutlined,
  CarOutlined,
  PlusOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons'
import api from '../services/api'

const { Title, Text } = Typography

const Dashboard = () => {
  const [stats, setStats] = useState({
    programs: 0,
    users: 0,
    productionLines: 0,
    vehicleModels: 0,
  })
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadStats()
  }, [])

  const loadStats = async () => {
    try {
      const [programs, users, lines, models] = await Promise.all([
        api.get('/programs'),
        api.get('/users'),
        api.get('/production-lines'),
        api.get('/vehicle-models'),
      ])

      setStats({
        programs: programs.data.length,
        users: users.data.length,
        productionLines: lines.data.length,
        vehicleModels: models.data.length,
      })
    } catch (error) {
      console.error('Failed to load stats:', error)
    } finally {
      setLoading(false)
    }
  }

  const recentActivities = [
    {
      title: '系统运行正常',
      time: '刚刚',
      status: 'success',
      icon: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
    },
    {
      title: '新用户注册',
      time: '5分钟前',
      status: 'info',
      icon: <UserOutlined style={{ color: '#1890ff' }} />,
    },
    {
      title: '生产线维护',
      time: '1小时前',
      status: 'warning',
      icon: <ExclamationCircleOutlined style={{ color: '#faad14' }} />,
    },
  ]

  const quickActions = [
    {
      title: '新建程序',
      icon: <FileTextOutlined />,
      color: '#1890ff',
    },
    {
      title: '用户管理',
      icon: <UserOutlined />,
      color: '#52c41a',
    },
    {
      title: '生产线',
      icon: <SettingOutlined />,
      color: '#faad14',
    },
    {
      title: '系统设置',
      icon: <SettingOutlined />,
      color: '#722ed1',
    },
  ]

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '100px' }}>
        <Title level={3}>加载中...</Title>
      </div>
    )
  }

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>系统概览</Title>
      <Text type="secondary">实时监控生产线管理系统运行状态</Text>
      
      <Row gutter={[16, 16]} style={{ marginTop: '24px' }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="程序总数"
              value={stats.programs}
              prefix={<FileTextOutlined style={{ color: '#1890ff' }} />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="用户总数"
              value={stats.users}
              prefix={<UserOutlined style={{ color: '#52c41a' }} />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="生产线数"
              value={stats.productionLines}
              prefix={<SettingOutlined style={{ color: '#faad14' }} />}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="车型数"
              value={stats.vehicleModels}
              prefix={<CarOutlined style={{ color: '#722ed1' }} />}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: '24px' }}>
        <Col xs={24} lg={12}>
          <Card title="快速操作" extra={<PlusOutlined />}>
            <Row gutter={[8, 8]}>
              {quickActions.map((action, index) => (
                <Col span={12} key={index}>
                  <Button
                    type="default"
                    block
                    size="large"
                    icon={action.icon}
                    style={{
                      height: '80px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexDirection: 'column',
                    }}
                  >
                    <div style={{ fontSize: '24px', marginBottom: '8px', color: action.color }}>
                      {action.icon}
                    </div>
                    <span>{action.title}</span>
                  </Button>
                </Col>
              ))}
            </Row>
          </Card>
        </Col>
        
        <Col xs={24} lg={12}>
          <Card title="最近活动" extra={<ClockCircleOutlined />}>
            <List
              itemLayout="horizontal"
              dataSource={recentActivities}
              renderItem={(item) => (
                <List.Item>
                  <List.Item.Meta
                    avatar={<Avatar icon={item.icon} />}
                    title={
                      <Space>
                        {item.title}
                        <Tag color={item.status}>{item.status}</Tag>
                      </Space>
                    }
                    description={item.time}
                  />
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard