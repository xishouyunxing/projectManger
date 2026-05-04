import { Card, Progress, Typography, Space, Row, Col } from 'antd';
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  DashboardOutlined,
  CarOutlined,
} from '@ant-design/icons';

const { Text, Title } = Typography;

interface StatsData {
  totalPrograms: number;
  completedPrograms: number;
  inProgressPrograms: number;
  totalLines: number;
  totalModels: number;
  overallRate: number;
  lineRates: { name: string; rate: number }[];
  modelRates: { name: string; rate: number }[];
}

interface Props {
  stats: StatsData;
  loading?: boolean;
}

const ExportStatsCards = ({ stats, loading }: Props) => {
  if (loading) {
    return (
      <Card loading style={{ marginBottom: 16 }}>
        <div style={{ height: 100 }} />
      </Card>
    );
  }

  return (
    <div style={{ marginBottom: 16 }}>
      {/* 概览统计 */}
      <Row gutter={[12, 12]} style={{ marginBottom: 12 }}>
        <Col span={6}>
          <Card size="small" bordered={false} style={{ background: '#f6ffed' }}>
            <Space direction="vertical" size={0} style={{ width: '100%' }}>
              <Text type="secondary" style={{ fontSize: 11 }}>
                程序总数
              </Text>
              <Title level={4} style={{ margin: 0 }}>
                {stats.totalPrograms}
              </Title>
            </Space>
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small" bordered={false} style={{ background: '#e6f4ff' }}>
            <Space direction="vertical" size={0} style={{ width: '100%' }}>
              <Space size={4}>
                <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 11 }} />
                <Text type="secondary" style={{ fontSize: 11 }}>
                  已完成
                </Text>
              </Space>
              <Title level={4} style={{ margin: 0, color: '#52c41a' }}>
                {stats.completedPrograms}
              </Title>
            </Space>
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small" bordered={false} style={{ background: '#fff7e6' }}>
            <Space direction="vertical" size={0} style={{ width: '100%' }}>
              <Space size={4}>
                <ClockCircleOutlined style={{ color: '#faad14', fontSize: 11 }} />
                <Text type="secondary" style={{ fontSize: 11 }}>
                  进行中
                </Text>
              </Space>
              <Title level={4} style={{ margin: 0, color: '#faad14' }}>
                {stats.inProgressPrograms}
              </Title>
            </Space>
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small" bordered={false}>
            <Space direction="vertical" size={4} style={{ width: '100%' }}>
              <Text type="secondary" style={{ fontSize: 11 }}>
                总体完成率
              </Text>
              <Progress
                percent={stats.overallRate}
                size="small"
                strokeColor={stats.overallRate === 100 ? '#52c41a' : '#1890ff'}
              />
            </Space>
          </Card>
        </Col>
      </Row>

      {/* 按产线完成率 */}
      {stats.lineRates.length > 0 && (
        <Card
          size="small"
          bordered={false}
          style={{ marginBottom: 12 }}
          title={
            <Space>
              <DashboardOutlined style={{ color: '#1890ff' }} />
              <span style={{ fontSize: 13 }}>产线完成率</span>
              <Text type="secondary" style={{ fontSize: 11 }}>
                {stats.totalLines} 条产线
              </Text>
            </Space>
          }
        >
          <Row gutter={[8, 8]}>
            {stats.lineRates.map((item) => (
              <Col span={8} key={item.name}>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                  }}
                >
                  <Text
                    ellipsis
                    style={{ fontSize: 12, maxWidth: 100 }}
                    title={item.name}
                  >
                    {item.name}
                  </Text>
                  <Progress
                    percent={item.rate}
                    size="small"
                    style={{ width: 80, margin: 0 }}
                    strokeColor={item.rate === 100 ? '#52c41a' : '#1890ff'}
                  />
                </div>
              </Col>
            ))}
          </Row>
        </Card>
      )}

      {/* 按车型完成率 */}
      {stats.modelRates.length > 0 && (
        <Card
          size="small"
          bordered={false}
          title={
            <Space>
              <CarOutlined style={{ color: '#722ed1' }} />
              <span style={{ fontSize: 13 }}>车型完成率</span>
              <Text type="secondary" style={{ fontSize: 11 }}>
                {stats.totalModels} 个车型
              </Text>
            </Space>
          }
        >
          <Row gutter={[8, 8]}>
            {stats.modelRates.map((item) => (
              <Col span={8} key={item.name}>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                  }}
                >
                  <Text
                    ellipsis
                    style={{ fontSize: 12, maxWidth: 100 }}
                    title={item.name}
                  >
                    {item.name}
                  </Text>
                  <Progress
                    percent={item.rate}
                    size="small"
                    style={{ width: 80, margin: 0 }}
                    strokeColor={item.rate === 100 ? '#52c41a' : '#1890ff'}
                  />
                </div>
              </Col>
            ))}
          </Row>
        </Card>
      )}
    </div>
  );
};

export default ExportStatsCards;
