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
          <div>
            <el-button type="success" @click="applyRules" :loading="applying">Apply Rules</el-button>
            <el-button type="primary" @click="showRuleDialog()">Add Rule</el-button>
          </div>
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
    <el-dialog v-model="ruleDialogVisible" :title="ruleForm.ID ? 'Edit Rule' : 'Add Rule'" width="900px">
      <el-form :model="ruleForm" label-width="150px">
        <el-form-item label="Rule Name">
          <el-input v-model="ruleForm.name" />
        </el-form-item>
        
        <el-form-item label="Priority">
          <el-input-number v-model="ruleForm.priority" :min="0" />
        </el-form-item>

        <el-divider content-position="left">Match Conditions</el-divider>
        
        <!-- Visual Condition Builder -->
        <div v-for="(condition, index) in conditions" :key="index" class="condition-row">
          <el-select v-model="condition.field" placeholder="Select Field" style="width: 150px">
            <el-option v-for="field in fields" :key="field.name" :label="field.name" :value="field.name" />
          </el-select>
          
          <el-select v-model="condition.operator" placeholder="Operator" style="width: 100px; margin-left: 10px">
            <el-option label="==" value="==" />
            <el-option label="!=" value="!=" />
            <el-option label=">" value=">" />
            <el-option label="<" value="<" />
            <el-option label=">=" value=">=" />
            <el-option label="<=" value="<=" />
          </el-select>
          
          <el-input v-model="condition.value" placeholder="Value" style="width: 200px; margin-left: 10px" />
          
          <el-select v-if="index < conditions.length - 1" v-model="condition.logic" style="width: 80px; margin-left: 10px">
            <el-option label="AND" value="&&" />
            <el-option label="OR" value="||" />
          </el-select>
          
          <el-button 
            v-if="conditions.length > 1" 
            @click="removeCondition(index)" 
            type="danger" 
            size="small" 
            style="margin-left: 10px"
            icon="Delete"
          />
        </div>
        
        <el-button @click="addCondition" type="primary" size="small" style="margin-top: 10px">
          Add Condition
        </el-button>

        <el-divider content-position="left">Actions</el-divider>
        
        <!-- Visual Action Builder -->
        <div v-for="(action, index) in actions" :key="index" class="action-row">
          <el-select v-model="action.field" placeholder="Select Field" style="width: 150px">
            <el-option v-for="field in fields" :key="field.name" :label="field.name" :value="field.name" />
          </el-select>
          
          <el-select v-model="action.op" placeholder="Operation" style="width: 120px; margin-left: 10px">
            <el-option label="Set" value="set" />
            <el-option label="Add" value="add" />
            <el-option label="Subtract" value="sub" />
            <el-option label="Multiply" value="mul" />
            <el-option label="Divide" value="div" />
            <el-option label="Shell" value="shell" />
          </el-select>
          
          <el-input v-model="action.value" placeholder="Value" style="width: 250px; margin-left: 10px" />
          
          <el-button 
            v-if="actions.length > 1" 
            @click="removeAction(index)" 
            type="danger" 
            size="small" 
            style="margin-left: 10px"
            icon="Delete"
          />
        </div>
        
        <el-button @click="addAction" type="primary" size="small" style="margin-top: 10px">
          Add Action
        </el-button>

        <el-divider content-position="left">Processing Options</el-divider>
        
        <el-form-item label="Compute Checksum">
          <el-checkbox v-model="computeChecksum">Automatically recalculate packet checksum</el-checkbox>
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
import { Delete } from '@element-plus/icons-vue'

const fields = ref([])
const rules = ref([])
const fieldDialogVisible = ref(false)
const ruleDialogVisible = ref(false)
const applying = ref(false)

// Condition builder
const conditions = ref([{ field: '', operator: '==', value: '', logic: '&&' }])

// Action builder
const actions = ref([{ field: '', op: 'set', value: '' }])

