<template>
  <div class="nft-rules-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>NFTables Firewall Rules</span>
          <div>
            <el-button type="success" @click="applyRules" :loading="applying">
              <el-icon><CircleCheck /></el-icon>
              Apply Rules
            </el-button>
            <el-button type="primary" @click="showAddDialog">
              <el-icon><Plus /></el-icon>
              Add Rule
            </el-button>
          </div>
        </div>
      </template

>

      <el-table :data="rules" border stripe style="width: 100%">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column prop="priority" label="Priority" width="80" sortable />
        <el-table-column prop="name" label="Name" width="150" />
        <el-table-column prop="summary" label="Filter (5-Tuple)" min-width="250" />
        <el-table-column prop="action" label="Action" width="100">
          <template #default="{ row }">
            <el-tag :type="getActionType(row.action)">{{ row.action.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="log_enabled" label="Logging" width="80" align="center">
          <template #default="{ row }">
            <el-icon v-if="row.log_enabled" color="#409EFF"><DocumentCopy /></el-icon>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="enabled" label="Enabled" width="90" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="toggleRule(row)" />
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="120" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="editRule(row)">Edit</el-button>
            <el-popconfirm title="Delete this rule?" @confirm="deleteRule(row.ID)">
              <template #reference>
                <el-button size="small" type="danger">Delete</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Add/Edit Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? 'Edit Rule' : 'Add Rule'"
      width="600px"
    >
      <el-form :model="form" label-width="120px">
        <el-form-item label="Rule Name">
          <el-input v-model="form.name" placeholder="e.g., Allow-HTTP" />
        </el-form-item>

        <el-form-item label="Priority">
          <el-input-number v-model="form.priority" :min="1" :max="999" />
          <div class="form-hint">Lower number = higher priority</div>
        </el-form-item>

        <el-divider content-position="left">5-Tuple Filtering</el-divider>

        <el-form-item label="Protocol">
          <el-select v-model="form.protocol" placeholder="Any">
            <el-option label="Any" value="" />
            <el-option label="TCP" value="tcp" />
            <el-option label="UDP" value="udp" />
            <el-option label="ICMP" value="icmp" />
          </el-select>
        </el-form-item>

        <el-form-item label="Source IP">
          <el-input v-model="form.src_ip" placeholder="any (e.g., 192.168.1.0/24)" />
        </el-form-item>

        <el-form-item label="Source Port">
          <el-input v-model="form.src_port" placeholder="any (e.g., 80 or 1024-65535)" :disabled="!form.protocol || form.protocol === 'icmp'" />
        </el-form-item>

        <el-form-item label="Dest IP">
          <el-input v-model="form.dst_ip" placeholder="any (e.g., 10.0.0.1)" />
        </el-form-item>

        <el-form-item label="Dest Port">
          <el-input v-model="form.dst_port" placeholder="any (e.g., 443)" :disabled="!form.protocol || form.protocol === 'icmp'" />
        </el-form-item>

        <el-divider content-position="left">Action & Logging</el-divider>

        <el-form-item label="Action">
          <el-radio-group v-model="form.action">
            <el-radio label="accept">Accept</el-radio>
            <el-radio label="drop">Drop</el-radio>
            <el-radio label="queue">Queue</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="Queue Number" v-if="form.action === 'queue'">
          <el-input v-model="form.queue_num" placeholder="e.g., 0 or 0-3" style="width: 200px" />
          <div class="form-hint">Single queue (0-15) or range (0-3). CPU cores: {{ cpuCores }}</div>
        </el-form-item>

        <el-form-item label="Enable Logging">
          <el-switch v-model="form.log_enabled" />
        </el-form-item>

        <el-form-item label="Log Prefix" v-if="form.log_enabled">
          <el-input v-model="form.log_prefix" placeholder="Leave empty to use rule name" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="saveRule" :loading="saving">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, CircleCheck, DocumentCopy } from '@element-plus/icons-vue'
import { nftRuleAPI } from '@/api'

const rules = ref([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const applying = ref(false)
const cpuCores = ref(navigator.hardwareConcurrency || 4)

const form = ref({
  name: '',
  priority: 100,
  src_ip: '',
  dst_ip: '',
  src_port: '',
  dst_port: '',
  protocol: '',
  log_enabled: false,
  log_prefix: '',
  action: 'accept',
  queue_num: '0',
  enabled: true
})

const loadRules = async () => {
  try {
    const response = await nftRuleAPI.list()
    rules.value = response.data.data || []
  } catch (error) {
    ElMessage.error('Failed to load rules: ' + error.message)
  }
}

const showAddDialog = () => {
  isEdit.value = false
  form.value = {
    name: '',
    priority: 100,
    src_ip: '',
    dst_ip: '',
    src_port: '',
    dst_port: '',
    protocol: '',
    log_enabled: false,
    log_prefix: '',
    action: 'accept',
    queue_num: '0',
    enabled: true
  }
  dialogVisible.value = true
}

const editRule = (rule) => {
  isEdit.value = true
  form.value = { ...rule }
  dialogVisible.value = true
}

const saveRule = async () => {
  if (!form.value.name) {
    ElMessage.warning('Please enter a rule name')
    return
  }

  saving.value = true
  try {
    if (isEdit.value) {
      await nftRuleAPI.update(form.value.ID, form.value)
      ElMessage.success('Rule updated successfully')
    } else {
      await nftRuleAPI.create(form.value)
      ElMessage.success('Rule created successfully')
    }
    dialogVisible.value = false
    loadRules()
  } catch (error) {
    ElMessage.error('Failed to save rule: ' + error.message)
  } finally {
    saving.value = false
  }
}

const deleteRule = async (id) => {
  try {
    await nftRuleAPI.delete(id)
    ElMessage.success('Rule deleted successfully')
    loadRules()
  } catch (error) {
    ElMessage.error('Failed to delete rule: ' + error.message)
  }
}

const toggleRule = async (rule) => {
  try {
    await nftRuleAPI.toggle(rule.ID)
    // No need to reload, switch already updated
  } catch (error) {
    ElMessage.error('Failed to toggle rule: ' + error.message)
    // Revert switch on error
    rule.enabled = !rule.enabled
  }
}

const applyRules = async () => {
  applying.value = true
  try {
    await nftRuleAPI.apply()
    ElMessage.success('Rules applied to nftables successfully!')
  } catch (error) {
    ElMessage.error('Failed to apply rules: ' + error.message)
  } finally {
    applying.value = false
  }
}

const getActionType = (action) => {
  const types = {
    accept: 'success',
    drop: 'danger',
    queue: 'warning'
  }
  return types[action] || ''
}

onMounted(() => {
  loadRules()
})
</script>

<style scoped>
.nft-rules-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
