<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header & Action Bar -->
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
            {{ t('admin.baizhi.title') }}
          </h1>
        </div>

        <div class="flex flex-wrap items-center gap-2.5">
          <button
            type="button"
            class="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-medium bg-white dark:bg-dark-800 text-gray-800 dark:text-gray-100 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer disabled:opacity-50"
            :disabled="syncingBalances"
            title="向百智云实时刷新所有配置了 Cookie 的账号积分与余额"
            @click="syncAllBalances"
          >
            <Icon name="bolt" size="sm" :class="{ 'animate-spin': syncingBalances }" />
            <span>{{ syncingBalances ? '正在同步...' : '同步全部账号积分' }}</span>
          </button>

          <button
            type="button"
            class="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-medium bg-white dark:bg-dark-800 text-gray-800 dark:text-gray-100 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer disabled:opacity-50"
            :disabled="loading"
            @click="loadConfig"
          >
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            <span>{{ t('common.refresh') }}</span>
          </button>

          <button
            type="button"
            class="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-semibold text-white bg-gray-900 hover:bg-black dark:bg-teal-600 dark:hover:bg-teal-500 shadow-sm transition-all cursor-pointer disabled:opacity-50"
            :disabled="saving"
            @click="saveConfig"
          >
            <Icon name="check" size="sm" :class="{ 'animate-spin': saving }" />
            <span>{{ saving ? t('common.saving') : t('admin.baizhi.overview.saveConfig') }}</span>
          </button>
        </div>
      </div>

      <!-- Top Gateway Overview Card -->
      <div class="rounded-2xl border border-gray-200/90 bg-white p-5 shadow-xs dark:border-dark-700 dark:bg-dark-800">
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-12 lg:items-center">
          <!-- Gateway Base URL -->
          <div class="space-y-2 lg:col-span-7">
            <div class="flex items-center gap-2">
              <span class="inline-flex h-2 w-2 rounded-full bg-emerald-500 animate-pulse"></span>
              <span class="text-xs font-bold uppercase tracking-wider text-gray-600 dark:text-gray-400">
                {{ t('admin.baizhi.overview.gatewayBaseUrl') }}
              </span>
            </div>
            <div class="flex items-center gap-2">
              <code class="rounded-xl border border-gray-200 bg-gray-100/90 px-3.5 py-1.5 font-mono text-sm font-bold text-gray-900 select-all dark:border-dark-700 dark:bg-dark-900 dark:text-teal-300">
                {{ gatewayBaseUrl }}
              </code>
              <button
                type="button"
                class="inline-flex items-center gap-1 px-3 py-1.5 rounded-full text-xs font-semibold bg-white dark:bg-dark-800 text-gray-800 dark:text-gray-100 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer"
                :title="t('admin.baizhi.docs.copyCode')"
                @click="copyGatewayUrl"
              >
                <Icon :name="copiedUrl ? 'check' : 'clipboard'" size="sm" />
                <span>{{ copiedUrl ? t('admin.baizhi.docs.copied') : t('common.copy') }}</span>
              </button>
            </div>
          </div>

          <!-- Auth & Stats -->
          <div class="flex flex-wrap items-center justify-between gap-4 border-t border-gray-100 pt-3 lg:col-span-5 lg:border-t-0 lg:border-l lg:border-gray-100 lg:pl-6 lg:pt-0 dark:border-dark-700">
            <!-- Auth Required Switch -->
            <div class="flex items-center gap-2.5">
              <input
                id="authRequiredToggle"
                v-model="config.auth_required"
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-gray-900 focus:ring-gray-900 dark:border-dark-600 dark:bg-dark-700 dark:text-teal-500 cursor-pointer"
              />
              <label for="authRequiredToggle" class="cursor-pointer text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ t('admin.baizhi.overview.authRequired') }}
              </label>
            </div>

            <!-- Stats badges -->
            <div class="flex items-center gap-2">
              <span class="inline-flex items-center gap-1.5 rounded-full bg-gray-100 dark:bg-dark-700/80 px-3 py-1 text-xs font-semibold text-gray-800 dark:text-gray-200 border border-gray-200/80 dark:border-dark-600">
                <Icon name="cube" size="xs" />
                {{ t('admin.baizhi.overview.activeServices') }}: {{ activeServicesCount }}/4
              </span>
              <span class="inline-flex items-center gap-1.5 rounded-full bg-gray-100 dark:bg-dark-700/80 px-3 py-1 text-xs font-semibold text-gray-800 dark:text-gray-200 border border-gray-200/80 dark:border-dark-600">
                <Icon name="key" size="xs" />
                {{ t('admin.baizhi.overview.totalKeys') }}: {{ totalKeysCount }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab Navigation -->
      <div class="flex border-b border-gray-200 dark:border-dark-700">
        <button
          type="button"
          class="inline-flex items-center gap-2 border-b-2 px-4 py-2.5 text-sm transition-colors cursor-pointer"
          :class="[
            activeTab === 'config'
              ? 'border-gray-900 text-gray-900 font-bold dark:border-white dark:text-white'
              : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200 font-medium'
          ]"
          @click="activeTab = 'config'"
        >
          <Icon name="cog" size="sm" />
          <span>{{ t('admin.baizhi.tabs.config') }}</span>
        </button>

        <button
          type="button"
          class="inline-flex items-center gap-2 border-b-2 px-4 py-2.5 text-sm transition-colors cursor-pointer"
          :class="[
            activeTab === 'docs'
              ? 'border-gray-900 text-gray-900 font-bold dark:border-white dark:text-white'
              : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200 font-medium'
          ]"
          @click="activeTab = 'docs'"
        >
          <Icon name="document" size="sm" />
          <span>{{ t('admin.baizhi.tabs.docs') }}</span>
        </button>
      </div>

      <!-- TAB 1: Configuration Cards -->
      <div v-show="activeTab === 'config'" class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- Card 1: Web Scrape -->
        <div class="flex flex-col justify-between rounded-2xl border border-gray-200/90 bg-white p-5 shadow-xs dark:border-dark-700 dark:bg-dark-800">
          <div class="space-y-4">
            <!-- Header -->
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2.5">
                <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200">
                  <Icon name="globe" size="md" />
                </div>
                <h3 class="font-bold text-gray-900 dark:text-white text-base">
                  {{ t('admin.baizhi.services.scrape') }}
                </h3>
              </div>
              <label class="relative inline-flex cursor-pointer items-center">
                <input v-model="config.web_scrape.enabled" type="checkbox" class="peer sr-only" />
                <div class="peer h-6 w-11 rounded-full bg-gray-200 peer-checked:bg-gray-900 dark:peer-checked:bg-teal-600 peer-focus:outline-none dark:bg-dark-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all"></div>
              </label>
            </div>

            <!-- Route Prefix & Upstream -->
            <div class="flex items-center justify-between text-xs px-3.5 py-2 rounded-xl bg-gray-50 dark:bg-dark-900/60 border border-gray-200/70 dark:border-dark-700/60">
              <span class="font-mono font-bold text-gray-900 dark:text-gray-100 select-all">/baizhi/scrape/*</span>
              <span class="font-mono text-gray-500 dark:text-gray-400 text-[11px] truncate max-w-[210px]">{{ config.web_scrape.base_url }}</span>
            </div>

            <!-- Dispatch Strategy via Pill Popover -->
            <div class="flex items-center justify-between py-0.5">
              <span class="text-xs font-semibold text-gray-700 dark:text-gray-300">{{ t('admin.baizhi.card.strategy') }}</span>
              <PillPopover
                v-model="config.web_scrape.strategy"
                :options="strategyOptions"
                group-title="调度模式"
                align="end"
              />
            </div>

            <!-- Key List -->
            <div class="space-y-2 pt-1 border-t border-gray-100 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <span class="text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                  {{ t('admin.baizhi.card.activeKeysCount') }} ({{ config.web_scrape.keys.filter(k => k.enabled).length }}/{{ config.web_scrape.keys.length }})
                </span>
                <button
                  type="button"
                  class="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold bg-white dark:bg-dark-800 text-gray-800 dark:text-gray-100 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer"
                  @click="openAddKeyDialog('web_scrape')"
                >
                  <Icon name="plus" size="xs" />
                  <span>{{ t('admin.baizhi.card.addKey') }}</span>
                </button>
              </div>

              <!-- Keys Table -->
              <div v-if="config.web_scrape.keys.length === 0" class="rounded-xl border border-dashed border-gray-200 py-6 text-center text-xs text-gray-400 dark:border-dark-700">
                {{ t('admin.baizhi.card.noKeys') }}
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="(keyItem, idx) in config.web_scrape.keys"
                  :key="keyItem.id || idx"
                  class="flex items-center justify-between rounded-xl border border-gray-200/80 bg-gray-50/70 p-2.5 text-xs dark:border-dark-700 dark:bg-dark-900/50"
                >
                  <div class="min-w-0 flex-1 space-y-0.5">
                    <div class="flex flex-wrap items-center gap-1.5">
                      <img v-if="keyItem.avatar" :src="keyItem.avatar" class="w-4 h-4 rounded-full" />
                      <span class="font-semibold text-gray-900 dark:text-white">{{ keyItem.account_name || 'Account' }}</span>

                      <span
                        v-if="keyItem.temporary_credits !== undefined"
                        class="inline-flex items-center gap-0.5 px-2 py-0.5 rounded-full text-[10px] font-bold border"
                        :class="keyItem.temporary_credits > 0 ? 'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/50 dark:text-emerald-300 dark:border-emerald-800' : 'bg-gray-100 text-gray-500 border-gray-200 dark:bg-dark-800 dark:text-gray-400 dark:border-dark-700'"
                        :title="`剩余临时积分: ${keyItem.temporary_credits}`"
                      >
                        ⚡ {{ keyItem.temporary_credits > 0 ? `${keyItem.temporary_credits} 临时` : '临时已尽' }}
                      </span>

                      <span
                        v-if="keyItem.permanent_credits !== undefined && keyItem.permanent_credits > 0"
                        class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-[10px] text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-dark-800"
                        :title="`永久积分: ${keyItem.permanent_credits}`"
                      >
                        💰 {{ keyItem.permanent_credits }}
                      </span>

                      <span
                        v-if="keyItem.daily_claimed"
                        class="text-[10px] text-teal-600 dark:text-teal-400 font-medium"
                        title="今日签到临时积分已领"
                      >
                        ✓ 已领
                      </span>

                      <span v-if="testStates[keyItem.id]?.latency" class="inline-flex items-center gap-0.5 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400">
                        <Icon name="checkCircle" size="xs" />
                        {{ testStates[keyItem.id].latency }}ms
                      </span>
                      <span v-else-if="testStates[keyItem.id]?.error" class="inline-flex items-center gap-0.5 text-[11px] font-semibold text-rose-600 dark:text-rose-400">
                        <Icon name="xCircle" size="xs" />
                        {{ testStates[keyItem.id].error }}
                      </span>
                    </div>
                    <div class="font-mono text-gray-600 dark:text-gray-400 select-all text-[11.5px] truncate max-w-[320px]">
                      {{ keyItem.api_key }}
                    </div>
                  </div>

                  <div class="flex items-center gap-2 pl-2">
                    <input
                      v-model="keyItem.enabled"
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-gray-900 focus:ring-gray-900 dark:border-dark-600 dark:bg-dark-700 cursor-pointer"
                      :title="t('admin.baizhi.card.status')"
                    />
                    <button
                      type="button"
                      class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium bg-white dark:bg-dark-800 text-gray-700 dark:text-gray-200 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer disabled:opacity-50"
                      :disabled="testStates[keyItem.id]?.loading"
                      :title="t('admin.baizhi.card.testConnection')"
                      @click="runTest('web_scrape', keyItem)"
                    >
                      <Icon name="bolt" size="xs" :class="{ 'animate-spin': testStates[keyItem.id]?.loading }" />
                      <span class="hidden sm:inline">{{ t('admin.baizhi.card.testConnection') }}</span>
                    </button>
                    <button
                      type="button"
                      class="p-1.5 rounded-full text-gray-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition-colors cursor-pointer"
                      @click="removeKey('web_scrape', idx)"
                    >
                      <Icon name="trash" size="xs" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Card 2: Web Search -->
        <div class="flex flex-col justify-between rounded-2xl border border-gray-200/90 bg-white p-5 shadow-xs dark:border-dark-700 dark:bg-dark-800">
          <div class="space-y-4">
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2.5">
                <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200">
                  <Icon name="search" size="md" />
                </div>
                <h3 class="font-bold text-gray-900 dark:text-white text-base">
                  {{ t('admin.baizhi.services.search') }}
                </h3>
              </div>
              <label class="relative inline-flex cursor-pointer items-center">
                <input v-model="config.web_search.enabled" type="checkbox" class="peer sr-only" />
                <div class="peer h-6 w-11 rounded-full bg-gray-200 peer-checked:bg-gray-900 dark:peer-checked:bg-teal-600 peer-focus:outline-none dark:bg-dark-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all"></div>
              </label>
            </div>

            <div class="flex items-center justify-between text-xs px-3.5 py-2 rounded-xl bg-gray-50 dark:bg-dark-900/60 border border-gray-200/70 dark:border-dark-700/60">
              <span class="font-mono font-bold text-gray-900 dark:text-gray-100 select-all">/baizhi/search/*</span>
              <span class="font-mono text-gray-500 dark:text-gray-400 text-[11px] truncate max-w-[210px]">{{ config.web_search.base_url }}</span>
            </div>

            <!-- Dispatch Strategy via Pill Popover -->
            <div class="flex items-center justify-between py-0.5">
              <span class="text-xs font-semibold text-gray-700 dark:text-gray-300">{{ t('admin.baizhi.card.strategy') }}</span>
              <PillPopover
                v-model="config.web_search.strategy"
                :options="strategyOptions"
                group-title="调度模式"
                align="end"
              />
            </div>

            <!-- Key List -->
            <div class="space-y-2 pt-1 border-t border-gray-100 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <span class="text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                  {{ t('admin.baizhi.card.activeKeysCount') }} ({{ config.web_search.keys.filter(k => k.enabled).length }}/{{ config.web_search.keys.length }})
                </span>
                <button
                  type="button"
                  class="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold bg-white dark:bg-dark-800 text-gray-800 dark:text-gray-100 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer"
                  @click="openAddKeyDialog('web_search')"
                >
                  <Icon name="plus" size="xs" />
                  <span>{{ t('admin.baizhi.card.addKey') }}</span>
                </button>
              </div>

              <div v-if="config.web_search.keys.length === 0" class="rounded-xl border border-dashed border-gray-200 py-6 text-center text-xs text-gray-400 dark:border-dark-700">
                {{ t('admin.baizhi.card.noKeys') }}
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="(keyItem, idx) in config.web_search.keys"
                  :key="keyItem.id || idx"
                  class="flex items-center justify-between rounded-xl border border-gray-200/80 bg-gray-50/70 p-2.5 text-xs dark:border-dark-700 dark:bg-dark-900/50"
                >
                  <div class="min-w-0 flex-1 space-y-0.5">
                    <div class="flex flex-wrap items-center gap-1.5">
                      <img v-if="keyItem.avatar" :src="keyItem.avatar" class="w-4 h-4 rounded-full" />
                      <span class="font-semibold text-gray-900 dark:text-white">{{ keyItem.account_name || 'Account' }}</span>

                      <span
                        v-if="keyItem.temporary_credits !== undefined"
                        class="inline-flex items-center gap-0.5 px-2 py-0.5 rounded-full text-[10px] font-bold border"
                        :class="keyItem.temporary_credits > 0 ? 'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/50 dark:text-emerald-300 dark:border-emerald-800' : 'bg-gray-100 text-gray-500 border-gray-200 dark:bg-dark-800 dark:text-gray-400 dark:border-dark-700'"
                        :title="`剩余临时积分: ${keyItem.temporary_credits}`"
                      >
                        ⚡ {{ keyItem.temporary_credits > 0 ? `${keyItem.temporary_credits} 临时` : '临时已尽' }}
                      </span>

                      <span
                        v-if="keyItem.permanent_credits !== undefined && keyItem.permanent_credits > 0"
                        class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-[10px] text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-dark-800"
                        :title="`永久积分: ${keyItem.permanent_credits}`"
                      >
                        💰 {{ keyItem.permanent_credits }}
                      </span>

                      <span
                        v-if="keyItem.daily_claimed"
                        class="text-[10px] text-teal-600 dark:text-teal-400 font-medium"
                        title="今日签到临时积分已领"
                      >
                        ✓ 已领
                      </span>

                      <span v-if="testStates[keyItem.id]?.latency" class="inline-flex items-center gap-0.5 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400">
                        <Icon name="checkCircle" size="xs" />
                        {{ testStates[keyItem.id].latency }}ms
                      </span>
                      <span v-else-if="testStates[keyItem.id]?.error" class="inline-flex items-center gap-0.5 text-[11px] font-semibold text-rose-600 dark:text-rose-400">
                        <Icon name="xCircle" size="xs" />
                        {{ testStates[keyItem.id].error }}
                      </span>
                    </div>
                    <div class="font-mono text-gray-600 dark:text-gray-400 select-all text-[11.5px] truncate max-w-[320px]">
                      {{ keyItem.api_key }}
                    </div>
                  </div>

                  <div class="flex items-center gap-2 pl-2">
                    <input
                      v-model="keyItem.enabled"
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-gray-900 focus:ring-gray-900 dark:border-dark-600 dark:bg-dark-700 cursor-pointer"
                    />
                    <button
                      type="button"
                      class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium bg-white dark:bg-dark-800 text-gray-700 dark:text-gray-200 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer disabled:opacity-50"
                      :disabled="testStates[keyItem.id]?.loading"
                      @click="runTest('web_search', keyItem)"
                    >
                      <Icon name="bolt" size="xs" :class="{ 'animate-spin': testStates[keyItem.id]?.loading }" />
                      <span class="hidden sm:inline">{{ t('admin.baizhi.card.testConnection') }}</span>
                    </button>
                    <button
                      type="button"
                      class="p-1.5 rounded-full text-gray-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition-colors cursor-pointer"
                      @click="removeKey('web_search', idx)"
                    >
                      <Icon name="trash" size="xs" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Card 3: Voice ASR/TTS -->
        <div class="flex flex-col justify-between rounded-2xl border border-gray-200/90 bg-white p-5 shadow-xs dark:border-dark-700 dark:bg-dark-800">
          <div class="space-y-4">
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2.5">
                <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200">
                  <Icon name="chat" size="md" />
                </div>
                <h3 class="font-bold text-gray-900 dark:text-white text-base">
                  {{ t('admin.baizhi.services.voice') }}
                </h3>
              </div>
              <label class="relative inline-flex cursor-pointer items-center">
                <input v-model="config.voice.enabled" type="checkbox" class="peer sr-only" />
                <div class="peer h-6 w-11 rounded-full bg-gray-200 peer-checked:bg-gray-900 dark:peer-checked:bg-teal-600 peer-focus:outline-none dark:bg-dark-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all"></div>
              </label>
            </div>

            <div class="flex items-center justify-between text-xs px-3.5 py-2 rounded-xl bg-gray-50 dark:bg-dark-900/60 border border-gray-200/70 dark:border-dark-700/60">
              <span class="font-mono font-bold text-gray-900 dark:text-gray-100 select-all">/baizhi/voice/* (HTTP & WS)</span>
              <span class="font-mono text-gray-500 dark:text-gray-400 text-[11px] truncate max-w-[210px]">{{ config.voice.base_url }}</span>
            </div>

            <!-- Dispatch Strategy via Pill Popover -->
            <div class="flex items-center justify-between py-0.5">
              <span class="text-xs font-semibold text-gray-700 dark:text-gray-300">{{ t('admin.baizhi.card.strategy') }}</span>
              <PillPopover
                v-model="config.voice.strategy"
                :options="strategyOptions"
                group-title="调度模式"
                align="end"
              />
            </div>

            <!-- Key List -->
            <div class="space-y-2 pt-1 border-t border-gray-100 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <span class="text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                  {{ t('admin.baizhi.card.activeKeysCount') }} ({{ config.voice.keys.filter(k => k.enabled).length }}/{{ config.voice.keys.length }})
                </span>
                <button
                  type="button"
                  class="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold bg-white dark:bg-dark-800 text-gray-800 dark:text-gray-100 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer"
                  @click="openAddKeyDialog('voice')"
                >
                  <Icon name="plus" size="xs" />
                  <span>{{ t('admin.baizhi.card.addKey') }}</span>
                </button>
              </div>

              <div v-if="config.voice.keys.length === 0" class="rounded-xl border border-dashed border-gray-200 py-6 text-center text-xs text-gray-400 dark:border-dark-700">
                {{ t('admin.baizhi.card.noKeys') }}
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="(keyItem, idx) in config.voice.keys"
                  :key="keyItem.id || idx"
                  class="flex items-center justify-between rounded-xl border border-gray-200/80 bg-gray-50/70 p-2.5 text-xs dark:border-dark-700 dark:bg-dark-900/50"
                >
                  <div class="min-w-0 flex-1 space-y-0.5">
                    <div class="flex flex-wrap items-center gap-1.5">
                      <img v-if="keyItem.avatar" :src="keyItem.avatar" class="w-4 h-4 rounded-full" />
                      <span class="font-semibold text-gray-900 dark:text-white">{{ keyItem.account_name || 'Account' }}</span>

                      <span
                        v-if="keyItem.temporary_credits !== undefined"
                        class="inline-flex items-center gap-0.5 px-2 py-0.5 rounded-full text-[10px] font-bold border"
                        :class="keyItem.temporary_credits > 0 ? 'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/50 dark:text-emerald-300 dark:border-emerald-800' : 'bg-gray-100 text-gray-500 border-gray-200 dark:bg-dark-800 dark:text-gray-400 dark:border-dark-700'"
                        :title="`剩余临时积分: ${keyItem.temporary_credits}`"
                      >
                        ⚡ {{ keyItem.temporary_credits > 0 ? `${keyItem.temporary_credits} 临时` : '临时已尽' }}
                      </span>

                      <span
                        v-if="keyItem.permanent_credits !== undefined && keyItem.permanent_credits > 0"
                        class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-[10px] text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-dark-800"
                        :title="`永久积分: ${keyItem.permanent_credits}`"
                      >
                        💰 {{ keyItem.permanent_credits }}
                      </span>

                      <span
                        v-if="keyItem.daily_claimed"
                        class="text-[10px] text-teal-600 dark:text-teal-400 font-medium"
                        title="今日签到临时积分已领"
                      >
                        ✓ 已领
                      </span>

                      <span v-if="testStates[keyItem.id]?.latency" class="inline-flex items-center gap-0.5 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400">
                        <Icon name="checkCircle" size="xs" />
                        {{ testStates[keyItem.id].latency }}ms
                      </span>
                      <span v-else-if="testStates[keyItem.id]?.error" class="inline-flex items-center gap-0.5 text-[11px] font-semibold text-rose-600 dark:text-rose-400">
                        <Icon name="xCircle" size="xs" />
                        {{ testStates[keyItem.id].error }}
                      </span>
                    </div>
                    <div class="font-mono text-gray-600 dark:text-gray-400 select-all text-[11.5px] truncate max-w-[320px]">
                      {{ keyItem.api_key }}
                    </div>
                  </div>

                  <div class="flex items-center gap-2 pl-2">
                    <input
                      v-model="keyItem.enabled"
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-gray-900 focus:ring-gray-900 dark:border-dark-600 dark:bg-dark-700 cursor-pointer"
                    />
                    <button
                      type="button"
                      class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium bg-white dark:bg-dark-800 text-gray-700 dark:text-gray-200 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer disabled:opacity-50"
                      :disabled="testStates[keyItem.id]?.loading"
                      @click="runTest('voice', keyItem)"
                    >
                      <Icon name="bolt" size="xs" :class="{ 'animate-spin': testStates[keyItem.id]?.loading }" />
                      <span class="hidden sm:inline">{{ t('admin.baizhi.card.testConnection') }}</span>
                    </button>
                    <button
                      type="button"
                      class="p-1.5 rounded-full text-gray-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition-colors cursor-pointer"
                      @click="removeKey('voice', idx)"
                    >
                      <Icon name="trash" size="xs" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Card 4: Image Generate -->
        <div class="flex flex-col justify-between rounded-2xl border border-gray-200/90 bg-white p-5 shadow-xs dark:border-dark-700 dark:bg-dark-800">
          <div class="space-y-4">
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2.5">
                <div class="flex h-9 w-9 items-center justify-center rounded-xl bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200">
                  <Icon name="sparkles" size="md" />
                </div>
                <h3 class="font-bold text-gray-900 dark:text-white text-base">
                  {{ t('admin.baizhi.services.image') }}
                </h3>
              </div>
              <label class="relative inline-flex cursor-pointer items-center">
                <input v-model="config.image_generate.enabled" type="checkbox" class="peer sr-only" />
                <div class="peer h-6 w-11 rounded-full bg-gray-200 peer-checked:bg-gray-900 dark:peer-checked:bg-teal-600 peer-focus:outline-none dark:bg-dark-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all"></div>
              </label>
            </div>

            <div class="flex items-center justify-between text-xs px-3.5 py-2 rounded-xl bg-gray-50 dark:bg-dark-900/60 border border-gray-200/70 dark:border-dark-700/60">
              <span class="font-mono font-bold text-gray-900 dark:text-gray-100 select-all">/baizhi/image/*</span>
              <span class="font-mono text-gray-500 dark:text-gray-400 text-[11px] truncate max-w-[210px]">{{ config.image_generate.base_url }}</span>
            </div>

            <!-- Dispatch Strategy via Pill Popover -->
            <div class="flex items-center justify-between py-0.5">
              <span class="text-xs font-semibold text-gray-700 dark:text-gray-300">{{ t('admin.baizhi.card.strategy') }}</span>
              <PillPopover
                v-model="config.image_generate.strategy"
                :options="strategyOptions"
                group-title="调度模式"
                align="end"
              />
            </div>

            <!-- Key List -->
            <div class="space-y-2 pt-1 border-t border-gray-100 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <span class="text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                  {{ t('admin.baizhi.card.activeKeysCount') }} ({{ config.image_generate.keys.filter(k => k.enabled).length }}/{{ config.image_generate.keys.length }})
                </span>
                <button
                  type="button"
                  class="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold bg-white dark:bg-dark-800 text-gray-800 dark:text-gray-100 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer"
                  @click="openAddKeyDialog('image_generate')"
                >
                  <Icon name="plus" size="xs" />
                  <span>{{ t('admin.baizhi.card.addKey') }}</span>
                </button>
              </div>

              <div v-if="config.image_generate.keys.length === 0" class="rounded-xl border border-dashed border-gray-200 py-6 text-center text-xs text-gray-400 dark:border-dark-700">
                {{ t('admin.baizhi.card.noKeys') }}
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="(keyItem, idx) in config.image_generate.keys"
                  :key="keyItem.id || idx"
                  class="flex items-center justify-between rounded-xl border border-gray-200/80 bg-gray-50/70 p-2.5 text-xs dark:border-dark-700 dark:bg-dark-900/50"
                >
                  <div class="min-w-0 flex-1 space-y-0.5">
                    <div class="flex flex-wrap items-center gap-1.5">
                      <img v-if="keyItem.avatar" :src="keyItem.avatar" class="w-4 h-4 rounded-full" />
                      <span class="font-semibold text-gray-900 dark:text-white">{{ keyItem.account_name || 'Account' }}</span>

                      <span
                        v-if="keyItem.temporary_credits !== undefined"
                        class="inline-flex items-center gap-0.5 px-2 py-0.5 rounded-full text-[10px] font-bold border"
                        :class="keyItem.temporary_credits > 0 ? 'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/50 dark:text-emerald-300 dark:border-emerald-800' : 'bg-gray-100 text-gray-500 border-gray-200 dark:bg-dark-800 dark:text-gray-400 dark:border-dark-700'"
                        :title="`剩余临时积分: ${keyItem.temporary_credits}`"
                      >
                        ⚡ {{ keyItem.temporary_credits > 0 ? `${keyItem.temporary_credits} 临时` : '临时已尽' }}
                      </span>

                      <span
                        v-if="keyItem.permanent_credits !== undefined && keyItem.permanent_credits > 0"
                        class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-[10px] text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-dark-800"
                        :title="`永久积分: ${keyItem.permanent_credits}`"
                      >
                        💰 {{ keyItem.permanent_credits }}
                      </span>

                      <span
                        v-if="keyItem.daily_claimed"
                        class="text-[10px] text-teal-600 dark:text-teal-400 font-medium"
                        title="今日签到临时积分已领"
                      >
                        ✓ 已领
                      </span>

                      <span v-if="testStates[keyItem.id]?.latency" class="inline-flex items-center gap-0.5 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400">
                        <Icon name="checkCircle" size="xs" />
                        {{ testStates[keyItem.id].latency }}ms
                      </span>
                      <span v-else-if="testStates[keyItem.id]?.error" class="inline-flex items-center gap-0.5 text-[11px] font-semibold text-rose-600 dark:text-rose-400">
                        <Icon name="xCircle" size="xs" />
                        {{ testStates[keyItem.id].error }}
                      </span>
                    </div>
                    <div class="font-mono text-gray-600 dark:text-gray-400 select-all text-[11.5px] truncate max-w-[320px]">
                      {{ keyItem.api_key }}
                    </div>
                  </div>

                  <div class="flex items-center gap-2 pl-2">
                    <input
                      v-model="keyItem.enabled"
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-gray-900 focus:ring-gray-900 dark:border-dark-600 dark:bg-dark-700 cursor-pointer"
                    />
                    <button
                      type="button"
                      class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium bg-white dark:bg-dark-800 text-gray-700 dark:text-gray-200 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer disabled:opacity-50"
                      :disabled="testStates[keyItem.id]?.loading"
                      @click="runTest('image_generate', keyItem)"
                    >
                      <Icon name="bolt" size="xs" :class="{ 'animate-spin': testStates[keyItem.id]?.loading }" />
                      <span class="hidden sm:inline">{{ t('admin.baizhi.card.testConnection') }}</span>
                    </button>
                    <button
                      type="button"
                      class="p-1.5 rounded-full text-gray-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition-colors cursor-pointer"
                      @click="removeKey('image_generate', idx)"
                    >
                      <Icon name="trash" size="xs" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- TAB 2: Documentation View -->
      <div v-show="activeTab === 'docs'" class="space-y-6">
        <!-- Documentation Header -->
        <div class="flex items-center justify-between text-xs px-3.5 py-2.5 rounded-xl bg-gray-50 dark:bg-dark-900/60 border border-gray-200/80 dark:border-dark-700">
          <span class="font-mono text-gray-700 dark:text-gray-300 font-medium">
            鉴权请求头格式: <code class="px-2 py-0.5 rounded bg-gray-200/80 dark:bg-dark-800 text-gray-900 dark:text-white font-bold">Authorization: Bearer YOUR_SUB2API_API_KEY</code>
          </span>
        </div>

        <!-- 1. Web Scrape Docs -->
        <div class="rounded-2xl border border-gray-200/90 bg-white p-6 shadow-xs dark:border-dark-700 dark:bg-dark-800 space-y-4">
          <div class="flex items-center gap-2.5 border-b border-gray-100 pb-3 dark:border-dark-700">
            <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200">
              <Icon name="globe" size="sm" />
            </div>
            <h3 class="text-base font-bold text-gray-900 dark:text-white">1. {{ t('admin.baizhi.services.scrape') }}</h3>
          </div>

          <div class="overflow-x-auto">
            <table class="w-full text-left text-xs">
              <thead class="bg-gray-50 text-gray-700 dark:bg-dark-900/50 dark:text-gray-300 font-semibold">
                <tr>
                  <th class="p-2.5">方法</th>
                  <th class="p-2.5">网关入口路径</th>
                  <th class="p-2.5">功能说明</th>
                  <th class="p-2.5">计费标准</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700 text-gray-800 dark:text-gray-200 font-mono">
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/scrape/openapi/scrape</td>
                  <td class="p-2.5 font-sans">打开网页返回整页 Markdown 或 JSON 正文</td>
                  <td class="p-2.5 font-sans">5 积分/次</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/scrape/openapi/extract</td>
                  <td class="p-2.5 font-sans">从网页结构化提取指定字段 (fields)</td>
                  <td class="p-2.5 font-sans">10 积分/次</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/scrape/openapi/extract/suggest</td>
                  <td class="p-2.5 font-sans">智能提取字段建议 (生成 name/type/hint)</td>
                  <td class="p-2.5 font-sans">2 积分/次</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/scrape/openapi/extract/batch</td>
                  <td class="p-2.5 font-sans">批量多网页提取相同字段 (1-20页)</td>
                  <td class="p-2.5 font-sans">10 积分/成功页</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/scrape/openapi/extract/organize</td>
                  <td class="p-2.5 font-sans">把多页提取结果整理合成一篇 Markdown 结构稿</td>
                  <td class="p-2.5 font-sans">5 积分/次</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-info text-[10px]">GET</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/scrape/openapi/v1/pricing</td>
                  <td class="p-2.5 font-sans">查询当前积分与功能定价规则</td>
                  <td class="p-2.5 font-sans">免费</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Code Snippet -->
          <div class="rounded-xl bg-gray-900 p-4 font-mono text-xs text-gray-200 shadow-sm">
            <div class="flex items-center justify-between pb-2 text-gray-400 border-b border-gray-800">
              <span class="font-sans">cURL 示例 (POST /baizhi/scrape/openapi/scrape)</span>
              <button
                type="button"
                class="px-2.5 py-1 rounded-full border border-gray-700 bg-gray-800 hover:bg-gray-700 text-gray-200 hover:text-white transition-all cursor-pointer"
                @click="copySnippet(scrapeSnippet)"
              >
                复制
              </button>
            </div>
            <pre class="mt-2 text-emerald-400 overflow-x-auto">{{ scrapeSnippet }}</pre>
          </div>
        </div>

        <!-- 2. Web Search Docs -->
        <div class="rounded-2xl border border-gray-200/90 bg-white p-6 shadow-xs dark:border-dark-700 dark:bg-dark-800 space-y-4">
          <div class="flex items-center gap-2.5 border-b border-gray-100 pb-3 dark:border-dark-700">
            <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200">
              <Icon name="search" size="sm" />
            </div>
            <h3 class="text-base font-bold text-gray-900 dark:text-white">2. {{ t('admin.baizhi.services.search') }}</h3>
          </div>

          <div class="overflow-x-auto">
            <table class="w-full text-left text-xs">
              <thead class="bg-gray-50 text-gray-700 dark:bg-dark-900/50 dark:text-gray-300 font-semibold">
                <tr>
                  <th class="p-2.5">方法</th>
                  <th class="p-2.5">网关入口路径</th>
                  <th class="p-2.5">功能说明</th>
                  <th class="p-2.5">计费标准</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700 text-gray-800 dark:text-gray-200 font-mono">
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/search/openapi/search</td>
                  <td class="p-2.5 font-sans">通用网页关键词搜索 (含标题、摘要、发布时间、域名过滤)</td>
                  <td class="p-2.5 font-sans">2.5 积分/次</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/search/openapi/ai_search</td>
                  <td class="p-2.5 font-sans">AI 综合搜索 (网关全程 SSE 实时流式响应 delta 推送)</td>
                  <td class="p-2.5 font-sans">2.5 积分/次</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/search/openapi/news_search</td>
                  <td class="p-2.5 font-sans">近期新闻事件搜索 (带 AI 综合要闻摘要 need_summary)</td>
                  <td class="p-2.5 font-sans">10 积分/次</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/search/openapi/image_search_by_text</td>
                  <td class="p-2.5 font-sans">文本搜索高清图片 (支持构图 wide/tall/square 筛选)</td>
                  <td class="p-2.5 font-sans">2.5 积分/次</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/search/openapi/image_search_by_image</td>
                  <td class="p-2.5 font-sans">以图搜图 (支持 multipart/form-data 原图上传)</td>
                  <td class="p-2.5 font-sans">10 积分/次</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Code Snippet -->
          <div class="rounded-xl bg-gray-900 p-4 font-mono text-xs text-gray-200 shadow-sm">
            <div class="flex items-center justify-between pb-2 text-gray-400 border-b border-gray-800">
              <span class="font-sans">cURL 示例 (POST /baizhi/search/openapi/search)</span>
              <button
                type="button"
                class="px-2.5 py-1 rounded-full border border-gray-700 bg-gray-800 hover:bg-gray-700 text-gray-200 hover:text-white transition-all cursor-pointer"
                @click="copySnippet(searchSnippet)"
              >
                复制
              </button>
            </div>
            <pre class="mt-2 text-purple-400 overflow-x-auto">{{ searchSnippet }}</pre>
          </div>
        </div>

        <!-- 3. Voice Docs -->
        <div class="rounded-2xl border border-gray-200/90 bg-white p-6 shadow-xs dark:border-dark-700 dark:bg-dark-800 space-y-4">
          <div class="flex items-center gap-2.5 border-b border-gray-100 pb-3 dark:border-dark-700">
            <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200">
              <Icon name="chat" size="sm" />
            </div>
            <h3 class="text-base font-bold text-gray-900 dark:text-white">3. {{ t('admin.baizhi.services.voice') }}</h3>
          </div>

          <div class="overflow-x-auto">
            <table class="w-full text-left text-xs">
              <thead class="bg-gray-50 text-gray-700 dark:bg-dark-900/50 dark:text-gray-300 font-semibold">
                <tr>
                  <th class="p-2.5">方法</th>
                  <th class="p-2.5">网关入口路径</th>
                  <th class="p-2.5">功能说明</th>
                  <th class="p-2.5">格式/协议</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700 text-gray-800 dark:text-gray-200 font-mono">
                <tr>
                  <td class="p-2.5"><span class="badge badge-warning text-[10px]">WSS</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/voice/openapi/v1/realtime</td>
                  <td class="p-2.5 font-sans">实时语音转文字 (STT) 或 文字转语音 (TTS)</td>
                  <td class="p-2.5 font-sans">WebSocket 全双工</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/voice/openapi/v1/audio</td>
                  <td class="p-2.5 font-sans">录音文件识别任务创建 或 HTTP 文字转语音 (WAV)</td>
                  <td class="p-2.5 font-sans">JSON / WAV 音频流</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-info text-[10px]">GET</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/voice/openapi/v1/audio/:task_id</td>
                  <td class="p-2.5 font-sans">查询录音识别状态及分句文本 (utterances)</td>
                  <td class="p-2.5 font-sans">JSON</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-info text-[10px]">GET</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/voice/openapi/v1/audio/:task_id/export</td>
                  <td class="p-2.5 font-sans">导出字幕或全文本 (?format=srt|vtt|txt)</td>
                  <td class="p-2.5 font-sans">Text / Subtitle</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-info text-[10px]">GET</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/voice/openapi/v1/voices?type=tts</td>
                  <td class="p-2.5 font-sans">查询所有可用音色列表与试听地址</td>
                  <td class="p-2.5 font-sans">JSON</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Code Snippet -->
          <div class="rounded-xl bg-gray-900 p-4 font-mono text-xs text-gray-200 shadow-sm">
            <div class="flex items-center justify-between pb-2 text-gray-400 border-b border-gray-800">
              <span class="font-sans">cURL 示例 (HTTP 文字转语音 POST /baizhi/voice/openapi/v1/audio)</span>
              <button
                type="button"
                class="px-2.5 py-1 rounded-full border border-gray-700 bg-gray-800 hover:bg-gray-700 text-gray-200 hover:text-white transition-all cursor-pointer"
                @click="copySnippet(voiceSnippet)"
              >
                复制
              </button>
            </div>
            <pre class="mt-2 text-amber-400 overflow-x-auto">{{ voiceSnippet }}</pre>
          </div>
        </div>

        <!-- 4. Image Generate Docs -->
        <div class="rounded-2xl border border-gray-200/90 bg-white p-6 shadow-xs dark:border-dark-700 dark:bg-dark-800 space-y-4">
          <div class="flex items-center gap-2.5 border-b border-gray-100 pb-3 dark:border-dark-700">
            <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-100 text-gray-800 dark:bg-dark-700 dark:text-gray-200">
              <Icon name="sparkles" size="sm" />
            </div>
            <h3 class="text-base font-bold text-gray-900 dark:text-white">4. {{ t('admin.baizhi.services.image') }}</h3>
          </div>

          <div class="overflow-x-auto">
            <table class="w-full text-left text-xs">
              <thead class="bg-gray-50 text-gray-700 dark:bg-dark-900/50 dark:text-gray-300 font-semibold">
                <tr>
                  <th class="p-2.5">方法</th>
                  <th class="p-2.5">网关入口路径</th>
                  <th class="p-2.5">功能说明</th>
                  <th class="p-2.5">返回说明</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700 text-gray-800 dark:text-gray-200 font-mono">
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/image/openapi/v1/text_to_image</td>
                  <td class="p-2.5 font-sans">文字提示词异步创建文生图任务</td>
                  <td class="p-2.5 font-sans">立即返回 task_id</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/image/openapi/v1/edit_image</td>
                  <td class="p-2.5 font-sans">基于参考图与提示词创建图生图任务</td>
                  <td class="p-2.5 font-sans">立即返回 task_id</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/image/openapi/v1/uploads/presign</td>
                  <td class="p-2.5 font-sans">获取本地参考图预签名上传凭证 (PUT 对象存储)</td>
                  <td class="p-2.5 font-sans">upload_url / image_url</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-info text-[10px]">GET</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/image/openapi/v1/tasks/:task_id</td>
                  <td class="p-2.5 font-sans">轮询任务状态及获取图片 URL (image_urls)</td>
                  <td class="p-2.5 font-sans">status / image_urls</td>
                </tr>
                <tr>
                  <td class="p-2.5"><span class="badge badge-success text-[10px]">POST</span></td>
                  <td class="p-2.5 font-bold text-gray-900 dark:text-white">/baizhi/image/openapi/v1/tasks/:task_id/retry</td>
                  <td class="p-2.5 font-sans">重试失败或补齐缺失图片任务</td>
                  <td class="p-2.5 font-sans">status: processing</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Code Snippet -->
          <div class="rounded-xl bg-gray-900 p-4 font-mono text-xs text-gray-200 shadow-sm">
            <div class="flex items-center justify-between pb-2 text-gray-400 border-b border-gray-800">
              <span class="font-sans">cURL 示例 (文生图 POST /baizhi/image/openapi/v1/text_to_image)</span>
              <button
                type="button"
                class="px-2.5 py-1 rounded-full border border-gray-700 bg-gray-800 hover:bg-gray-700 text-gray-200 hover:text-white transition-all cursor-pointer"
                @click="copySnippet(imageSnippet)"
              >
                复制
              </button>
            </div>
            <pre class="mt-2 text-pink-400 overflow-x-auto">{{ imageSnippet }}</pre>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Account Key Dialog -->
    <BaseDialog
      :show="showAddKeyDialog"
      :title="t('admin.baizhi.dialog.addKeyTitle')"
      @close="showAddKeyDialog = false"
    >
      <div class="space-y-4">
        <!-- Cookie / Session input with inspect button -->
        <div>
          <label class="block text-xs font-semibold text-gray-700 dark:text-gray-300">
            百智云账号 Cookie / Session
          </label>
          <div class="mt-1.5 flex gap-2">
            <input
              v-model.trim="newKeyForm.cookie"
              type="text"
              class="input input-sm flex-1 font-mono text-xs"
              placeholder="粘贴完整 Cookie 或 baizhi_session=..."
            />
            <button
              type="button"
              class="inline-flex items-center gap-1 px-3 py-1.5 rounded-xl text-xs font-semibold bg-white dark:bg-dark-800 text-gray-800 dark:text-gray-100 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 shadow-xs transition-all cursor-pointer disabled:opacity-50"
              :disabled="inspectingCookie || !newKeyForm.cookie"
              @click="inspectCookie"
            >
              <Icon name="bolt" size="xs" :class="{ 'animate-spin': inspectingCookie }" />
              <span>{{ inspectingCookie ? '识别中...' : '检测账号' }}</span>
            </button>
          </div>
        </div>

        <!-- Inspect Result Panel if detected -->
        <div v-if="inspectResult" class="p-3 rounded-xl bg-gray-50 dark:bg-dark-900/60 border border-gray-200/80 dark:border-dark-700 space-y-2 text-xs">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <img v-if="inspectResult.avatar" :src="inspectResult.avatar" class="w-6 h-6 rounded-full" />
              <span class="font-bold text-gray-900 dark:text-white">{{ inspectResult.user_name }}</span>
              <span class="text-gray-400 text-[11px]">ID: {{ inspectResult.user_id }}</span>
            </div>
            <span v-if="inspectResult.daily_claimed" class="text-emerald-700 dark:text-emerald-400 font-semibold text-[11px]">
              ✓ 已签到
            </span>
            <span v-else class="text-amber-700 dark:text-amber-400 font-semibold text-[11px]">
              未签到
            </span>
          </div>

          <div class="grid grid-cols-3 gap-2 text-center pt-1">
            <div class="p-2 rounded-lg bg-white dark:bg-dark-800 border border-gray-100 dark:border-dark-700">
              <div class="text-[10.5px] text-gray-500 dark:text-gray-400">今日临时积分</div>
              <div class="text-sm font-bold text-emerald-600 dark:text-emerald-400">⚡ {{ inspectResult.temporary_credits }}</div>
            </div>
            <div class="p-2 rounded-lg bg-white dark:bg-dark-800 border border-gray-100 dark:border-dark-700">
              <div class="text-[10.5px] text-gray-500 dark:text-gray-400">永久积分</div>
              <div class="text-sm font-bold text-gray-900 dark:text-gray-100">💰 {{ inspectResult.permanent_credits }}</div>
            </div>
            <div class="p-2 rounded-lg bg-white dark:bg-dark-800 border border-gray-100 dark:border-dark-700">
              <div class="text-[10.5px] text-gray-500 dark:text-gray-400">总可用积分</div>
              <div class="text-sm font-bold text-gray-900 dark:text-gray-100">{{ inspectResult.available_credits }}</div>
            </div>
          </div>

          <!-- Discovered keys list if any -->
          <div v-if="inspectResult.discovered_keys && inspectResult.discovered_keys.length > 0" class="pt-2 border-t border-gray-200/60 dark:border-dark-700">
            <div class="text-[11px] font-semibold text-gray-700 dark:text-gray-300 mb-1.5">
              关联密钥
            </div>
            <div class="flex flex-wrap gap-1.5">
              <button
                v-for="k in inspectResult.discovered_keys"
                :key="k.key"
                type="button"
                class="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-mono bg-white dark:bg-dark-800 text-gray-800 dark:text-gray-100 border border-gray-300 dark:border-dark-600 hover:border-gray-900 dark:hover:border-teal-500 cursor-pointer"
                @click="newKeyForm.api_key = k.key; if (!newKeyForm.account_name) newKeyForm.account_name = k.name || inspectResult.user_name"
              >
                <Icon name="key" size="xs" />
                <span>{{ k.name ? `${k.name}: ` : '' }}{{ k.key.substring(0, 12) }}...</span>
              </button>
            </div>
          </div>
        </div>

        <div>
          <label class="block text-xs font-semibold text-gray-700 dark:text-gray-300">
            {{ t('admin.baizhi.dialog.accountName') }}
          </label>
          <input
            v-model.trim="newKeyForm.account_name"
            type="text"
            class="input input-sm mt-1.5 w-full"
            :placeholder="t('admin.baizhi.dialog.accountNamePlaceholder')"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-gray-700 dark:text-gray-300">
            {{ t('admin.baizhi.dialog.apiKey') }}
          </label>
          <input
            v-model.trim="newKeyForm.api_key"
            type="password"
            class="input input-sm mt-1.5 w-full font-mono"
            :placeholder="t('admin.baizhi.dialog.apiKeyPlaceholder')"
          />
        </div>
      </div>

      <template #footer>
        <button
          type="button"
          class="px-3.5 py-1.5 rounded-xl text-xs font-medium bg-white dark:bg-dark-800 text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-dark-600 hover:bg-gray-50 dark:hover:bg-dark-700 cursor-pointer"
          @click="showAddKeyDialog = false"
        >
          {{ t('admin.baizhi.dialog.cancel') }}
        </button>
        <button
          type="button"
          class="px-4 py-1.5 rounded-xl text-xs font-semibold text-white bg-gray-900 hover:bg-black dark:bg-teal-600 dark:hover:bg-teal-500 cursor-pointer disabled:opacity-50"
          :disabled="!newKeyForm.account_name || !newKeyForm.api_key"
          @click="confirmAddKey"
        >
          {{ t('admin.baizhi.dialog.confirm') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PillPopover, { type PillOption } from '@/components/common/PillPopover.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  getBaizhiConfig,
  updateBaizhiConfig,
  testBaizhiConnection,
  inspectBaizhiAccount,
  syncBaizhiBalances,
  type BaizhiConfig,
  type BaizhiKeyEntry,
  type BaizhiAccountInspectResult
} from '@/api/admin/baizhi'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const activeTab = ref<'config' | 'docs'>('config')
const loading = ref(false)
const saving = ref(false)
const copiedUrl = ref(false)

const strategyOptions: PillOption[] = [
  { value: 'temp_credit_first', label: '临时积分优先', hint: '每日100额度' },
  { value: 'round_robin', label: '轮询均衡', hint: 'Round Robin' },
  { value: 'failover', label: '主备容灾', hint: 'Failover' }
]

const config = reactive<BaizhiConfig>({
  auth_required: true,
  web_scrape: {
    enabled: true,
    strategy: 'round_robin',
    base_url: 'https://web-scrape.app.baizhi.cloud',
    keys: []
  },
  web_search: {
    enabled: true,
    strategy: 'round_robin',
    base_url: 'https://websearch.app.baizhi.cloud',
    keys: []
  },
  voice: {
    enabled: true,
    strategy: 'round_robin',
    base_url: 'https://asrmodel.app.baizhi.cloud/openapi/v1',
    wss_url: 'wss://asrmodel.app.baizhi.cloud/openapi/v1',
    keys: []
  },
  image_generate: {
    enabled: true,
    strategy: 'round_robin',
    base_url: 'https://image-generate.app.baizhi.cloud/openapi/v1',
    keys: []
  }
})

// Diagnostics / Testing states per key ID
interface TestState {
  loading: boolean
  latency?: number
  error?: string
}
const testStates = reactive<Record<string, TestState>>({})

// Dynamic Base URL
const gatewayBaseUrl = computed(() => {
  return `${window.location.origin}/baizhi`
})

const activeServicesCount = computed(() => {
  let count = 0
  if (config.web_scrape.enabled) count++
  if (config.web_search.enabled) count++
  if (config.voice.enabled) count++
  if (config.image_generate.enabled) count++
  return count
})

const totalKeysCount = computed(() => {
  return (
    config.web_scrape.keys.length +
    config.web_search.keys.length +
    config.voice.keys.length +
    config.image_generate.keys.length
  )
})

// Dialog state
const showAddKeyDialog = ref(false)
const targetServiceForAdd = ref<'web_scrape' | 'web_search' | 'voice' | 'image_generate'>('web_scrape')
const inspectingCookie = ref(false)
const inspectResult = ref<BaizhiAccountInspectResult | null>(null)
const syncingBalances = ref(false)

const newKeyForm = reactive({
  account_name: '',
  api_key: '',
  cookie: '',
  temporary_credits: 0,
  permanent_credits: 0,
  available_credits: 0,
  daily_claimed: false,
  user_id: 0,
  user_name: '',
  avatar: ''
})

async function inspectCookie() {
  if (!newKeyForm.cookie) {
    appStore.showError('请先粘贴百智云 Cookie 或 baizhi_session')
    return
  }
  inspectingCookie.value = true
  inspectResult.value = null
  try {
    const res = await inspectBaizhiAccount(newKeyForm.cookie)
    inspectResult.value = res
    if (res.user_name && (!newKeyForm.account_name || newKeyForm.account_name.startsWith('账号'))) {
      newKeyForm.account_name = res.user_name
    }
    newKeyForm.temporary_credits = res.temporary_credits
    newKeyForm.permanent_credits = res.permanent_credits
    newKeyForm.available_credits = res.available_credits
    newKeyForm.daily_claimed = res.daily_claimed
    newKeyForm.user_id = res.user_id
    newKeyForm.user_name = res.user_name
    newKeyForm.avatar = res.avatar
    appStore.showSuccess(`成功识别账号「${res.user_name}」，剩余临时积分 ${res.temporary_credits}`)
  } catch (err: unknown) {
    appStore.showError((err as Error).message || '识别账号失败')
  } finally {
    inspectingCookie.value = false
  }
}

async function syncAllBalances() {
  syncingBalances.value = true
  try {
    const updated = await syncBaizhiBalances()
    if (updated && typeof updated === 'object') {
      Object.assign(config, updated)
      appStore.showSuccess('各账号积分与余额已同步刷新！')
    }
  } catch (err: unknown) {
    appStore.showError((err as Error).message || '同步账号积分失败')
  } finally {
    syncingBalances.value = false
  }
}

async function loadConfig() {
  loading.value = true
  try {
    const data = await getBaizhiConfig()
    if (data && typeof data === 'object') {
      Object.assign(config, data)
    }
  } catch (err: unknown) {
    appStore.showError((err as Error).message || t('admin.baizhi.overview.saveFailed'))
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  try {
    await updateBaizhiConfig(config)
    appStore.showSuccess(t('admin.baizhi.overview.saveSuccess'))
  } catch (err: unknown) {
    appStore.showError((err as Error).message || t('admin.baizhi.overview.saveFailed'))
  } finally {
    saving.value = false
  }
}

function openAddKeyDialog(serviceType: 'web_scrape' | 'web_search' | 'voice' | 'image_generate') {
  targetServiceForAdd.value = serviceType
  newKeyForm.account_name = ''
  newKeyForm.api_key = ''
  newKeyForm.cookie = ''
  newKeyForm.temporary_credits = 0
  newKeyForm.permanent_credits = 0
  newKeyForm.available_credits = 0
  newKeyForm.daily_claimed = false
  newKeyForm.user_id = 0
  newKeyForm.user_name = ''
  newKeyForm.avatar = ''
  inspectResult.value = null
  showAddKeyDialog.value = true
}

function confirmAddKey() {
  if (!newKeyForm.account_name || !newKeyForm.api_key) return

  const newKey: BaizhiKeyEntry = {
    id: `key_${Date.now()}_${Math.random().toString(36).substring(2, 7)}`,
    account_name: newKeyForm.account_name,
    api_key: newKeyForm.api_key,
    cookie: newKeyForm.cookie || undefined,
    temporary_credits: newKeyForm.temporary_credits,
    permanent_credits: newKeyForm.permanent_credits,
    available_credits: newKeyForm.available_credits,
    daily_claimed: newKeyForm.daily_claimed,
    user_id: newKeyForm.user_id || undefined,
    user_name: newKeyForm.user_name || undefined,
    avatar: newKeyForm.avatar || undefined,
    last_sync_at: newKeyForm.temporary_credits !== undefined ? new Date().toISOString() : undefined,
    enabled: true,
    weight: 1
  }

  config[targetServiceForAdd.value].keys.push(newKey)
  showAddKeyDialog.value = false
  appStore.showSuccess(`已添加「${newKey.account_name}」密钥，别忘了点击右上角保存配置！`)
}

function removeKey(serviceType: 'web_scrape' | 'web_search' | 'voice' | 'image_generate', idx: number) {
  config[serviceType].keys.splice(idx, 1)
}

async function runTest(serviceType: 'web_scrape' | 'web_search' | 'voice' | 'image_generate', keyItem: BaizhiKeyEntry) {
  testStates[keyItem.id] = { loading: true }
  try {
    const res = await testBaizhiConnection({
      service_type: serviceType,
      api_key: keyItem.api_key,
      key_id: keyItem.id
    })

    if (res?.success) {
      testStates[keyItem.id] = { loading: false, latency: res.latency_ms }
      appStore.showSuccess(`「${keyItem.account_name}」测试通过，延迟 ${res.latency_ms}ms`)
    } else {
      const errMsg = res?.message || 'Error'
      testStates[keyItem.id] = { loading: false, error: errMsg }
      appStore.showError(`「${keyItem.account_name}」测试失败: ${errMsg}`)
    }
  } catch (err: unknown) {
    testStates[keyItem.id] = { loading: false, error: 'Failed' }
    appStore.showError((err as Error).message || 'Test failed')
  }
}

async function copyGatewayUrl() {
  await copyToClipboard(gatewayBaseUrl.value, t('admin.baizhi.overview.copySuccess'))
  copiedUrl.value = true
  setTimeout(() => {
    copiedUrl.value = false
  }, 2000)
}

async function copySnippet(text: string) {
  await copyToClipboard(text, t('admin.baizhi.docs.copied'))
}

// Documentation code snippets
const scrapeSnippet = computed(() => {
  return `curl -X POST "${gatewayBaseUrl.value}/scrape/openapi/scrape" \\
  -H "Authorization: Bearer YOUR_SUB2API_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "url": "https://example.com",
    "return_format": "json"
  }'`
})

const searchSnippet = computed(() => {
  return `curl -X POST "${gatewayBaseUrl.value}/search/openapi/search" \\
  -H "Authorization: Bearer YOUR_SUB2API_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "query": "Claude 3.7 Sonnet 最新评测",
    "count": 10,
    "time_range": "month",
    "need_summary": true
  }'`
})

const voiceSnippet = computed(() => {
  return `curl -X POST "${gatewayBaseUrl.value}/voice/openapi/v1/audio" \\
  -H "Authorization: Bearer YOUR_SUB2API_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "task": "text-to-speech",
    "input": { "text": "你好，欢迎使用百智云智能语音服务！" },
    "parameters": { "voice": "vivi", "speed": "normal" }
  }' \\
  --output speech.wav`
})

const imageSnippet = computed(() => {
  return `curl -X POST "${gatewayBaseUrl.value}/image/openapi/v1/text_to_image" \\
  -H "Authorization: Bearer YOUR_SUB2API_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "prompt": "一只可爱的布偶猫，高精写实，工作室光影",
    "image_count": 1,
    "ratio": "16:9",
    "resolution": "2K"
  }'`
})

onMounted(() => {
  loadConfig()
})
</script>
