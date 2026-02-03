<template>
  <div class="rules-page">
    <!-- Fields Management -->
    <el-card class="section-card">
      <template #header>
        <div class="card-header">
          <span>Field Definitions</span>
          <el-button type="primary" @click="showFieldDialog()">Add Field</el-button>
        </div>
      </template>

      <el-table :data="fields" style="width: 100%">
        <el-table-column prop="name" label="Name" width="150" />
        <el-table-column prop="offset" label="Offset">
          <template #default="{ row }">
            0x{{ row.offset.toString(16).toUpperCase() }} ({{ row.offset }})
          </template>
        </el-table-column>
        <el-table-column prop="length" label="Length" width="100" />
        <el-table-column prop="type" label="Type" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="180">
          <template #default="{ row }">
            <el-button size="small" @click="showFieldDialog(row)">Edit</el-button>
            <el-button size="small" type="danger" @click="deleteField(row.ID)">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Rules Management -->
    <el-card class="section-card">
      <template #header>
        <div class="card-header">
          <span>Packet Modification Rules</span>
          <el-button type="primary" @click="showRuleDialog()">Add Rule</el-button>
        </div>
      </template>

      <el-table :data="rules" style="width: 100%">
        <el-table-column prop="name" label="Rule Name" width="200" />
        <el-table-column prop="match_condition" label="Match Condition" show-overflow-tooltip />
        <el-table-column prop="priority" label="Priority" width="100" />
        <el-table-column prop="enabled" label="Status" width="100">
          <template #default="{ row }">
            <el-switch 
              v-model="row.enabled" 
              @change="toggleRule(row.ID)"
            />
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="180">
          <template #default="{ row }">
            <el-button size="small" @click="showRuleDialog(row)">Edit</el-button>
            <el-button size="small" type="danger" @click="deleteRule(row.ID)">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Field Dialog -->
    <el-dialog v-model="fieldDialogVisible" :title="fieldForm.ID ? 'Edit Field' : 'Add Field'" width="500px">
      <el-form :model="fieldForm" label-width="100px">
        <el-form-item label="Name">
          <el-input v-model="fieldForm.name" />
        </el-form-item>
        <el-form-item label="Offset">
          <el-input v-model="fieldForm.offset" placeholder="e.g., 0x58 or 88">
            <template #prepend>Hex/Dec</template>
          </el-input>
        </el-form-item>
        <el-form-item label="Length">
          <el-input-number v-model="fieldForm.length" :min="1" />
        </el-form-item>
        <el-form-item label="Type">
          <el-select v-model="fieldForm.type">
            <el-option label="Hex" value="hex" />
            <el-option label="Decimal" value="decimal" />
            <el-option label="String" value="string" />
            <el-option label="Built-in" value="builtin" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="fieldDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="saveField">Save</el-button>
      </template>
    </el-dialog>

    <!-- Rule Dialog -->
    <el-dialog v-model="ruleDialogVisible" :title="ruleForm.ID ? 'Edit Rule' : 'Add Rule'" width="800px">
      <el-form :model="ruleForm" label-width="150px">
        <el-form-item label="Rule Name">
          <el-input v-model="ruleForm.name" />
        </el-form-item>
        
        <el-form-item label="Priority">
          <el-input-number v-model="ruleForm.priority" :min="0" />
        </el-form-item>

        <el-form-item label="Match Condition">
          <el-input 
            v-model="ruleForm.match_condition" 
            type="textarea" 
            :rows="3"
            placeholder='e.g., tagName == "BHB10A01YP01_pmt" && option == "opset"'
          />
          <div class="form-hint">Use field names with ==, !=, &&, ||, !, ()</div>
        </el-form-item>

        <el-form-item label="Actions">
          <el-input 
            v-model="ruleForm.actions" 
            type="textarea" 
            :rows="4"
            placeholder='[{"field": "tagName", "op": "set", "value": "NewValue"}]'
          />
          <div class="form-hint">JSON array of actions: set, add, sub, mul, div, shell</div>
        </el-form-item>

        <el-form-item label="Output Template">
          <el-input 
            v-model="ruleForm.output_template" 
            type="textarea" 
            :rows="2"
            placeholder='e.g., tagName + 0x2e + option'
          />
          <div class="form-hint">Field names separated by + with optional hex literals (0xHH)</div>
        </el-form-item>

        <el-form-item label="Enabled">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="saveRule">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { fieldAPI, ruleAPI } from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const fields = ref([])
