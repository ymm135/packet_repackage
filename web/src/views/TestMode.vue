<template>
  <div class="test-mode">
    <el-card class="section-card">
      <template #header>
        <span>Test Rule Against Packet</span>
      </template>

      <el-form :model="testForm" label-width="120px">
        <el-form-item label="Hex Packet">
          <el-input 
            v-model="testForm.hex_packet" 
            type="textarea" 
            :rows="8"
            placeholder="Enter hex packet data (e.g., 000400010006002381672e81...)"
          />
          <div class="form-hint">Enter packet as continuous hex string or with spaces</div>
        </el-form-item>

        <el-form-item label="Rule">
          <el-select v-model="testForm.rule_id" clearable placeholder="Auto-match any rule">
            <el-option 
              v-for="rule in rules" 
              :key="rule.ID" 
              :label="rule.name" 
              :value="rule.ID"
            />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="runTest" :loading="testing">
            Run Test
          </el-button>
          <el-button @click="loadSample">Load Sample Packet</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- Results -->
    <el-card v-if="result" class="section-card">
      <template #header>
        <span>Test Results</span>
      </template>

      <!-- Processing Steps -->
      <div class="result-section">
        <h4>Processing Steps:</h4>
        <el-timeline>
          <el-timeline-item 
            v-for="(step, index) in result.processing_steps" 
            :key="index"
            :type="index === result.processing_steps.length - 1 ? 'success' : 'primary'"
          >
            {{ step }}
          </el-timeline-item>
          <el-timeline-item v-if="result.error" type="danger">
            Error: {{ result.error }}
          </el-timeline-item>
        </el-timeline>
      </div>

      <!-- Matched Rule -->
      <div v-if="result.matched_rule" class="result-section">
        <h4>Matched Rule:</h4>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="Name">{{ result.matched_rule.name }}</el-descriptions-item>
          <el-descriptions-item label="Condition">{{ result.matched_rule.match_condition }}</el-descriptions-item>
          <el-descriptions-item label="Priority">{{ result.matched_rule.priority }}</el-descriptions-item>
        </el-descriptions>
      </div>

      <!-- 5-Tuple Info -->
      <div v-if="result.src_ip" class="result-section">
        <h4>5-Tuple Info:</h4>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Source">{{ result.src_ip }}:{{ result.src_port }}</el-descriptions-item>
          <el-descriptions-item label="Destination">{{ result.dst_ip }}:{{ result.dst_port }}</el-descriptions-item>
          <el-descriptions-item label="Protocol">{{ result.protocol }}</el-descriptions-item>
        </el-descriptions>
      </div>

      <!-- Parsed Fields -->
      <div class="result-section">
        <h4>Extracted Fields:</h4>
        <el-table :data="formatFields(result.parsed_fields)" border style="width: 100%">
          <el-table-column prop="name" label="Field Name" width="200" />
          <el-table-column prop="value" label="Value" />
        </el-table>
      </div>

      <!-- Modified Fields -->
      <div v-if="Object.keys(result.modified_fields || {}).length > 0" class="result-section">
        <h4>Modified Fields:</h4>
        <el-table :data="formatModifiedFields(result.modified_fields)" border style="width: 100%">
          <el-table-column prop="name" label="Field Name" width="200" />
          <el-table-column prop="before" label="Before" />
          <el-table-column prop="after" label="After">
            <template #default="{ row }">
              <span style="color: #67c23a; font-weight: bold">{{ row.after }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- Packets -->
      <div class="result-section">
        <h4>Original Packet:</h4>
        <hex-viewer :hex="result.original_packet" />
      </div>

      <div v-if="result.modified_packet" class="result-section">
        <h4>Modified Packet:</h4>
        <hex-viewer :hex="result.modified_packet" />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { testAPI, ruleAPI } from '@/api'
import { ElMessage } from 'element-plus'
import HexViewer from '@/components/HexViewer.vue'

const rules = ref([])
const testing = ref(false)
const result = ref(null)

const testForm = ref({
  hex_packet: '',
  rule_id: null
})

const samplePacket = '000400010006002381672e81000008004500005e4ba640004011f49eac10a0edac10013c18d018d0004ac047b2c20a00fcf26469010000004b3c030058480500115104000017000b5c5df2f109000000000001000082304248423130413031595030315f706d742e6f7073657400'

const loadRules = async () => {
  try {
    const response = await ruleAPI.list()
    rules.value = response.data.data || []
  } catch (error) {
    ElMessage.error('Failed to load rules')
  }
}

const loadSample = () => {
  testForm.value.hex_packet = samplePacket
}

const runTest = async () => {
  testing.value = true
  result.value = null

  try {
    // Clean hex input (remove spaces, newlines)
    const cleanHex = testForm.value.hex_packet.replace(/[\s\n\r]/g, '')
    
    const response = await testAPI.test({
      hex_packet: cleanHex,
      rule_id: testForm.value.rule_id || 0
    })
    
    result.value = response.data
    ElMessage.success('Test completed')
  } catch (error) {
    ElMessage.error('Test failed: ' + error.message)
  } finally {
    testing.value = false
  }
}

const formatFields = (fields) => {
  if (!fields) return []
  return Object.entries(fields).map(([name, value]) => ({
    name,
    value: value
  }))
}

const formatModifiedFields = (fields) => {
  if (!fields) return []
  return Object.entries(fields).map(([name, values]) => ({
    name,
    before: values.before,
    after: values.after
  }))
}

onMounted(() => {
  loadRules()
})
</script>

<style scoped>
.test-mode {
  max-width: 1400px;
  margin: 0 auto;
}

.section-card {
  margin-bottom: 20px;
}

.result-section {
  margin-bottom: 30px;
}

.result-section h4 {
  color: #303133;
  margin-bottom: 15px;
  font-size: 16px;
}

.form-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
