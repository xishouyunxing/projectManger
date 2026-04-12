import { useEffect, useState } from 'react';
import {
  Card,
  Row,
  Col,
  Typography,
  Space,
  List,
  Tag,
  Progress,
  Table,
  Button,
  Select,
  message,
} from 'antd';
import {
  FileTextOutlined,
  CarOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  TeamOutlined,
  DashboardOutlined,
  SafetyOutlined,
  FileExcelOutlined,
  AppstoreOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import api from '../services/api';
import { useAuth } from '../contexts/AuthContext';

const { Title, Text } = Typography;

// 动画样式
const styles = `
  @keyframes fadeInUp {
    from {
      opacity: 0;
      transform: translateY(15px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .stat-card {
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    background: white;
    border-radius: 16px;
    box-shadow: 0px 12px 40px rgba(0, 0, 0, 0.02);
    height: 100%;
    border: none;
    overflow: hidden;
  }
  
  .stat-card:hover {
    transform: translateY(-4px);
    box-shadow: 0px 12px 40px rgba(0, 91, 193, 0.06);
  }

  .welcome-banner {
    background: transparent;
    border-radius: 16px;
    margin-bottom: 32px;
    animation: fadeInUp 0.4s ease both;
  }

  .quick-action-item {
    background: #F1F4F5;
    border-radius: 12px;
    padding: 16px;
    text-align: center;
    cursor: pointer;
    transition: all 0.2s ease;
    border: 1px solid transparent;
  }

  .quick-action-item:hover {
    background: white;
    border-color: rgba(61, 137, 255, 0.2);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
    transform: translateY(-2px);
  }

  .quick-action-item:active {
    transform: translateY(0) scale(0.98);
  }
`;

// 注入样式
if (typeof document !== 'undefined') {
  const styleSheet = document.createElement('style');
  styleSheet.textContent = styles;
  document.head.appendChild(styleSheet);
}

const Dashboard = () => {
  const navigate = useNavigate();
  const { isAdmin } = useAuth();
  const [stats, setStats] = useState({
    programs: 0,
    users: 0,
    productionLines: 0,
    vehicleModels: 0,
  });
  const [loading, setLoading] = useState(true);
  const [recentActivities, setRecentActivities] = useState<any[]>([]);

  // 程序预览相关状态
  const [programs, setPrograms] = useState<any[]>([]);
  const [productionLines, setProductionLines] = useState<any[]>([]);
  const [vehicleModels, setVehicleModels] = useState<any[]>([]);
  const [previewFilter, setPreviewFilter] = useState<number | null>(null);

  // 页面切换状态
  const [currentPage, setCurrentPage] = useState<'dashboard' | 'preview'>('dashboard');

  useEffect(() => {
    loadStats();
    loadRecentActivities();
    loadProgramsPreview();
  }, []);

  const loadStats = async () => {
    try {
      const [programs, users, lines, models] = await Promise.all([
        api.get('/programs'),
        api.get('/users'),
        api.get('/production-lines'),
        api.get('/vehicle-models'),
      ]);

      setStats({
        programs: programs.data.length,
        users: users.data.length,
        productionLines: lines.data.length,
        vehicleModels: models.data.length,
      });
      setProductionLines(lines.data);
      setVehicleModels(models.data);
    } catch (error) {
      console.error('Failed to load stats:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadProgramsPreview = async () => {
    try {
      const response = await api.get('/programs');
      setPrograms(response.data);
    } catch (error) {
      console.error('Failed to load programs:', error);
    }
  };

  const loadRecentActivities = async () => {
    try {
      const response = await api.get('/files/ignore/logs');
      const logs = response.data || [];

      if (logs.length > 0) {
        setRecentActivities(
          logs.slice(0, 5).map((log: any) => ({
            title: log.action || '文件操作',
            time: new Date(log.timestamp).toLocaleString('zh-CN'),
            status: log.status === 'success' ? 'success' : 'warning',
            icon:
              log.status === 'success' ? (
                <CheckCircleOutlined style={{ color: '#52c41a' }} />
              ) : (
                <ExclamationCircleOutlined style={{ color: '#faad14' }} />
              ),
            type: log.status === 'success' ? 'success' : 'warning'
          })),
        );
      }
    } catch (error) {
      console.error('Failed to load recent activities:', error);
    }
  };

  const quickActions = [
    {
      title: '新建程序',
      icon: <FileTextOutlined />,
      color: '#005BC1',
      path: '/programs',
    },
    {
      title: '用户管理',
      icon: <TeamOutlined />,
      color: '#52c41a',
      path: '/users',
    },
    {
      title: '生产线管理',
      icon: <DashboardOutlined />,
      color: '#faad14',
      path: '/production-lines',
    },
    {
      title: '车型管理',
      icon: <CarOutlined />,
      color: '#722ed1',
      path: '/vehicle-models',
    },
    {
      title: '数据预览',
      icon: <FileExcelOutlined />,
      color: '#13c2c2',
      onClick: () => setCurrentPage('preview'),
    },
    ...(isAdmin
      ? [
          {
            title: '系统管理',
            icon: <AppstoreOutlined />,
            color: '#ff4d4f',
            path: '/system-management',
          },
        ]
      : []),
  ];

  // 矩阵预览相关函数
  const getProgramForCell = (modelId: number, lineId: number) => {
    return programs.find(
      (p: any) => p.vehicle_model_id === modelId && p.production_line_id === lineId
    );
  };

  const getModelCompletionRate = (modelId: number) => {
    const totalLines = filteredLines.length;
    if (totalLines === 0) return 0;
    const completedLines = filteredLines.filter((line: any) =>
      getProgramForCell(modelId, line.id)
    ).length;
    return Math.round((completedLines / totalLines) * 100);
  };

  const getLineCompletionRate = (lineId: number) => {
    const totalModels = vehicleModels.length;
    if (totalModels === 0) return 0;
    const completedModels = vehicleModels.filter((model: any) =>
      getProgramForCell(model.id, lineId)
    ).length;
    return Math.round((completedModels / totalModels) * 100);
  };

  const filteredLines = previewFilter
    ? productionLines.filter((line: any) => line.id === previewFilter)
    : productionLines;

  const renderCell = (modelId: number, lineId: number) => {
    const program = getProgramForCell(modelId, lineId);
    if (program) {
      return (
        <Tag color={program.status === 'completed' ? 'blue' : 'purple'}>
          有程序
        </Tag>
      );
    }
    return <Tag color="default">无程序</Tag>;
  };

  const renderPreviewPage = () => {
    const totalPrograms = vehicleModels.length * filteredLines.length;
    const completedPrograms = vehicleModels.reduce((sum: number, model: any) => {
      return (
        sum +
        filteredLines.filter((line: any) => getProgramForCell(model.id, line.id)).length
      );
    }, 0);
    const overallCompletionRate =
      totalPrograms > 0 ? Math.round((completedPrograms / totalPrograms) * 100) : 0;

    const handleExportExcel = async () => {
      try {
        const params: any = {};
        if (previewFilter) params.production_line_id = previewFilter;

        const response = await api.get('/programs/export/excel', {
          params,
          responseType: 'blob',
        });

        const url = window.URL.createObjectURL(new Blob([response.data]));
        const link = document.createElement('a');
        link.href = url;
        
        const contentDisposition = response.headers['content-disposition'];
        let fileName = '程序列表.xlsx';
        if (contentDisposition) {
          const match = contentDisposition.match(/filename=(.+)/);
          if (match) fileName = match[1];
        }
        
        link.setAttribute('download', fileName);
        document.body.appendChild(link);
        link.click();
        link.remove();
        window.URL.revokeObjectURL(url);
        
        message.success('导出成功');
      } catch (error) {
        console.error('Failed to export:', error);
        message.error('导出失败');
      }
    };

    const tableColumns = [
      {
        title: '生产线',
        dataIndex: ['production_line', 'name'],
        key: 'production_line',
        render: (text: string) => text || '-',
      },
      {
        title: '程序名称',
        dataIndex: 'name',
        key: 'name',
      },
      {
        title: '车型',
        dataIndex: ['vehicle_model', 'name'],
        key: 'vehicle_model',
        render: (text: string) => text || '-',
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        render: (status: string) => {
          const isCompleted = status === 'completed';
          return (
            <div style={{ 
              background: isCompleted ? 'rgba(61, 137, 255, 0.20)' : 'rgba(222, 204, 253, 0.40)', 
              borderRadius: '9999px', 
              display: 'inline-flex', 
              alignItems: 'center',
              padding: '2px 10px',
              gap: '6px'
            }}>
              <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: isCompleted ? '#005BC1' : '#50426B' }}></div>
              <span style={{ color: isCompleted ? '#005BC1' : '#50426B', fontSize: '11px', fontWeight: 700 }}>
                {isCompleted ? '已完成' : '进行中'}
              </span>
            </div>
          );
        },
      },
    ];

    const filteredProgramsForTable = previewFilter
      ? programs.filter((p: any) => p.production_line_id === previewFilter)
      : programs;

    return (
      <div style={{ padding: '24px', maxWidth: '1600px', margin: '0 auto' }}>
        <Card style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Space>
              <FileExcelOutlined style={{ fontSize: '24px', color: '#1890ff' }} />
              <Title level={3} style={{ margin: 0 }}>数据预览</Title>
            </Space>
            <Space>
              <Select
                style={{ width: 200 }}
                placeholder="按生产线筛选"
                allowClear
                value={previewFilter}
                onChange={setPreviewFilter}
              >
                {productionLines.map((line: any) => (
                  <Select.Option key={line.id} value={line.id}>{line.name}</Select.Option>
                ))}
              </Select>
              <Button 
                type="primary" 
                icon={<FileExcelOutlined />} 
                onClick={handleExportExcel}
                style={{
                  background: 'linear-gradient(176deg, #005BC1 0%, #3D89FF 100%)',
                  border: 'none',
                  borderRadius: '8px',
                  fontWeight: 600,
                }}
              >
                导出Excel
              </Button>
              <Button 
                onClick={() => setCurrentPage('dashboard')}
                style={{ borderRadius: '8px', fontWeight: 600 }}
              >
                返回仪表盘
              </Button>
            </Space>
          </div>
        </Card>

        <Card style={{ marginBottom: 16 }}>
          <Space size="large">
            <div><Text type="secondary">车型数</Text><Title level={4} style={{ margin: 0 }}>{vehicleModels.length}</Title></div>
            <div><Text type="secondary">生产线数</Text><Title level={4} style={{ margin: 0 }}>{filteredLines.length}</Title></div>
            <div><Text type="secondary">程序数</Text><Title level={4} style={{ margin: 0 }}>{completedPrograms}</Title></div>
            <div>
              <Text type="secondary">总体完成率</Text>
              <Progress percent={overallCompletionRate} size="small" style={{ width: 100 }} strokeColor={overallCompletionRate === 100 ? '#52c41a' : '#1890ff'} />
            </div>
          </Space>
        </Card>

        <Card title="程序矩阵" style={{ marginBottom: 16 }}>
          <Table
            dataSource={vehicleModels}
            rowKey="id"
            pagination={false}
            scroll={{ x: 'max-content' }}
            bordered
            size="small"
            columns={[
              {
                title: '车型',
                dataIndex: 'name',
                key: 'name',
                fixed: 'left' as const,
                width: 150,
                render: (text: string, record: any) => (
                  <Space>
                    <Text strong>{text}</Text>
                    <Tag color={getModelCompletionRate(record.id) === 100 ? 'green' : 'orange'}>
                      {getModelCompletionRate(record.id)}%
                    </Tag>
                  </Space>
                ),
              },
              ...filteredLines.map((line: any) => ({
                title: (
                  <div style={{ textAlign: 'center' }}>
                    <div>{line.name}</div>
                    <Progress percent={getLineCompletionRate(line.id)} size="small" showInfo={false} strokeColor={getLineCompletionRate(line.id) === 100 ? '#52c41a' : '#1890ff'} />
                    <Text type="secondary" style={{ fontSize: '12px' }}>{getLineCompletionRate(line.id)}% 完成</Text>
                  </div>
                ),
                key: `line_${line.id}`,
                width: 150,
                render: (_: any, record: any) => (
                  <div style={{ textAlign: 'center' }}>{renderCell(record.id, line.id)}</div>
                ),
              })),
            ]}
          />
        </Card>

        <div style={{ background: '#fff', borderRadius: '16px', boxShadow: '0px 12px 40px rgba(0, 91, 193, 0.03)', overflow: 'hidden' }}>
          <div style={{ padding: '16px 24px', fontSize: '16px', fontWeight: 600, borderBottom: '1px solid #f0f0f0' }}>数据列表（与Excel导出格式一致）</div>
          <Table
            className="custom-table"
            dataSource={filteredProgramsForTable}
            rowKey="id"
            columns={tableColumns}
            pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 条数据` }}
          />
        </div>
      </div>
    );
  };

  if (currentPage === 'preview') {
    return renderPreviewPage();
  }

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '100px' }}>
        <Title level={3}>加载中...</Title>
      </div>
    );
  }

  const currentDate = new Date().toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  return (
    <div style={{ padding: '32px', maxWidth: '1600px', margin: '0 auto', background: '#F8F9FA' }}>
      {/* 欢迎区域 */}
      <div className="welcome-banner">
        <Text type="secondary" style={{ fontSize: '16px', fontWeight: 500, color: '#5A6062' }}>
          {currentDate}
        </Text>
        <Title level={1} style={{ margin: '8px 0 16px', fontSize: '36px', fontWeight: 800, color: '#2D3335' }}>
          欢迎来到控制台！
        </Title>
      </div>

      {/* 顶部统计卡片 */}
      <Row gutter={[24, 24]} style={{ marginBottom: '32px' }}>
        <Col xs={24} sm={12} lg={6}>
          <div className="stat-card" style={{ padding: '24px', animation: `fadeInUp 0.4s both 0.1s` }}>
            <Space direction="vertical" style={{ width: '100%', height: '100%', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Text type="secondary" style={{ fontSize: '14px', fontWeight: 500, color: '#5A6062' }}>程序总数</Text>
                <FileTextOutlined style={{ color: '#005BC1', fontSize: '16px' }} />
              </div>
              <div style={{ marginTop: '24px' }}>
                <Title level={2} style={{ margin: 0, fontSize: '32px', fontWeight: 700, fontFamily: 'Manrope, sans-serif' }}>
                  {stats.programs}
                </Title>
              </div>
            </Space>
          </div>
        </Col>
        
        <Col xs={24} sm={12} lg={6}>
          <div className="stat-card" style={{ padding: '24px', animation: `fadeInUp 0.4s both 0.15s` }}>
            <Space direction="vertical" style={{ width: '100%', height: '100%', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Text type="secondary" style={{ fontSize: '14px', fontWeight: 500, color: '#5A6062' }}>用户总数</Text>
                <TeamOutlined style={{ color: '#52c41a', fontSize: '16px' }} />
              </div>
              <div style={{ marginTop: '24px' }}>
                <Title level={2} style={{ margin: 0, fontSize: '32px', fontWeight: 700, fontFamily: 'Manrope, sans-serif' }}>
                  {stats.users}
                </Title>
              </div>
            </Space>
          </div>
        </Col>
        
        <Col xs={24} sm={12} lg={6}>
          <div className="stat-card" style={{ padding: '24px', animation: `fadeInUp 0.4s both 0.2s` }}>
            <Space direction="vertical" style={{ width: '100%', height: '100%', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Text type="secondary" style={{ fontSize: '14px', fontWeight: 500, color: '#5A6062' }}>生产线数</Text>
                <DashboardOutlined style={{ color: '#faad14', fontSize: '16px' }} />
              </div>
              <div style={{ marginTop: '24px' }}>
                <Title level={2} style={{ margin: 0, fontSize: '32px', fontWeight: 700, fontFamily: 'Manrope, sans-serif' }}>
                  {stats.productionLines}
                </Title>
              </div>
            </Space>
          </div>
        </Col>
        
        <Col xs={24} sm={12} lg={6}>
          <div className="stat-card" style={{ padding: '24px', animation: `fadeInUp 0.4s both 0.25s` }}>
            <Space direction="vertical" style={{ width: '100%', height: '100%', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Text type="secondary" style={{ fontSize: '14px', fontWeight: 500, color: '#5A6062' }}>车型数</Text>
                <CarOutlined style={{ color: '#722ed1', fontSize: '16px' }} />
              </div>
              <div style={{ marginTop: '24px' }}>
                <Title level={2} style={{ margin: 0, fontSize: '32px', fontWeight: 700, fontFamily: 'Manrope, sans-serif' }}>
                  {stats.vehicleModels}
                </Title>
              </div>
            </Space>
          </div>
        </Col>
      </Row>

      {/* 底部面板 */}
      <Row gutter={[24, 24]}>
        <Col xs={24} lg={12}>
          <Space direction="vertical" size={24} style={{ width: '100%' }}>
            {/* 快捷操作 */}
            <div className="stat-card" style={{ padding: '24px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                 <Title level={4} style={{ margin: 0, color: '#2D3335' }}>快捷操作</Title>
              </div>
              <Row gutter={[12, 12]}>
                {quickActions.map((item, idx) => (
                   <Col span={8} key={idx}>
                     <div className="quick-action-item" onClick={() => item.onClick ? item.onClick() : navigate(item.path || '#')}>
                       <div style={{ color: item.color, fontSize: '24px', marginBottom: '8px' }}>{item.icon}</div>
                       <div style={{ fontSize: '12px', color: '#2D3335', fontWeight: 500 }}>{item.title}</div>
                     </div>
                   </Col>
                ))}
              </Row>
            </div>
          </Space>
        </Col>

        <Col xs={24} lg={12}>
          <Space direction="vertical" size={24} style={{ width: '100%' }}>
            {/* 最近活动 */}
            <div className="stat-card" style={{ padding: '24px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
                <Title level={4} style={{ margin: 0, color: '#2D3335' }}>最近活动</Title>
              </div>
              
              <List
                itemLayout="horizontal"
                dataSource={recentActivities}
                renderItem={(item) => (
                  <List.Item style={{ padding: '16px 0', borderBottom: '1px solid #EBEEF0' }}>
                    <List.Item.Meta
                      avatar={
                        <div style={{ width: '48px', height: '48px', borderRadius: '12px', background: item.type === 'success' ? '#f6ffed' : item.type === 'warning' ? '#fffbe6' : '#e6f4ff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                          {item.type === 'success' ? <CheckCircleOutlined style={{ color: '#52c41a', fontSize: '20px' }} /> : item.type === 'warning' ? <ExclamationCircleOutlined style={{ color: '#faad14', fontSize: '20px' }} /> : <SafetyOutlined style={{ color: '#1890ff', fontSize: '20px' }} />}
                        </div>
                      }
                      title={<span style={{ fontSize: '16px', fontWeight: 600, color: '#2D3335' }}>{item.title}</span>}
                      description={<span style={{ fontSize: '14px', color: '#5A6062' }}>{item.desc || item.time}</span>}
                    />
                    <div style={{ textAlign: 'right' }}>
                      <Tag color={item.type === 'success' ? 'success' : item.type === 'warning' ? 'warning' : 'processing'} style={{ borderRadius: '4px', padding: '2px 8px', border: 'none' }}>
                        {item.status}
                      </Tag>
                      <div style={{ color: '#94A3B8', fontSize: '12px', marginTop: '8px' }}>{item.time}</div>
                    </div>
                  </List.Item>
                )}
              />
            </div>
          </Space>
        </Col>
      </Row>
      {/* 注入表格自定义样式 */}
      <style>{`
        .custom-table .ant-table-thead > tr > th {
          background: #EBEEF0 !important;
          color: #5A6062 !important;
          font-size: 10px !important;
          font-weight: 700 !important;
          letter-spacing: 1px !important;
          border-bottom: 1px solid #DEE3E6 !important;
          padding: 16px 24px !important;
        }
        .custom-table .ant-table-tbody > tr > td {
          padding: 16px 24px !important;
          border-bottom: 1px solid #EBEEF0 !important;
        }
        .custom-table .ant-table {
          border-radius: 16px 16px 0 0 !important;
        }
        .custom-table .ant-pagination-total-text {
          color: #5A6062;
          font-size: 12px;
          font-weight: 500;
        }
      `}</style>
    </div>
  );
};

export default Dashboard;