// Processing options
const computeChecksum = ref(true)

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
  output_options: '',
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
    
    // Parse existing match condition
    parseMatchCondition(rule.match_condition)
    
    // Parse existing actions
    parseActions(rule.actions)
    
    // Parse output options
    parseOutputOptions(rule.output_options)
  } else {
    ruleForm.value = {
      name: '',
      priority: 0,
      match_condition: '',
      actions: '',
      output_options: '',
      enabled: true
    }
    conditions.value = [{ field: '', operator: '==', value: '', logic: '&&' }]
    actions.value = [{ field: '', op: 'set', value: '' }]
    computeChecksum.value = true
  }
  ruleDialogVisible.value = true
}

const parseMatchCondition = (conditionStr) => {
  // Simple parser for condition format: field1 == "value1" && field2 == "value2"
  if (!conditionStr) {
    conditions.value = [{ field: '', operator: '==', value: '', logic: '&&' }]
    return
  }
  
  // Split by && or ||
  const parts = conditionStr.split(/(\s+&&\s+|\s+\|\|\s+)/)
  const parsed = []
  
  for (let i = 0; i < parts.length; i += 2) {
    const part = parts[i].trim()
    const match = part.match(/(\w+)\s*(==|!=|>=|<=|>|<)\s*"([^"]*)"/)
    
    if (match) {
      parsed.push({
        field: match[1],
        operator: match[2],
        value: match[3],
        logic: i + 1 < parts.length ? parts[i + 1].trim() : '&&'
      })
    }
  }
  
  conditions.value = parsed.length > 0 ? parsed : [{ field: '', operator: '==', value: '', logic: '&&' }]
}

const parseActions = (actionsStr) => {
  if (!actionsStr) {
    actions.value = [{ field: '', op: 'set', value: '' }]
    return
  }
  
  try {
    const parsed = JSON.parse(actionsStr)
    actions.value = parsed.length > 0 ? parsed : [{ field: '', op: 'set', value: '' }]
  } catch {
    actions.value = [{ field: '', op: 'set', value: '' }]
  }
}

const parseOutputOptions = (optionsStr) => {
  if (!optionsStr) {
    computeChecksum.value = true
    return
  }
  
  try {
    const parsed = JSON.parse(optionsStr)
    computeChecksum.value = parsed.includes('compute_checksum')
  } catch {
    computeChecksum.value = true
  }
}

const addCondition = () => {
  conditions.value.push({ field: '', operator: '==', value: '', logic: '&&' })
}

const removeCondition = (index) => {
  conditions.value.splice(index, 1)
}

const addAction = () => {
  actions.value.push({ field: '', op: 'set', value: '' })
}

const removeAction = (index) => {
  actions.value.splice(index, 1)
}

const buildMatchCondition = () => {
  return conditions.value
    .filter(c => c.field && c.value)
    .map((c, index) => {
      const cond = `${c.field} ${c.operator} "${c.value}"`
      return index < conditions.value.length - 1 ? `${cond} ${c.logic}` : cond
    })
    .join(' ')
}

const buildActions = () => {
  return JSON.stringify(
    actions.value.filter(a => a.field && a.value)
  )
}

const buildOutputOptions = () => {
  const options = []
  if (computeChecksum.value) {
    options.push('compute_checksum')
  }
  return JSON.stringify(options)
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
    // Build condition and actions from UI
    const matchCondition = buildMatchCondition()
    const actionsJson = buildActions()
    const outputOptions = buildOutputOptions()
    
    if (!matchCondition) {
      ElMessage.warning('Please add at least one condition')
      return
    }
    
    if (!actionsJson || actionsJson === '[]') {
      ElMessage.warning('Please add at least one action')
      return
    }

    const data = {
      ...ruleForm.value,
      match_condition: matchCondition,
      actions: actionsJson,
      output_options: outputOptions
    }

    if (ruleForm.value.ID) {
      await ruleAPI.update(ruleForm.value.ID, data)
    } else {
      await ruleAPI.create(data)
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

const applyRules = async () => {
  applying.value = true
  try {
    const { nftRuleAPI } = await import('@/api')
    await nftRuleAPI.apply()
    ElMessage.success('Rules applied successfully')
  } catch (error) {
    ElMessage.error('Failed to apply rules: ' + error.message)
  } finally {
    applying.value = false
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

.condition-row, .action-row {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

.form-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
