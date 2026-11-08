export default {
  baizhi: {
    title: '百智云调用',
    description: '聚合百智云 4 大开放服务（网页抓取、网络搜索、智能语音、图像生成），多账号 API Key 负载轮询与统一网关中转。',
    tabs: {
      config: '服务与密钥配置',
      docs: '网关统一接入文档'
    },
    overview: {
      gatewayBaseUrl: '统一网关 Base URL',
      gatewayBaseUrlHint: '向业务客户端或 AI 智能体分发的统一代理入口地址，调用时自动轮询已配置账号的 API Key。',
      authRequired: '网关 API Key 鉴权',
      authRequiredHint: '开启后，客户端调用网关必须在 Header 携带有效的 Sub2API 令牌 (Authorization: Bearer sk-...)。',
      saveConfig: '保存配置',
      saveSuccess: '百智云网关配置已更新并生效',
      saveFailed: '保存配置失败',
      activeServices: '活跃服务',
      totalKeys: '已配置密钥',
      copySuccess: '网关地址已复制到剪贴板'
    },
    services: {
      scrape: '网页抓取与结构化提取',
      scrapeDesc: '打开网页返回正文、指定字段结构化提取、字段建议、批量提取与整理成稿',
      search: '网络综合搜索',
      searchDesc: '通用网页搜索、AI 搜索(SSE流式推送)、新闻搜索、文搜图、图搜图',
      voice: '智能语音服务 (ASR / TTS)',
      voiceDesc: '实时语音识别与合成(WebSocket)、录音文件转文字、文字转语音(WAV)、音色库',
      image: '图像生成与编辑',
      imageDesc: '文生图、参考图生图、上传预签名凭证、异步任务查询与智能重试'
    },
    card: {
      status: '服务状态',
      enabled: '启用',
      disabled: '停用',
      strategy: '调度策略',
      strategyRoundRobin: '轮询 (Round-Robin)',
      strategyFailover: '主备 (Failover)',
      upstreamBaseUrl: '上游 Base URL',
      wssUrl: 'WebSocket URL',
      activeKeysCount: '生效密钥',
      addKey: '添加账号密钥',
      noKeys: '暂无配置账号 Key，请点击上方按钮添加',
      testConnection: '连通性测试',
      testing: '测试中...',
      testSuccess: '连通正常',
      testFailed: '连接失败',
      accountName: '账号备注',
      apiKey: 'API Key',
      actions: '操作',
      deleteConfirm: '确定要删除此账号密钥吗？',
      gatewayPath: '网关路由前缀'
    },
    dialog: {
      addKeyTitle: '添加账号 API Key',
      accountName: '账号名称 / 备注',
      accountNamePlaceholder: '例如：主账号-个人 或 团队号-A',
      apiKey: '百智云 API Key',
      apiKeyPlaceholder: '粘贴对应百智云业务的 API Key',
      cancel: '取消',
      confirm: '确认添加'
    },
    docs: {
      title: '百智云统一网关接入指南',
      subtitle: '通过统一网关无缝接入 4 大业务能力，自动分摊多账号额度并隐藏真实凭据。',
      authGuideTitle: '身份鉴权',
      authGuideText: '在向网关发起 HTTP 或 WebSocket 请求时，携带 Sub2API 颁发的 API Key：',
      quickNav: '业务快速直达',
      endpoints: '接口列表',
      method: '请求方式',
      endpoint: '接口路径',
      desc: '说明',
      example: '调用示例',
      copyCode: '复制代码',
      copied: '已复制'
    }
  }
}
