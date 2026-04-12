import { useEffect, useState } from 'react';
import {
  Card,
  Table,
  Select,
  Typography,
  Tag,
  Progress,
  Space,
  Button,
  Tooltip,
} from 'antd';
import {
  FileTextOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import api from '../services/api';

const { Title, Text } = Typography;

interface Program {
  id: number;
  name: string;
  code: string;
  production_line_id: number;
  vehicle_model_id: number;
  status: string;
  production_line?: { id: number; name: string };
  vehicle_model?: { id: number; name: string };
}

interface ProductionLine {
  id: number;
  name: string;
}

interface VehicleModel {
  id: number;
  name: string;
}

const ProgramMatrixPreview = () => {
  const [programs, setPrograms] = useState<Program[]>([]);
  const [productionLines, setProductionLines] = useState<ProductionLine[]>([]);
  const [vehicleModels, setVehicleModels] = useState<VehicleModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedLine, setSelectedLine] = useState<number | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [programsRes, linesRes, modelsRes] = await Promise.all([
        api.get('/programs'),
        api.get('/production-lines'),
        api.get('/vehicle-models'),
      ]);
      setPrograms(programsRes.data);
      setProductionLines(linesRes.data);
      setVehicleModels(modelsRes.data);
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  // 根据筛选条件获取生产线
  const filteredLines = selectedLine
    ? productionLines.filter((line) => line.id === selectedLine)
    : productionLines;

  // 获取车型在指定生产线的程序
  const getProgramForCell = (modelId: number, lineId: number) => {
    return programs.find(
      (p) => p.vehicle_model_id === modelId && p.production_line_id === lineId
    );
  };

  // 计算车型在指定生产线的完成率
  // const getCompletionRate = (modelId: number, lineId: number) => {
  //   const program = getProgramForCell(modelId, lineId);
  //   return program ? 100 : 0;
  // };

  // 计算车型总体完成率
  const getModelCompletionRate = (modelId: number) => {
    const totalLines = filteredLines.length;
    if (totalLines === 0) return 0;
    const completedLines = filteredLines.filter((line) =>
      getProgramForCell(modelId, line.id)
    ).length;
    return Math.round((completedLines / totalLines) * 100);
  };

  // 计算生产线总体完成率
  const getLineCompletionRate = (lineId: number) => {
    const totalModels = vehicleModels.length;
    if (totalModels === 0) return 0;
    const completedModels = vehicleModels.filter((model) =>
      getProgramForCell(model.id, lineId)
    ).length;
    return Math.round((completedModels / totalModels) * 100);
  };

  // 渲染单元格
  const renderCell = (modelId: number, lineId: number) => {
    const program = getProgramForCell(modelId, lineId);
    if (program) {
      return (
        <Tooltip title={`${program.name} (${program.code})`}>
          <Tag
            color={program.status === 'completed' ? 'blue' : 'purple'}
            style={{ cursor: 'pointer' }}
          >
            <CheckCircleOutlined /> 有程序
          </Tag>
        </Tooltip>
      );
    }
    return (
      <Tag color="default">
        <CloseCircleOutlined /> 无程序
      </Tag>
    );
  };

  // 构建表格列
  const columns: any[] = [
    {
      title: '车型',
      dataIndex: 'name',
      key: 'name',
      fixed: 'left' as const,
      width: 150,
      render: (text: string, record: VehicleModel) => (
        <Space>
          <Text strong>{text}</Text>
          <Tag color={getModelCompletionRate(record.id) === 100 ? 'green' : 'orange'}>
            {getModelCompletionRate(record.id)}%
          </Tag>
        </Space>
      ),
    },
    ...filteredLines.map((line) => ({
      title: (
        <div style={{ textAlign: 'center' }}>
          <div>{line.name}</div>
          <Progress
            percent={getLineCompletionRate(line.id)}
            size="small"
            showInfo={false}
            strokeColor={getLineCompletionRate(line.id) === 100 ? '#52c41a' : '#1890ff'}
          />
          <Text type="secondary" style={{ fontSize: '12px' }}>
            {getLineCompletionRate(line.id)}% 完成
          </Text>
        </div>
      ),
      dataIndex: ['vehicle_model', 'name'],
      key: `line_${line.id}`,
      width: 150,
      render: (_: any, record: VehicleModel) => (
        <div style={{ textAlign: 'center' }}>
          {renderCell(record.id, line.id)}
        </div>
      ),
    })),
  ];

  // 计算总体完成率
  const totalPrograms = vehicleModels.length * filteredLines.length;
  const completedPrograms = vehicleModels.reduce((sum, model) => {
    return (
      sum +
      filteredLines.filter((line) => getProgramForCell(model.id, line.id)).length
    );
  }, 0);
  const overallCompletionRate =
    totalPrograms > 0 ? Math.round((completedPrograms / totalPrograms) * 100) : 0;

  return (
    <div style={{ padding: '24px', maxWidth: '1600px', margin: '0 auto' }}>
      {/* 标题和筛选 */}
      <Card style={{ marginBottom: 16 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <Space>
            <FileTextOutlined style={{ fontSize: '24px', color: '#1890ff' }} />
            <Title level={3} style={{ margin: 0 }}>
              程序矩阵预览
            </Title>
          </Space>
          <Space>
            <Select
              style={{ width: 200 }}
              placeholder="按生产线筛选"
              allowClear
              value={selectedLine}
              onChange={setSelectedLine}
            >
              {productionLines.map((line) => (
                <Select.Option key={line.id} value={line.id}>
                  {line.name}
                </Select.Option>
              ))}
            </Select>
            <Button icon={<ReloadOutlined />} onClick={loadData}>
              刷新
            </Button>
          </Space>
        </div>
      </Card>

      {/* 统计卡片 */}
      <Card style={{ marginBottom: 16 }}>
        <Space size="large">
          <div>
            <Text type="secondary">车型数</Text>
            <Title level={4} style={{ margin: 0 }}>
              {vehicleModels.length}
            </Title>
          </div>
          <div>
            <Text type="secondary">生产线数</Text>
            <Title level={4} style={{ margin: 0 }}>
              {filteredLines.length}
            </Title>
          </div>
          <div>
            <Text type="secondary">程序数</Text>
            <Title level={4} style={{ margin: 0 }}>
              {completedPrograms}
            </Title>
          </div>
          <div>
            <Text type="secondary">总体完成率</Text>
            <Progress
              percent={overallCompletionRate}
              size="small"
              style={{ width: 100 }}
              strokeColor={overallCompletionRate === 100 ? '#52c41a' : '#1890ff'}
            />
          </div>
        </Space>
      </Card>

      {/* 矩阵表格 */}
      <div style={{ background: '#fff', borderRadius: '16px', boxShadow: '0px 12px 40px rgba(0, 91, 193, 0.03)', overflow: 'hidden' }}>
        <Table
          className="custom-table"
          columns={columns}
          dataSource={vehicleModels}
          rowKey="id"
          loading={loading}
          pagination={false}
          scroll={{ x: 'max-content' }}
          bordered
        />
      </div>

      {/* 图例 */}
      <Card style={{ marginTop: 16 }}>
        <Space>
          <Text type="secondary">图例：</Text>
          <Tag color="green">
            <CheckCircleOutlined /> 有程序
          </Tag>
          <Tag color="red">
            <CheckCircleOutlined /> 有程序（停用）
          </Tag>
          <Tag color="default">
            <CloseCircleOutlined /> 无程序
          </Tag>
        </Space>
      </Card>
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

export default ProgramMatrixPreview;
