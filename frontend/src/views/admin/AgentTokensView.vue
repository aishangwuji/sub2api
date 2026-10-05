<template>
  <AppLayout>
    <TablePageLayout>
      <!-- Filters and Action Toolbar -->
      <template #filters>
        <div class="card p-4 sm:p-6 mb-4">
          <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
            <!-- Left: Search and Filters -->
            <div class="flex flex-1 flex-wrap items-center gap-3">
              <div class="w-full sm:w-72">
                <div class="relative">
                  <Icon
                    name="search"
                    size="md"
                    class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
                  />
                  <input
                    v-model.trim="searchQuery"
                    type="text"
                    class="input pl-10"
                    :placeholder="t('admin.agentTokens.searchPlaceholder')"
                    @keyup.enter="handleSearch"
                  />
                </div>
              </div>

              <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300 cursor-pointer select-none">
                <input
                  v-model="includeRevoked"
                  type="checkbox"
                  class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800"
                  @change="loadTokens"
                />
                {{ t('admin.agentTokens.includeRevoked') }}
              </label>
            </div>

            <!-- Right: Actions -->
            <div class="flex flex-wrap items-center gap-2">
              <button
                type="button"
                class="btn btn-secondary"
                :disabled="loading"
                :title="t('common.refresh')"
                @click="loadTokens"
              >
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>

              <button
                type="button"
                class="btn btn-secondary"
                @click="openSkillDocDialog"
              >
                <Icon name="document" size="md" class="mr-1.5" />
                {{ t('admin.agentTokens.skillDoc') }}
              </button>

              <button
                type="button"
                class="btn btn-primary"
                @click="openCreateDialog"
              >
                <Icon name="plus" size="md" class="mr-1.5" />
                {{ t('admin.agentTokens.createToken') }}
              </button>
            </div>
          </div>

          <!-- Security Callout -->
          <div class="mt-4 flex items-start gap-3 rounded-xl bg-blue-50/80 p-3.5 text-xs leading-relaxed text-blue-900 ring-1 ring-blue-200/70 dark:bg-blue-950/30 dark:text-blue-200 dark:ring-blue-900/50">
            <Icon name="shield" size="md" class="mt-0.5 flex-shrink-0 text-blue-600 dark:text-blue-400" />
            <div>
              <span class="font-bold">{{ t('admin.agentTokens.securityNoticeTitle') }}：</span>
              <span>{{ t('admin.agentTokens.securityNoticeText') }}</span>
            </div>
          </div>
        </div>
      </template>

      <!-- Table View -->
      <template #table>
        <DataTable
          :columns="columns"
          :data="filteredTokens"
          :loading="loading"
          row-key="id"
        >
          <!-- Token Name & Description -->
          <template #cell-name="{ row }">
            <div class="min-w-0 max-w-[220px]">
              <div class="truncate font-medium text-gray-900 dark:text-white" :title="row.name">
                {{ row.name }}
              </div>
              <div v-if="row.description" class="mt-0.5 truncate text-xs text-gray-400" :title="row.description">
                {{ row.description }}
              </div>
            </div>
          </template>

          <!-- Prefix & copy -->
          <template #cell-token_prefix="{ value }">
            <div class="flex items-center gap-1.5">
              <code class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-200">
                {{ value }}...
              </code>
              <button
                type="button"
                class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
                :title="t('keys.copyToClipboard')"
                @click="copyText(value)"
              >
                <Icon name="copy" size="xs" />
              </button>
            </div>
          </template>

          <!-- Scopes Badges -->
          <template #cell-scopes="{ value }">
            <div class="flex flex-wrap gap-1 max-w-xs">
              <span
                v-for="scope in value"
                :key="scope"
                :class="[
                  'inline-flex items-center rounded px-1.5 py-0.5 text-[11px] font-mono font-medium',
                  getScopeClass(scope)
                ]"
              >
                {{ scope }}
              </span>
            </div>
          </template>

          <!-- Status Badge -->
          <template #cell-status="{ row }">
            <span
              :class="[
                'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                getTokenStatusClass(row)
              ]"
            >
              {{ getTokenStatusLabel(row) }}
            </span>
          </template>

          <!-- Created At -->
          <template #cell-created_at="{ value }">
            <span class="whitespace-nowrap text-xs text-gray-600 dark:text-gray-400">
              {{ formatDateTime(value) }}
            </span>
          </template>

          <!-- Expires At -->
          <template #cell-expires_at="{ value, row }">
            <span
              :class="[
                'whitespace-nowrap text-xs',
                isExpired(row) ? 'text-red-500 font-medium' : 'text-gray-600 dark:text-gray-400'
              ]"
            >
              {{ value ? formatDateTime(value) : t('admin.agentTokens.createDialog.expiresOptions.never') }}
            </span>
          </template>

          <!-- Last Used -->
          <template #cell-last_used_at="{ value }">
            <span class="whitespace-nowrap text-xs text-gray-500 dark:text-gray-400">
              {{ value ? formatDateTime(value) : '—' }}
            </span>
          </template>

          <!-- Actions -->
          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end gap-2">
              <button
                v-if="!row.revoked_at"
                type="button"
                class="btn btn-sm btn-danger"
                @click="promptRevoke(row)"
              >
                {{ t('admin.agentTokens.revokeConfirm.confirm') }}
              </button>
              <span v-else class="text-xs text-gray-400 italic">
                {{ t('admin.agentTokens.status.revoked') }}
              </span>
            </div>
          </template>
        </DataTable>

        <Pagination
          v-if="total > pageSize"
          :page="page"
          :total="total"
          :page-size="pageSize"
          class="mt-4"
          @update:page="handlePageChange"
        />
      </template>
    </TablePageLayout>

    <!-- Create Token Dialog -->
    <BaseDialog
      :show="showCreateModal"
      :title="t('admin.agentTokens.createDialog.title')"
      width="normal"
      @close="closeCreateDialog"
    >
      <form class="space-y-4 py-1" @submit.prevent="submitCreateToken">
        <div>
          <label class="input-label required">{{ t('admin.agentTokens.createDialog.name') }}</label>
          <input
            v-model.trim="form.name"
            type="text"
            required
            class="input"
            :placeholder="t('admin.agentTokens.createDialog.namePlaceholder')"
          />
        </div>

        <div>
          <label class="input-label">{{ t('admin.agentTokens.createDialog.expiresIn') }}</label>
          <Select
            v-model="form.expiresInDays"
            :options="expirationOptions"
          />
        </div>

        <div>
          <div class="flex items-center justify-between mb-1.5">
            <label class="input-label mb-0 required">{{ t('admin.agentTokens.createDialog.scopes') }}</label>
            <div class="flex items-center gap-2 text-xs">
              <button
                type="button"
                class="text-primary-600 hover:text-primary-700 dark:text-primary-400 font-medium"
                @click="handleSelectAllScopes"
              >
                {{ t('admin.agentTokens.createDialog.selectAll') }}
              </button>
              <span class="text-gray-300 dark:text-dark-600">|</span>
              <button
                type="button"
                class="text-primary-600 hover:text-primary-700 dark:text-primary-400 font-medium"
                @click="handleSelectReadonlyScopes"
              >
                {{ t('admin.agentTokens.createDialog.selectReadonly') }}
              </button>
              <span class="text-gray-300 dark:text-dark-600">|</span>
              <button
                type="button"
                class="text-gray-500 hover:text-gray-700 dark:text-gray-400"
                @click="handleClearScopes"
              >
                {{ t('admin.agentTokens.createDialog.clearAll') }}
              </button>
            </div>
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
            {{ t('admin.agentTokens.createDialog.scopesHint') }}
          </p>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 max-h-64 overflow-y-auto p-1">
            <label
              v-for="item in availableScopes"
              :key="item.scope"
              :class="[
                'flex items-start gap-2.5 p-2.5 rounded-lg border text-left cursor-pointer transition-colors',
                form.scopes.includes(item.scope)
                  ? 'border-primary-500 bg-primary-50/50 dark:border-primary-500/80 dark:bg-primary-950/20'
                  : 'border-gray-200 hover:border-gray-300 dark:border-dark-700 dark:hover:border-dark-600 bg-white dark:bg-dark-800'
              ]"
            >
              <input
                type="checkbox"
                :value="item.scope"
                :checked="form.scopes.includes(item.scope)"
                class="mt-0.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800"
                @change="toggleScope(item.scope)"
              />
              <div class="min-w-0 flex-1">
                <div class="font-mono text-xs font-bold text-gray-900 dark:text-gray-100">
                  {{ item.scope }}
                </div>
                <div class="text-[11px] text-gray-500 dark:text-gray-400 mt-0.5 leading-snug">
                  {{ item.description }}
                </div>
              </div>
            </label>
          </div>
        </div>
      </form>

      <template #footer>
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="submitting"
          @click="closeCreateDialog"
        >
          {{ t('admin.agentTokens.createDialog.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="submitting || !form.name || form.scopes.length === 0"
          @click="submitCreateToken"
        >
          <Icon v-if="submitting" name="refresh" size="sm" class="animate-spin mr-1.5" />
          {{ t('admin.agentTokens.createDialog.submit') }}
        </button>
      </template>
    </BaseDialog>

    <!-- Token Created Success Dialog (Shown ONCE with plaintext) -->
    <BaseDialog
      :show="showCreatedResultModal"
      :title="t('admin.agentTokens.createdDialog.title')"
      width="normal"
      @close="closeCreatedResultModal"
    >
      <div class="py-2 space-y-4">
        <div class="rounded-xl bg-amber-50 p-4 text-amber-800 ring-1 ring-amber-300 dark:bg-amber-950/40 dark:text-amber-200 dark:ring-amber-900/60">
          <div class="flex items-start gap-3">
            <Icon name="exclamationTriangle" size="lg" class="flex-shrink-0 text-amber-600 dark:text-amber-400 mt-0.5" />
            <div class="text-xs leading-relaxed font-medium">
              {{ t('admin.agentTokens.createdDialog.warning') }}
            </div>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.agentTokens.createdDialog.tokenLabel') }}</label>
          <div class="flex items-center gap-2">
            <input
              type="text"
              readonly
              :value="createdTokenResult?.raw_token"
              class="input font-mono text-xs bg-gray-50 select-all dark:bg-dark-900 font-semibold text-primary-700 dark:text-primary-300"
              @focus="($event.target as HTMLInputElement).select()"
            />
            <button
              type="button"
              class="btn btn-primary flex-shrink-0"
              @click="copyText(createdTokenResult?.raw_token || '')"
            >
              <Icon name="copy" size="sm" class="mr-1" />
              {{ t('keys.copyToClipboard') }}
            </button>
          </div>
        </div>

        <div class="rounded-lg bg-gray-50 p-3 text-xs text-gray-600 dark:bg-dark-900 dark:text-gray-400 space-y-1">
          <div><span class="font-medium text-gray-900 dark:text-white">{{ t('admin.agentTokens.columns.name') }}:</span> {{ createdTokenResult?.name }}</div>
          <div><span class="font-medium text-gray-900 dark:text-white">{{ t('admin.agentTokens.columns.prefix') }}:</span> {{ createdTokenResult?.token_prefix }}...</div>
          <div><span class="font-medium text-gray-900 dark:text-white">{{ t('admin.agentTokens.columns.scopes') }}:</span> {{ createdTokenResult?.scopes?.join(', ') }}</div>
        </div>
      </div>

      <template #footer>
        <button
          type="button"
          class="btn btn-primary w-full sm:w-auto"
          @click="closeCreatedResultModal"
        >
          {{ t('admin.agentTokens.createdDialog.close') }}
        </button>
      </template>
    </BaseDialog>

    <!-- Revoke Confirmation Dialog -->
    <ConfirmDialog
      :show="showRevokeModal"
      :title="t('admin.agentTokens.revokeConfirm.title')"
      :message="t('admin.agentTokens.revokeConfirm.message', { name: tokenToRevoke?.name || '' })"
      :confirm-text="t('admin.agentTokens.revokeConfirm.confirm')"
      :cancel-text="t('admin.agentTokens.revokeConfirm.cancel')"
      :is-danger="true"
      @confirm="submitRevoke"
      @cancel="showRevokeModal = false"
    />

    <!-- Internal Agent Skill Spec Dialog -->
    <BaseDialog
      :show="showSkillDocModal"
      :title="t('admin.agentTokens.skillDocDialog.title')"
      width="wide"
      @close="showSkillDocModal = false"
    >
      <div class="py-1 space-y-3">
        <p class="text-xs text-gray-600 dark:text-gray-400">
          {{ t('admin.agentTokens.skillDocDialog.subtitle') }}
        </p>

        <div v-if="loadingSkillDoc" class="flex items-center justify-center py-16">
          <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
          <span class="ml-2 text-sm text-gray-500">{{ t('admin.agentTokens.skillDocDialog.loading') }}</span>
        </div>

        <div v-else class="relative">
          <div class="absolute top-2 right-2 z-10">
            <button
              type="button"
              class="btn btn-sm btn-secondary shadow-sm"
              @click="copyText(skillDocContent)"
            >
              <Icon name="copy" size="xs" class="mr-1" />
              {{ t('admin.agentTokens.skillDocDialog.copyDoc') }}
            </button>
          </div>
          <pre class="max-h-[500px] overflow-auto rounded-xl bg-gray-900 p-4 font-mono text-xs text-gray-100 leading-relaxed scrollbar-thin">{{ skillDocContent }}</pre>
        </div>
      </div>

      <template #footer>
        <button
          type="button"
          class="btn btn-secondary"
          @click="showSkillDocModal = false"
        >
          {{ t('admin.agentTokens.skillDocDialog.close') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useClipboard } from '@/composables/useClipboard'
import {
  adminAPI,
} from '@/api/admin'
import type {
  AgentToken,
  CreateAgentTokenResponse,
  AgentScopeItem
} from '@/api/admin/agentTokens'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

// State
const loading = ref(false)
const tokens = ref<AgentToken[]>([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const searchQuery = ref('')
const includeRevoked = ref(true)

// Scopes dictionary
const availableScopes = ref<AgentScopeItem[]>([])

// Create Dialog State
const showCreateModal = ref(false)
const submitting = ref(false)
const form = reactive({
  name: '',
  expiresInDays: 90,
  scopes: [] as string[]
})

// Created Result State
const showCreatedResultModal = ref(false)
const createdTokenResult = ref<CreateAgentTokenResponse | null>(null)

// Revoke State
const showRevokeModal = ref(false)
const tokenToRevoke = ref<AgentToken | null>(null)

// Skill Doc State
const showSkillDocModal = ref(false)
const loadingSkillDoc = ref(false)
const skillDocContent = ref('')

const expirationOptions = computed(() => [
  { value: 30, label: t('admin.agentTokens.createDialog.expiresOptions.days30') },
  { value: 90, label: t('admin.agentTokens.createDialog.expiresOptions.days90') },
  { value: 180, label: t('admin.agentTokens.createDialog.expiresOptions.days180') },
  { value: 365, label: t('admin.agentTokens.createDialog.expiresOptions.days365') },
  { value: 0, label: t('admin.agentTokens.createDialog.expiresOptions.never') }
])

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.agentTokens.columns.name') },
  { key: 'token_prefix', label: t('admin.agentTokens.columns.prefix') },
  { key: 'scopes', label: t('admin.agentTokens.columns.scopes') },
  { key: 'status', label: t('admin.agentTokens.columns.status') },
  { key: 'created_at', label: t('admin.agentTokens.columns.createdAt') },
  { key: 'expires_at', label: t('admin.agentTokens.columns.expiresAt') },
  { key: 'last_used_at', label: t('admin.agentTokens.columns.lastUsedAt') },
  { key: 'actions', label: t('admin.agentTokens.columns.actions'), align: 'right' }
])

const filteredTokens = computed(() => {
  if (!searchQuery.value) return tokens.value
  const q = searchQuery.value.toLowerCase()
  return tokens.value.filter(
    (tk: AgentToken) => tk.name.toLowerCase().includes(q) || tk.token_prefix.toLowerCase().includes(q)
  )
})

function formatDateTime(isoString?: string): string {
  if (!isoString) return '—'
  const date = new Date(isoString)
  if (isNaN(date.getTime())) return isoString
  return date.toLocaleString()
}

function isExpired(token: AgentToken): boolean {
  if (!token.expires_at) return false
  return new Date(token.expires_at).getTime() < Date.now()
}

function getTokenStatusClass(token: AgentToken): string {
  if (token.revoked_at) {
    return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'
  }
  if (isExpired(token)) {
    return 'bg-red-100 text-red-700 dark:bg-red-950/40 dark:text-red-400'
  }
  return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-400'
}

function getTokenStatusLabel(token: AgentToken): string {
  if (token.revoked_at) return t('admin.agentTokens.status.revoked')
  if (isExpired(token)) return t('admin.agentTokens.status.expired')
  return t('admin.agentTokens.status.active')
}

function getScopeClass(scope: string): string {
  if (scope.endsWith(':write') || scope === 'ops:trigger') {
    return 'bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300'
  }
  return 'bg-blue-100 text-blue-800 dark:bg-blue-950/40 dark:text-blue-300'
}

async function copyText(text: string) {
  if (!text) return
  const success = await copyToClipboard(text)
  if (success) {
    appStore.showSuccess(t('admin.agentTokens.createdDialog.copySuccess'))
  }
}

async function loadTokens() {
  loading.value = true
  try {
    const res = await adminAPI.agentTokens.listAgentTokens({
      page: page.value,
      page_size: pageSize.value,
      include_revoked: includeRevoked.value
    })
    tokens.value = res.items || []
    total.value = res.total || 0
  } catch (err: any) {
    appStore.showError(err.message || 'Failed to load tokens')
  } finally {
    loading.value = false
  }
}

async function loadScopes() {
  try {
    const res = await adminAPI.agentTokens.getAgentScopes()
    availableScopes.value = res.data || []
  } catch {
    availableScopes.value = [
      { scope: 'status:read', description: 'Read system overview & health metrics' },
      { scope: 'accounts:read', description: 'List upstream accounts and statuses' },
      { scope: 'accounts:write', description: 'Update status of upstream accounts' },
      { scope: 'proxies:read', description: 'List proxy nodes and latency' },
      { scope: 'proxies:write', description: 'Update proxy nodes statuses' },
      { scope: 'users:read', description: 'Read user list and basic statistics' },
      { scope: 'redemptions:write', description: 'Batch generate redemption codes' },
      { scope: 'ops:trigger', description: 'Trigger batch sync and cache flushes' }
    ]
  }
}

function handleSearch() {
  // filteredTokens computed handles client-side filtering
}

function handlePageChange(newPage: number) {
  page.value = newPage
  loadTokens()
}

function openCreateDialog() {
  form.name = ''
  form.expiresInDays = 90
  form.scopes = ['status:read', 'accounts:read']
  showCreateModal.value = true
}

function closeCreateDialog() {
  showCreateModal.value = false
}

function toggleScope(scope: string) {
  const index = form.scopes.indexOf(scope)
  if (index > -1) {
    form.scopes.splice(index, 1)
  } else {
    form.scopes.push(scope)
  }
}

function handleSelectAllScopes() {
  form.scopes = availableScopes.value.map((s: AgentScopeItem) => s.scope)
}

function handleSelectReadonlyScopes() {
  form.scopes = availableScopes.value
    .filter((s: AgentScopeItem) => s.scope.endsWith(':read'))
    .map((s: AgentScopeItem) => s.scope)
}

function handleClearScopes() {
  form.scopes = []
}

async function submitCreateToken() {
  if (!form.name || form.scopes.length === 0) return
  submitting.value = true
  try {
    const res = await adminAPI.agentTokens.createAgentToken({
      name: form.name,
      scopes: form.scopes,
      expires_in_days: form.expiresInDays > 0 ? form.expiresInDays : undefined
    })
    closeCreateDialog()
    createdTokenResult.value = res.data
    showCreatedResultModal.value = true
    loadTokens()
  } catch (err: any) {
    appStore.showError(err.message || 'Failed to create agent token')
  } finally {
    submitting.value = false
  }
}

function closeCreatedResultModal() {
  showCreatedResultModal.value = false
  createdTokenResult.value = null
}

function promptRevoke(token: AgentToken) {
  tokenToRevoke.value = token
  showRevokeModal.value = true
}

async function submitRevoke() {
  if (!tokenToRevoke.value) return
  try {
    await adminAPI.agentTokens.revokeAgentToken(tokenToRevoke.value.id)
    appStore.showSuccess(t('admin.agentTokens.revokeConfirm.success'))
    showRevokeModal.value = false
    tokenToRevoke.value = null
    loadTokens()
  } catch (err: any) {
    appStore.showError(err.message || 'Failed to revoke token')
  }
}

async function openSkillDocDialog() {
  showSkillDocModal.value = true
  if (skillDocContent.value) return
  loadingSkillDoc.value = true
  try {
    const res = await adminAPI.agentTokens.getAgentSkillDoc()
    skillDocContent.value = res.data?.content || ''
  } catch (err: any) {
    appStore.showError(err.message || 'Failed to load skill document')
  } finally {
    loadingSkillDoc.value = false
  }
}

onMounted(() => {
  loadTokens()
  loadScopes()
})
</script>
