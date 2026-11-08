export default {
  baizhi: {
    title: 'Baizhi Cloud API',
    description: 'Unified gateway relay and multi-account API key pooling for Baizhi Cloud Web Scrape, Web Search, Voice, and Image Generation services.',
    tabs: {
      config: 'Service & Key Configuration',
      docs: 'Unified Gateway Documentation'
    },
    overview: {
      gatewayBaseUrl: 'Unified Gateway Base URL',
      gatewayBaseUrlHint: 'Base URL distributed to client apps and AI agents. Requests automatically round-robin across configured account keys.',
      authRequired: 'Gateway API Key Auth',
      authRequiredHint: 'When enabled, requests to the gateway must provide a valid Sub2API Bearer token (Authorization: Bearer sk-...).',
      saveConfig: 'Save Configuration',
      saveSuccess: 'Baizhi gateway configuration saved successfully',
      saveFailed: 'Failed to save configuration',
      activeServices: 'Active Services',
      totalKeys: 'Configured Keys',
      copySuccess: 'Gateway URL copied to clipboard'
    },
    services: {
      scrape: 'Web Scrape & Extraction',
      scrapeDesc: 'Scrape full pages, structured field extraction, field suggestions, batch extraction, and document synthesis.',
      search: 'Web Search Suite',
      searchDesc: 'General web search, AI search (SSE streaming), news search, text-to-image search, and image-to-image search.',
      voice: 'Intelligent Voice (ASR / TTS)',
      voiceDesc: 'Realtime speech-to-text & text-to-speech (WebSocket), audio file transcription, WAV speech synthesis, and voice catalog.',
      image: 'Image Generation & Editing',
      imageDesc: 'Text-to-image, reference image editing, presigned upload tickets, task polling, and automatic retry.'
    },
    card: {
      status: 'Service Status',
      enabled: 'Enabled',
      disabled: 'Disabled',
      strategy: 'Dispatch Strategy',
      strategyRoundRobin: 'Round-Robin',
      strategyFailover: 'Failover',
      upstreamBaseUrl: 'Upstream Base URL',
      wssUrl: 'WebSocket URL',
      activeKeysCount: 'Active Keys',
      addKey: 'Add Account Key',
      noKeys: 'No account keys configured yet. Click above to add one.',
      testConnection: 'Test Connection',
      testing: 'Testing...',
      testSuccess: 'Connected',
      testFailed: 'Connection Failed',
      accountName: 'Account Remark',
      apiKey: 'API Key',
      actions: 'Actions',
      deleteConfirm: 'Are you sure you want to delete this account key?',
      gatewayPath: 'Gateway Route Prefix'
    },
    dialog: {
      addKeyTitle: 'Add Account API Key',
      accountName: 'Account Name / Remark',
      accountNamePlaceholder: 'e.g., Primary-Account or Team-A',
      apiKey: 'Baizhi API Key',
      apiKeyPlaceholder: 'Paste the API key provided by Baizhi Cloud',
      cancel: 'Cancel',
      confirm: 'Confirm'
    },
    docs: {
      title: 'Baizhi Unified Gateway Integration Guide',
      subtitle: 'Seamlessly access all 4 service capabilities through a single unified endpoint with automated key pooling.',
      authGuideTitle: 'Authentication',
      authGuideText: 'Include your Sub2API token when making HTTP or WebSocket requests to the gateway:',
      quickNav: 'Quick Navigation',
      endpoints: 'Endpoints',
      method: 'Method',
      endpoint: 'Endpoint',
      desc: 'Description',
      example: 'Example',
      copyCode: 'Copy Code',
      copied: 'Copied'
    }
  }
}
