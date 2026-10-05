export default {
  agentTokens: {
    title: 'Agent 运维凭证',
    description: '为自动化运维与日常巡检的 AI Agent 签发受控独立令牌。Token 数据库加盐哈希存储，严格按最小权限隔离。',
    createToken: '签发 Agent Token',
    skillDoc: 'Agent 技能规约 (SKILL.md)',
    empty: '暂无已签发的 Agent 令牌',
    searchPlaceholder: '搜索 Token 名称 / 前缀',
    includeRevoked: '包含已吊销',
    activeTokensCount: '有效令牌',
    totalTokensCount: '全部令牌',
    securityNoticeTitle: '最高安全准则',
    securityNoticeText: 'Agent Token 与管理员登录凭证物理隔离。严禁明文外泄。支持细粒度 Scopes 权限约束及单次调用审计留痕。',
    columns: {
      name: '令牌名称',
      prefix: '前缀标识',
      scopes: '授予权限 (Scopes)',
      createdAt: '创建时间',
      expiresAt: '过期时间',
      lastUsedAt: '最近活跃',
      status: '状态',
      actions: '操作'
    },
    status: {
      active: '生效中',
      expired: '已过期',
      revoked: '已吊销'
    },
    createDialog: {
      title: '签发新的 Agent 令牌',
      name: '令牌名称 / 标识',
      namePlaceholder: '例如：Daily-Ops-Bot 或 Claude-Inspector',
      description: '用途说明',
      descriptionPlaceholder: '可选：简述此 Agent 的任务目标与运行环境',
      expiresIn: '有效期限',
      expiresOptions: {
        days30: '30 天',
        days90: '90 天',
        days180: '180 天',
        days365: '1 年',
        never: '永久有效'
      },
      scopes: '权限范围 (Scopes)',
      scopesHint: '遵循最小特权原则，仅勾选该 Agent 执行任务所需的必要权限。',
      selectAll: '全选',
      selectReadonly: '仅读感知',
      clearAll: '清空',
      submit: '立即签发',
      cancel: '取消'
    },
    createdDialog: {
      title: '令牌签发成功',
      warning: '请立即复制并保存此令牌！出于安全防线设计，系统仅展示一次明文令牌，关闭后将无法再次找回。',
      tokenLabel: 'Agent Token 明文',
      copySuccess: '令牌已复制到剪贴板',
      close: '我已妥善保存'
    },
    revokeConfirm: {
      title: '吊销 Agent 令牌',
      message: '确定要吊销令牌 “{name}” 吗？吊销后该 Agent 将立即失去所有接口访问权限，且无法撤销。',
      confirm: '确认吊销',
      cancel: '取消',
      success: '令牌已成功吊销'
    },
    skillDocDialog: {
      title: 'Agent 技能规约与接入指引',
      subtitle: '请将以下规范配置给您的外部 AI Agent（如 Claude, AutoGen, Dify 等），配合签发的 Token 进行自动化运维。',
      copyDoc: '复制完整规约 (Markdown)',
      copySuccess: '规约文档已成功复制',
      close: '关闭',
      loading: '正在加载规范文档...'
    }
  }
}
