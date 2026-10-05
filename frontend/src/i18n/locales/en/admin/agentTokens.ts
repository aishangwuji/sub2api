export default {
  agentTokens: {
    title: 'Agent Tokens',
    description: 'Issue fine-grained API credentials for AI agents to automate monitoring and ops. Tokens are salted & hashed in database.',
    createToken: 'Issue Agent Token',
    skillDoc: 'Agent Skill Spec (SKILL.md)',
    empty: 'No agent tokens found',
    searchPlaceholder: 'Search token name / prefix',
    includeRevoked: 'Include Revoked',
    activeTokensCount: 'Active Tokens',
    totalTokensCount: 'Total Tokens',
    securityNoticeTitle: 'Security Baseline',
    securityNoticeText: 'Agent tokens are physically isolated from admin logins. Never leak plain tokens. Fine-grained scopes and audit trails are enforced.',
    columns: {
      name: 'Token Name',
      prefix: 'Prefix',
      scopes: 'Scopes',
      createdAt: 'Created At',
      expiresAt: 'Expires At',
      lastUsedAt: 'Last Used',
      status: 'Status',
      actions: 'Actions'
    },
    status: {
      active: 'Active',
      expired: 'Expired',
      revoked: 'Revoked'
    },
    createDialog: {
      title: 'Issue New Agent Token',
      name: 'Token Name / Identity',
      namePlaceholder: 'e.g. Daily-Ops-Bot or Claude-Inspector',
      description: 'Description',
      descriptionPlaceholder: 'Optional: Brief task goals and deployment environment',
      expiresIn: 'Expiration',
      expiresOptions: {
        days30: '30 Days',
        days90: '90 Days',
        days180: '180 Days',
        days365: '1 Year',
        never: 'Never Expires'
      },
      scopes: 'Scopes',
      scopesHint: 'Follow principle of least privilege, select only necessary scopes.',
      selectAll: 'Select All',
      selectReadonly: 'Read-only',
      clearAll: 'Clear All',
      submit: 'Generate Token',
      cancel: 'Cancel'
    },
    createdDialog: {
      title: 'Agent Token Generated',
      warning: 'Copy and save this token now! For security reasons, the plaintext token is shown only once and cannot be retrieved later.',
      tokenLabel: 'Agent Token (Plaintext)',
      copySuccess: 'Token copied to clipboard',
      close: 'I Have Saved It'
    },
    revokeConfirm: {
      title: 'Revoke Agent Token',
      message: 'Are you sure you want to revoke "{name}"? The agent will immediately lose access and this action cannot be undone.',
      confirm: 'Revoke Token',
      cancel: 'Cancel',
      success: 'Agent token revoked successfully'
    },
    skillDocDialog: {
      title: 'Agent Skill Specification',
      subtitle: 'Provide this specification to your external AI Agent (e.g. Claude, AutoGen, Dify) along with the issued token.',
      copyDoc: 'Copy Spec (Markdown)',
      copySuccess: 'Skill document copied to clipboard',
      close: 'Close',
      loading: 'Loading skill specification...'
    }
  }
}
