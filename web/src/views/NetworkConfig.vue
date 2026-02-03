<template>
  <div class="network-config">
    <el-card class="config-card">
      <template #header>
        <div class="card-header">
          <span>Network Interfaces</span>
          <el-button @click="loadInterfaces" :icon="Refresh">Refresh</el-button>
        </div>
      </template>

      <el-table :data="interfaces" style="width: 100%">
        <el-table-column prop="name" label="Interface" width="150" />
        <el-table-column prop="hardware_addr" label="MAC Address" width="180" />
        <el-table-column prop="ip_addresses" label="IP Addresses">
          <template #default="{ row }">
            <el-tag v-for="ip in row.ip_addresses" :key="ip" size="small" style="margin: 2px">
              {{ ip }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="is_up" label="Status" width="100">
          <template #default="{ row }">
            <el-tag :type="row.is_up ? 'success' : 'info'">
              {{ row.is_up ? 'UP' : 'DOWN' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="configureVLAN(row.name)">
              Configure VLAN
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- VLAN Configuration Dialog -->
    <el-dialog v-model="vlanDialogVisible" title="Configure VLAN" width="600px">
      <el-form :model="vlanForm" label-width="120px">
        <el-form-item label="Interface">
          <el-input v-model="vlanForm.interface" disabled />
        </el-form-item>
        
        <el-form-item label="Link Type">
          <el-radio-group v-model="vlanForm.link_type">
            <el-radio label="access">Access</el-radio>
            <el-radio label="trunk">Trunk</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="vlanForm.link_type === 'access'" label="VLAN ID">
          <el-input-number v-model.number="vlanForm.vlan_id" :min="0" :max="4094" />
          <div class="form-hint">Set to 0 to remove VLAN configuration</div>
        </el-form-item>

        <template v-if="vlanForm.link_type === 'trunk'">
          <el-form-item label="Trunk VLANs">
            <el-input 
              v-model="vlanForm.trunk_vlan_id" 
              placeholder="e.g., 2,3,5-10"
            />
            <div class="form-hint">Comma-separated VLANs or ranges (e.g., 2,3,5-10)</div>
          </el-form-item>

          <el-form-item label="Default VLAN">
            <el-input-number v-model.number="vlanForm.default_id" :min="1" :max="4094" />
          </el-form-item>
        </template>
      </el-form>

      <template #footer>
        <el-button @click="vlanDialogVisible = false">Cancel</el-button>
        <el-button type="primary" @click="submitVLAN" :loading="submitting">
          Apply Configuration
        </el-button>
      </template>
    </el-dialog>

    <!-- VLAN IP Configuration -->
    <el-card class="config-card" style="margin-top: 20px">
      <template #header>
        <div class="card-header">
          <span>VLAN Interfaces</span>
        </div>
      </template>

      <el-form :model="ipForm" label-width="150px">
        <el-form-item label="VLAN Interface">
          <el-input v-model="ipForm.vlan_interface" placeholder="e.g., vlan_2" />
        </el-form-item>

        <el-form-item label="IP Addresses">
          <el-input 
            v-model="ipForm.ip_addresses" 
            placeholder="e.g., 192.168.10.100/24"
            type="textarea"
            :rows="3"
          />
          <div class="form-hint">One IP per line in CIDR notation</div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="addIP">Add IP</el-button>
          <el-button @click="flushIP">Flush All IPs</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { networkAPI } from '@/api'
import { ElMessage } from 'element-plus'

const interfaces = ref([])
const vlanDialogVisible = ref(false)
const submitting = ref(false)

const vlanForm = ref({
  interface: '',
  link_type: 'access',
  vlan_id: 2,
  trunk_vlan_id: '',
  default_id: 1
})

const ipForm = ref({
  vlan_interface: '',
  ip_addresses: ''
})

const loadInterfaces = async () => {
  try {
    const response = await networkAPI.listInterfaces()
    interfaces.value = response.data.data
  } catch (error) {
    ElMessage.error('Failed to load interfaces: ' + error.message)
  }
}

const configureVLAN = async (interfaceName) => {
  vlanForm.value.interface = interfaceName
  
  // Load existing configuration from database
  try {
    const response = await networkAPI.getVLANConfig(interfaceName)
    const config = response.data.data
    
    // Pre-populate form with existing config
    vlanForm.value.link_type = config.link_type || 'access'
    vlanForm.value.vlan_id = parseInt(config.vlan_id) || 0
    vlanForm.value.trunk_vlan_id = config.trunk_vlan_id || ''
    vlanForm.value.default_id = parseInt(config.default_id) || 1
  } catch (error) {
    ElMessage.warning('Could not load existing configuration: ' + error.message)
    // Use defaults if loading fails
    vlanForm.value.link_type = 'access'
    vlanForm.value.vlan_id = 0
    vlanForm.value.trunk_vlan_id = ''
    vlanForm.value.default_id = 1
  }
  
  vlanDialogVisible.value = true
}

const submitVLAN = async () => {
  submitting.value = true
  try {
    const data = {
      interface: vlanForm.value.interface,
      link_type: vlanForm.value.link_type
    }

    if (vlanForm.value.link_type === 'access') {
      data.vlan_id = String(vlanForm.value.vlan_id)
    } else {
      data.trunk_vlan_id = vlanForm.value.trunk_vlan_id
      data.default_id = String(vlanForm.value.default_id)
    }

    await networkAPI.configureVLAN(data)
    ElMessage.success('VLAN configured successfully')
    vlanDialogVisible.value = false
    loadInterfaces()
  } catch (error) {
    ElMessage.error('Failed to configure VLAN: ' + error.message)
  } finally {
    submitting.value = false
  }
}

const addIP = async () => {
  try {
    const ips = ipForm.value.ip_addresses.split('\n').filter(ip => ip.trim())
    await networkAPI.addVLANIP({
      vlan_interface: ipForm.value.vlan_interface,
      ip_addresses: ips
    })
    ElMessage.success('IP addresses added successfully')
    ipForm.value.ip_addresses = ''
  } catch (error) {
    ElMessage.error('Failed to add IP: ' + error.message)
  }
}

const flushIP = async () => {
  try {
    await networkAPI.removeVLANIP(ipForm.value.vlan_interface)
    ElMessage.success('IP addresses flushed successfully')
  } catch (error) {
    ElMessage.error('Failed to flush IP: ' + error.message)
  }
}

onMounted(() => {
  loadInterfaces()
})
</script>

<style scoped>
.network-config {
  max-width: 1400px;
  margin: 0 auto;
}

.config-card {
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
