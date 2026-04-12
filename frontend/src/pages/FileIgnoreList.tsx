import { useEffect, useState } from 'react';
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
  Tag,
  Tooltip,
  Row,
  Col,
  Switch,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  HistoryOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import api from '../services/api';

const { Title } = Typography;
const { TextArea } = Input;

interface FileIgnoreRule {
  id: number;
  name: string;
  type: string;
  pattern: string;
  description: string;
  enabled: boolean;
  scope: string;
  program_id?: number;
  program?: any;
  creator: any;
  created_at: string;
}

interface IgnoreLog {
  id: number;
  ignore_rule_id: number;
  file_name: string;
  file_size: number;
  user: any;
  ignore_rule: FileIgnoreRule;
  program: any;
  created_at: string;
}

const FileIgnoreList = () => {
  const [rules, setRules] = useState<FileIgnoreRule[]>([]);
  const [programs, setPrograms] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [logsModalVisible, setLogsModalVisible] = useState(false);
  const [currentRule, setCurrentRule] = useState<FileIgnoreRule | null>(null);
  const [logs, setLogs] = useState<IgnoreLog[]>([]);
  const [form] = Form.useForm();

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [rulesRes, programsRes] = await Promise.all([
        api.get('/files/ignore'),
        api.get('/programs'),
      ]);
      setRules(rulesRes.data);
      setPrograms(programsRes.data);
    } catch (error) {
      console.error('Failed to load data:', error);
      message.error('加载数据失败');
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = () => {
    setCurrentRule(null);
    form.resetFields();
    form.setFieldsValue({
      type: 'extension',
      scope: 'global',
      enabled: true,
    });
    setModalVisible(true);
  };

  const handleEdit = (record: FileIgnoreRule) => {
    setCurrentRule(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/files/ignore/${id}`);
      message.success('删除成功');
      loadData();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleSubmit = async (values: any) => {
    try {
      if (currentRule) {
        await api.put(`/files/ignore/${currentRule.id}`, values);
        message.success('更新成功');
      } else {
        await api.post('/files/ignore', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      loadData();
    } catch (error) {
      console.error('Failed to submit:', error);
      message.error('操作失败');
    }
  };

  const handleViewLogs = async (rule?: FileIgnoreRule) => {
    setLoading(true);
    try {
      const params: any = {};
      if (rule) {
        params.start_time = new Date(
          Date.now() - 30 * 24 * 60 * 60 * 1000,
        ).toISOString();
      }

      const response = await api.get('/files/ignore/logs', { params });
      setLogs(response.data.logs || []);
      setLogsModalVisible(true);
    } catch (error) {
      message.error('加载日志失败');
    } finally {
      setLoading(false);
    }
  };

  const getTypeColor = (type: string) => {
    const colors = {
      extension: 'blue',
      filename: 'green',
      pattern: 'purple',
    };
    return colors[type as keyof typeof colors] || 'default';
  };

  const getScopeColor = (scope: string) => {
    const colors = {
      global: 'red',
      program: 'orange',
    };
    return colors[scope as keyof typeof colors] || 'default';
  };

  const columns = [
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: FileIgnoreRule) => (
        <Space>
          {text}
          {!record.enabled && <Tag color="default">已禁用</Tag>}
        </Space>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={getTypeColor(type)}>
          {type === 'extension'
            ? '扩展名'
            : type === 'filename'
              ? '文件名'
              : '模式'}
        </Tag>
      ),
    },
    {
      title: '忽略模式',
      dataIndex: 'pattern',
      key: 'pattern',
      render: (pattern: string) => (
        <code
          style={{
            background: '#f5f5f5',
            padding: '2px 6px',
            borderRadius: '3px',
          }}
        >
          {pattern}
        </code>
      ),
    },
    {
      title: '作用范围',
      dataIndex: 'scope',
      key: 'scope',
      width: 120,
      render: (scope: string, record: FileIgnoreRule) => (
        <Space>
          <Tag color={getScopeColor(scope)}>
            {scope === 'global' ? '全局' : '程序'}
          </Tag>
          {record.program && (
            <Tooltip title={record.program.name}>
              <span>{record.program.code}</span>
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '创建者',
      dataIndex: ['creator', 'name'],
      key: 'creator',
      width: 100,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (time: string) => {
        const dateObj = new Date(time);
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
      width: 200,
      render: (_: any, record: FileIgnoreRule) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
            style={{ padding: 0 }}
          >
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            icon={<HistoryOutlined />}
            onClick={() => handleViewLogs(record)}
            style={{ padding: 0 }}
          >
            日志
          </Button>
          <Popconfirm
            title="确定删除?"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="link" danger icon={<DeleteOutlined />} size="small" style={{ padding: 0 }}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const logColumns = [
    {
      title: '文件名',
      dataIndex: 'file_name',
      key: 'file_name',
      render: (text: string) => <code>{text}</code>,
    },
    {
      title: '忽略规则',
      dataIndex: ['ignore_rule', 'name'],
      key: 'ignore_rule',
      render: (_: string, record: IgnoreLog) => (
        <Space>
          <Tag color={getTypeColor(record.ignore_rule.type)}>
            {record.ignore_rule.pattern}
          </Tag>
        </Space>
      ),
    },
    {
      title: '文件大小',
      dataIndex: 'file_size',
      key: 'file_size',
      width: 100,
      render: (size: number) => `${(size / 1024).toFixed(2)} KB`,
    },
    {
      title: '用户',
      dataIndex: ['user', 'name'],
      key: 'user',
      width: 100,
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (time: string) => {
        const dateObj = new Date(time);
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
  ];

  return (
    <div style={{ padding: '0 8px', maxWidth: '1024px', margin: '0 auto', fontFamily: '"WenQuanYi Zen Hei", Inter, Manrope, sans-serif' }}>
      {/* 顶部标题区 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginTop: '24px' }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: '8px' }}>
            <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, letterSpacing: '1.2px' }}>系统</span>
            <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>/</span>
            <span style={{ color: '#005BC1', fontSize: '12px', fontWeight: 700, letterSpacing: '1.2px' }}>文件忽略列表</span>
          </div>
          <Title level={2} style={{ margin: 0, color: '#2D3335', fontSize: '30px', fontWeight: 800 }}>
            文件忽略列表
          </Title>
        </div>
        <Space>
          <Button 
            icon={<HistoryOutlined />} 
            onClick={() => handleViewLogs()}
            style={{ height: '44px', borderRadius: '8px', fontWeight: 600, padding: '0 16px' }}
          >
            查看所有日志
          </Button>
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
            新建忽略规则
          </Button>
        </Space>
      </div>

      <div style={{ marginTop: '32px', background: '#fff', borderRadius: '16px', boxShadow: '0px 12px 40px rgba(0, 91, 193, 0.03)', overflow: 'hidden' }}>
        <Table
          columns={columns}
          dataSource={rules}
          rowKey="id"
          loading={loading}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
            style: { padding: '16px 24px', margin: 0, background: 'rgba(241, 244, 245, 0.50)' }
          }}
          className="custom-table"
        />
      </div>

      <Modal
        title={currentRule ? '编辑忽略规则' : '新建忽略规则'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
        width={600}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="规则名称" rules={[{ required: true }]}>
            <Input placeholder="例如: 日志文件忽略规则" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="type"
                label="规则类型"
                rules={[{ required: true }]}
              >
                <Select>
                  <Select.Option value="extension">扩展名</Select.Option>
                  <Select.Option value="filename">文件名</Select.Option>
                  <Select.Option value="pattern">正则表达式</Select.Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="scope"
                label="作用范围"
                rules={[{ required: true }]}
              >
                <Select>
                  <Select.Option value="global">全局</Select.Option>
                  <Select.Option value="program">程序级别</Select.Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="pattern"
            label="忽略模式"
            rules={[{ required: true }]}
          >
            <Input
              placeholder="例如: *.log 或 temp_* 或 .DS_Store"
              suffix={
                <Tooltip title="扩展名以点开头，如 .log；模式使用通配符或正则表达式">
                  <InfoCircleOutlined style={{ color: '#999' }} />
                </Tooltip>
              }
            />
          </Form.Item>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) =>
              prevValues.scope !== currentValues.scope
            }
          >
            {({ getFieldValue }) =>
              getFieldValue('scope') === 'program' ? (
                <Form.Item
                  name="program_id"
                  label="关联程序"
                  rules={[{ required: true }]}
                >
                  <Select placeholder="选择程序">
                    {programs.map((program: any) => (
                      <Select.Option key={program.id} value={program.id}>
                        {program.code} - {program.name}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              ) : null
            }
          </Form.Item>

          <Form.Item name="description" label="描述">
            <TextArea rows={3} placeholder="描述这个忽略规则的用途" />
          </Form.Item>

          <Form.Item name="enabled" label="状态" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="忽略日志"
        open={logsModalVisible}
        onCancel={() => setLogsModalVisible(false)}
        footer={null}
        width={800}
      >
        <Table
          columns={logColumns}
          dataSource={logs}
          rowKey="id"
          loading={loading}
          size="small"
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条记录`,
          }}
        />
      </Modal>
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

export default FileIgnoreList;
