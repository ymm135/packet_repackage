<template>
  <div class="logs-page">
    <el-card class="section-card">
      <template #header>
        <div class="card-header">
          <span>Processing Logs</span>
          <div>
            <el-button @click="loadLogs" :icon="Refresh">Refresh</el-button>
            <el-button @click="clearLogs" type="danger">Clear Logs</el-button>
          </div>
        </div>
      </template>

      <!-- Filters -->
      <el-form :inline="true" :model="filters" class="filter-form">
        <el-form-item label="Rule">
          <el-select v-model="filters.rule_id" clearable placeholder="All Rules" style="width: 200px">
            <el-option 
              v-for="rule in rules" 
              :key="rule.ID" 
              :label="rule.name" 
              :value="rule.ID"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="Result">
          <el-select v-model="filters.result" clearable placeholder="All Results" style="width: 150px">
            <el-option label="Success" value="success" />
            <el-option label="Error" value="error" />
            <el-option label="Dropped" value="dropped" />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="loadLogs">Filter</el-button>
        </el-form-item>
      </el-form>

      <!-- Logs Table -->
      <el-table :data="logs" style="width: 100%">
        <el-table-column prop="processed_at" label="Time" width="180">
          <template #default="{ row }">
            {{ formatTime(row.processed_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="rule_name" label="Rule" width="200" />
        <el-table-column prop="result" label="Result" width="100">
          <template #default="{ row }">
            <el-tag :type="getResultType(row.result)">
              {{ row.result }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="original_packet" label="Original Packet" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.original_packet.substring(0, 60) }}...
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="120">
          <template #default="{ row }">
            <el-button size="small" @click="showLogDetail(row)">Details</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.page_size"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadLogs"
          @size-change="loadLogs"
        />
      </div>
    </el-card>

    <!-- Log Detail Dialog -->
    <el-dialog v-model="detailVisible" title="Log Details" width="900px">
      <div v-if="selectedLog">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Time">{{ formatTime(selectedLog.processed_at) }}</el-descriptions-item>
          <el-descriptions-item label="Rule">{{ selectedLog.rule_name }}</el-descriptions-item>
          <el-descriptions-item label="Result">
            <el-tag :type="getResultType(selectedLog.result)">{{ selectedLog.result }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Error" v-if="selectedLog.error_message">
            {{ selectedLog.error_message }}
          </el-descriptions-item>
        </el-descriptions>

        <div class="detail-section">
          <h4>Field Values:</h4>
          <el-table v-if="selectedLog.field_values" :data="formatFieldValues(selectedLog.field_values)" border>
            <el-table-column prop="field" label="Field" width="200" />
            <el-table-column prop="before" label="Before" />
            <el-table-column prop="after" label="After">
              <template #default="{ row }">
                <span :style="{ color: row.before !== row.after ? '#67c23a' : 'inherit' }">
                  {{ row.after }}
                </span>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="detail-section">
          <h4>Original Packet:</h4>
          <hex-viewer :hex="selectedLog.original_packet" />
        </div>

        <div v-if="selectedLog.modified_packet" class="detail-section">
          <h4>Modified Packet:</h4>
          <hex-viewer :hex="selectedLog.modified_packet" />
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { logAPI, ruleAPI } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'
import HexViewer from '@/components/HexViewer.vue'

const logs = ref([])
const rules = ref([])
const detailVisible = ref(false)
const selectedLog = ref(null)

const filters = ref({
  rule_id: '',
  result: ''
})

const pagination = ref({
  page: 1,
  page_size: 20,
  total: 0
})

const loadLogs = async () => {
  try {
    const params = {
      page: pagination.value.page,
      page_size: pagination.value.page_size
    }
    
    if (filters.value.rule_id) {
      params.rule_id = filters.value.rule_id
    }
    if (filters.value.result) {
      params.result = filters.value.result
    }

    const response = await logAPI.list(params)
    logs.value = response.data.data || []
    
    if (response.data.pagination) {
      pagination.value.total = response.data.pagination.total
    }
  } catch (error) {
    ElMessage.error('Failed to load logs')
  }
}

const loadRules = async () => {
  try {
    const response = await ruleAPI.list()
    rules.value = response.data.data || []
  } catch (error) {
    ElMessage.error('Failed to load rules')
  }
}

const clearLogs = async () => {
  try {
    await ElMessageBox.confirm('Clear all logs?', 'Warning', { type: 'warning' })
    await logAPI.clear()
    ElMessage.success('Logs cleared')
    loadLogs()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to clear logs')
    }
  }
}

const showLogDetail = (log) => {
  selectedLog.value = log
  detailVisible.value = true
}

const formatTime = (timeStr) => {
  const date = new Date(timeStr)
  return date.toLocaleString()
}

const getResultType = (result) => {
  const typeMap = {
    success: 'success',
    error: 'danger',
    dropped: 'warning'
  }
  return typeMap[result] || 'info'
}

const formatFieldValues = (fieldValuesJSON) => {
  try {
    const fields = JSON.parse(fieldValuesJSON)
    return Object.entries(fields).map(([field, values]) => ({
      field,
      before: JSON.stringify(values.before),
      after: JSON.stringify(values.after)
    }))
  } catch {
    return []
  }
}

onMounted(() => {
  loadLogs()
  loadRules()
})
</script>

<style scoped>
.logs-page {
  max-width: 1400px;
  margin: 0 auto;
}

.section-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.filter-form {
  margin-bottom: 20px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.detail-section {
  margin-top: 20px;
}

.detail-section h4 {
  color: #303133;
  margin-bottom: 10px;
  font-size: 16px;
}
</style>