const rules = ref([])
const fieldDialogVisible = ref(false)
const ruleDialogVisible = ref(false)

const fieldForm = ref({
  name: '',
  offset: '',
  length: 1,
  type: 'hex'
})

const ruleForm = ref({
  name: '',
  priority: 0,
  match_condition: '',
  actions: '',
  output_template: '',
  enabled: true
})

const loadFields = async () => {
  try {
    const response = await fieldAPI.list()
    fields.value = response.data.data || []
  } catch (error) {
    ElMessage.error('Failed to load fields')
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

const showFieldDialog = (field = null) => {
  if (field) {
    fieldForm.value = { ...field }
  } else {
    fieldForm.value = { name: '', offset: '', length: 1, type: 'hex' }
  }
  fieldDialogVisible.value = true
}

const showRuleDialog = (rule = null) => {
  if (rule) {
    ruleForm.value = { ...rule }
  } else {
    ruleForm.value = {
      name: '',
      priority: 0,
      match_condition: '',
      actions: '',
      output_template: '',
      enabled: true
    }
  }
  ruleDialogVisible.value = true
}

const saveField = async () => {
  try {
    // Parse offset (support hex and decimal)
    let offset = fieldForm.value.offset
    if (typeof offset === 'string') {
      if (offset.startsWith('0x')) {
        offset = parseInt(offset, 16)
      } else {
        offset = parseInt(offset, 10)
      }
    }
    
    const data = {
      ...fieldForm.value,
      offset
    }

    if (fieldForm.value.ID) {
      await fieldAPI.update(fieldForm.value.ID, data)
    } else {
      await fieldAPI.create(data)
    }
    
    ElMessage.success('Field saved successfully')
    fieldDialogVisible.value = false
    loadFields()
  } catch (error) {
    ElMessage.error('Failed to save field: ' + error.message)
  }
}

const saveRule = async () => {
  try {
    // Validate JSON actions if provided
    if (ruleForm.value.actions) {
      JSON.parse(ruleForm.value.actions)
    }

    if (ruleForm.value.ID) {
      await ruleAPI.update(ruleForm.value.ID, ruleForm.value)
    } else {
      await ruleAPI.create(ruleForm.value)
    }
    
    ElMessage.success('Rule saved successfully')
    ruleDialogVisible.value = false
    loadRules()
  } catch (error) {
    ElMessage.error('Failed to save rule: ' + error.message)
  }
}

const deleteField = async (id) => {
  try {
    await ElMessageBox.confirm('Delete this field?', 'Warning', { type: 'warning' })
    await fieldAPI.delete(id)
    ElMessage.success('Field deleted')
    loadFields()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to delete field')
    }
  }
}

const deleteRule = async (id) => {
  try {
    await ElMessageBox.confirm('Delete this rule?', 'Warning', { type: 'warning' })
    await ruleAPI.delete(id)
    ElMessage.success('Rule deleted')
    loadRules()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('Failed to delete rule')
    }
  }
}

const toggleRule = async (id) => {
  try {
    await ruleAPI.toggle(id)
    ElMessage.success('Rule status updated')
  } catch (error) {
    ElMessage.error('Failed to toggle rule')
    loadRules() // Reload to revert UI
  }
}

onMounted(() => {
  loadFields()
  loadRules()
})
</script>

<style scoped>
.rules-page {
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

.form-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
