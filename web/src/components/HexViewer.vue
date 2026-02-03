<template>
  <div class="hex-viewer">
    <div class="hex-display">
      <table>
        <tbody>
          <tr v-for="(line, index) in lines" :key="index">
            <td class="offset">{{ formatOffset(index * 16) }}</td>
            <td class="hex-bytes">{{ line.hex }}</td>
            <td class="ascii">{{ line.ascii }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  hex: {
    type: String,
    required: true
  }
})

const lines = computed(() => {
  const hexStr = props.hex.replace(/\s/g, '')
  const result = []
  
  for (let i = 0; i < hexStr.length; i += 32) {
    const hexChunk = hexStr.substring(i, i + 32)
    const bytes = []
    const ascii = []
    
    for (let j = 0; j < hexChunk.length; j += 2) {
      const byte = hexChunk.substring(j, j + 2)
      bytes.push(byte)
      
      // Convert to ASCII
      const charCode = parseInt(byte, 16)
      if (charCode >= 32 && charCode <= 126) {
        ascii.push(String.fromCharCode(charCode))
      } else {
        ascii.push('.')
      }
    }
    
    // Format hex bytes with spaces every 2 bytes
    const hexFormatted = bytes.join(' ')
    
    result.push({
      hex: hexFormatted.padEnd(47, ' '), // 16 bytes * 3 - 1
      ascii: ascii.join('')
    })
  }
  
  return result
})

const formatOffset = (offset) => {
  return offset.toString(16).toUpperCase().padStart(4, '0')
}
</script>

<style scoped>
.hex-viewer {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 15px;
  border-radius: 4px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  overflow-x: auto;
}

.hex-display table {
  border-collapse: collapse;
  width: 100%;
}

.offset {
  color: #858585;
  padding-right: 15px;
  text-align: right;
  user-select: none;
}

.hex-bytes {
  color: #9cdcfe;
  padding-right: 20px;
  font-weight: 500;
}

.ascii {
  color: #ce9178;
}

td {
  padding: 2px 0;
  white-space: nowrap;
}
</style>
