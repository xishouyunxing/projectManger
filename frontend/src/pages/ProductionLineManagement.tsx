import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
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
  Popconfirm,
  DatePicker,
  ConfigProvider,
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined } from '@ant-design/icons';
import api from '../services/api';
import ProductionLineCustomFieldManager from '../components/production-line/ProductionLineCustomFieldManager';

const { Title } = Typography;
const { TextArea } = Input;

const ProductionLineManagement = () => {
  const [searchParams] = useSearchParams();
  const selectedLineId = Number(searchParams.get('id') || 0);
  const [productionLines, setProductionLines] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [customFieldManagerVisible, setCustomFieldManagerVisible] = useState(false);
  const [currentLine, setCurrentLine] = useState<any>(null);
  const [form] = Form.useForm();

  // 筛选相关状态
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterType, setFilterType] = useState<string | null>(null);
  const [filterStatus, setFilterStatus] = useState<string | null>(null);
  const [filterDateRange, setFilterDateRange] = useState<[string | null, string | null]>([null, null]);

  useEffect(() => {
    loadData();
  }, []);

  useEffect(() => {
    const keyword = searchParams.get('keyword');
    if (keyword) {
      setSearchKeyword(keyword);
    }
  }, [searchParams]);

  const loadData = async () => {
    setLoading(true);
    try {
      const linesRes = await api.get('/production-lines');
      setProductionLines(linesRes.data);
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = () => {
    setCurrentLine(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: any) => {
    setCurrentLine(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleManageCustomFields = (record: any) => {
    setCurrentLine(record);
    setCustomFieldManagerVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/production-lines/${id}`);
      message.success('删除成功');
      loadData();
    } catch (error) {
      console.error('Failed to delete:', error);
    }
  };

  const handleSubmit = async (values: any) => {
    try {
      if (currentLine) {
        await api.put(`/production-lines/${currentLine.id}`, values);
        message.success('更新成功');
      } else {
        await api.post('/production-lines', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      loadData();
    } catch (error) {
      console.error('Failed to submit:', error);
    }
  };

  // 重置筛选
  const handleResetFilter = () => {
    setSearchKeyword('');
    setFilterType(null);
    setFilterStatus(null);
    setFilterDateRange([null, null]);
  };

  // 筛选后的生产线列表
  const filteredProductionLines = productionLines.filter((line: any) => {
    // 关键词搜索（模糊匹配名称、编号、描述）
    if (searchKeyword) {
      const keyword = searchKeyword.toLowerCase();
      const nameMatch = line.name?.toLowerCase().includes(keyword);
      const codeMatch = line.code?.toLowerCase().includes(keyword);
      const descMatch = line.description?.toLowerCase().includes(keyword);
      if (!nameMatch && !codeMatch && !descMatch) {
        return false;
      }
    }
    if (filterType && line.type !== filterType) {
      return false;
    }
    if (filterStatus && line.status !== filterStatus) {
      return false;
    }
    // 时间筛选
    if (filterDateRange[0] || filterDateRange[1]) {
      const lineDate = new Date(line.created_at);
      if (filterDateRange[0]) {
        const startDate = new Date(filterDateRange[0]);
        startDate.setHours(0, 0, 0, 0);
        if (lineDate < startDate) return false;
      }
      if (filterDateRange[1]) {
        const endDate = new Date(filterDateRange[1]);
        endDate.setHours(23, 59, 59, 999);
        if (lineDate > endDate) return false;
      }
    }
    return true;
  }).sort((a: any, b: any) => {
    if (selectedLineId) {
      if (a.id === selectedLineId) return -1;
      if (b.id === selectedLineId) return 1;
    }
    return 0;
  });

  const columns = [
    {
      title: '生产线名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <span style={{ color: '#2D3335', fontSize: '14px', fontWeight: 700, fontFamily: 'Inter, sans-serif' }}>
          {text}
        </span>
      ),
    },
    {
      title: '编号',
      dataIndex: 'code',
      key: 'code',
      render: (text: string) => (
        <span style={{ color: '#5A6062', fontSize: '14px', fontWeight: 500, fontFamily: 'Inter, sans-serif' }}>
          {text}
        </span>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => (
        <div style={{ background: '#EBEEF0', borderRadius: '4px', display: 'inline-block', padding: '2px 8px' }}>
          <span style={{ color: '#2D3335', fontSize: '12px', fontWeight: 700, letterSpacing: '0.6px' }}>
            {type === 'upper' ? '上车' : type === 'lower' ? '下车' : type}
          </span>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const isActive = status === 'active';
        return (
          <div style={{ 
            background: isActive ? 'rgba(61, 137, 255, 0.20)' : 'rgba(222, 204, 253, 0.40)', 
            borderRadius: '9999px', 
            display: 'inline-flex', 
            alignItems: 'center',
            padding: '2px 10px',
            gap: '6px'
          }}>
            <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: isActive ? '#005BC1' : '#50426B' }}></div>
            <span style={{ color: isActive ? '#005BC1' : '#50426B', fontSize: '11px', fontWeight: 700 }}>
              {isActive ? '运行中' : '停止'}
            </span>
          </div>
        );
      },
    },
    {
      title: '最后更新',
      dataIndex: 'updated_at',
      key: 'updated_at',
      render: (text: string, record: any) => {
        const dateStr = text || record.created_at || new Date().toISOString();
        const dateObj = new Date(dateStr);
        const dateFmt = dateObj.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' }).replace(/\//g, '-');
        const timeFmt = dateObj.toLocaleTimeString('zh-CN', { hour12: false });
        return (
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 500, fontFamily: 'Inter, sans-serif' }}>{dateFmt}</span>
            <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 400, fontFamily: 'Inter, sans-serif', opacity: 0.6 }}>{timeFmt}</span>
          </div>
        );
      },
    },
    {
      title: '操作',
      key: 'action',
      align: 'right' as const,
      render: (_: any, record: any) => (
        <Space size="small">
          <Button
            type="text"
            icon={<EditOutlined style={{ color: '#5A6062' }} />}
            onClick={() => handleEdit(record)}
            style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
          />
          <Button
            type="text"
            onClick={() => handleManageCustomFields(record)}
            style={{
              minWidth: '72px',
              height: '32px',
              borderRadius: '4px',
              background: '#F8F9FA',
              color: '#5A6062',
              fontWeight: 600,
            }}
          >
            管理字段
          </Button>
          <Popconfirm
            title="确定删除?"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button 
              type="text" 
              icon={<DeleteOutlined style={{ color: '#A83836' }} />} 
              style={{ width: '32px', height: '32px', borderRadius: '4px', background: 'rgba(168, 56, 54, 0.05)' }} 
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="management-page">
      {/* 顶部标题区 */}
      <div className="management-page-header">
        <div>
          <div className="management-page-breadcrumb">
            <span>制造</span>
            <span style={{ margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>/</span>
            <span className="active">生产线</span>
          </div>
          <Title level={2} className="management-page-title">
            生产线管理
          </Title>
        </div>
        <Button 
          type="primary" 
          icon={<PlusOutlined />} 
          onClick={handleAdd}
          style={{
            background: 'linear-gradient(176deg, #005BC1 0%, #3D89FF 100%)',
            border: 'none',
            boxShadow: '0px 4px 6px -4px rgba(0, 91, 193, 0.10), 0px 10px 15px -3px rgba(0, 91, 193, 0.10)',
            borderRadius: '8px',
            height: '44px',
            padding: '0 24px',
            fontWeight: 600,
            fontSize: '16px'
          }}
        >
          新建生产线
        </Button>
      </div>

      {/* 搜索/筛选区域 */}
      <ConfigProvider
        theme={{
          components: {
            Input: {
              controlHeight: 36,
              borderRadius: 8,
              colorBorder: 'transparent',
              colorPrimaryHover: 'transparent',
              controlOutline: 'none',
            },
            Select: {
              controlHeight: 36,
              borderRadius: 8,
              colorBorder: 'transparent',
              colorPrimaryHover: 'transparent',
              controlOutline: 'none',
            },
            DatePicker: {
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
          <div className="management-filter-field flex">
            <div className="management-filter-label">生产线名称/编号</div>
            <Input 
              placeholder="搜索参数..." 
              value={searchKeyword} 
              onChange={(e) => setSearchKeyword(e.target.value)}
            />
          </div>
          <div className="management-filter-field">
            <div className="management-filter-label">类型</div>
            <Select 
              placeholder="所有类型" 
              value={filterType} 
              onChange={setFilterType}
              allowClear
              style={{ width: '100%' }}
            >
              <Select.Option value="upper">上车</Select.Option>
              <Select.Option value="lower">下车</Select.Option>
            </Select>
          </div>
          <div className="management-filter-field">
            <div className="management-filter-label">状态</div>
            <Select 
              placeholder="所有状态" 
              value={filterStatus} 
              onChange={setFilterStatus}
              allowClear
              style={{ width: '100%' }}
            >
              <Select.Option value="active">活跃</Select.Option>
              <Select.Option value="inactive">停用</Select.Option>
            </Select>
          </div>
          <div className="management-filter-field">
            <div className="management-filter-label">创建日期</div>
            <DatePicker.RangePicker 
              style={{ width: '100%' }}
              onChange={(_, dateStrings) => {
                setFilterDateRange([dateStrings[0] || null, dateStrings[1] || null]);
              }}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <Button 
              icon={<SearchOutlined />}
              style={{ height: '40px', width: '115px', borderRadius: '8px', background: '#DEE3E6', color: '#2D3335', fontWeight: 700, border: 'none' }}
            >
              查询
            </Button>
            <Button 
              type="text"
              onClick={handleResetFilter} 
              style={{ height: '40px', color: '#005BC1', fontWeight: 700, letterSpacing: '1.2px', border: 'none', background: 'transparent' }}
            >
              重置
            </Button>
          </div>
        </div>
      </ConfigProvider>

      {/* 数据表格 */}
      <div className="management-table-card">
        <Table
          columns={columns}
          dataSource={filteredProductionLines}
          rowKey="id"
          loading={loading}
          pagination={{
            showTotal: (total, range) => `显示第 ${range[0]} 至 ${range[1]} 条，共 ${total} 条记录`,
            style: { padding: '16px 24px', margin: 0, background: 'rgba(241, 244, 245, 0.50)' }
          }}
          className="custom-table"
        />
      </div>

      <Modal
        title={currentLine ? '编辑生产线' : '新建生产线'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            name="name"
            label="生产线名称"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="code"
            label="生产线编号"
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="type" label="类型" rules={[{ required: true }]}>
            <Select>
              <Select.Option value="upper">上车</Select.Option>
              <Select.Option value="lower">下车</Select.Option>
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

      <ProductionLineCustomFieldManager
        open={customFieldManagerVisible}
        productionLine={currentLine}
        onClose={() => setCustomFieldManagerVisible(false)}
      />
      
    </div>
  );
};

export default ProductionLineManagement;
