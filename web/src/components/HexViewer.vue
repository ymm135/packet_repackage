<template>
  <div class="hex-viewer">
    <div class="hex-display">
      <table>
        <tbody>
          <tr v-for="(line, lineIndex) in lines" :key="lineIndex">
            <td class="offset">{{ formatOffset(lineIndex * 16) }}</td>
            <td class="hex-bytes">
              <span 
                v-for="(item, itemIndex) in line.bytes" 
                :key="itemIndex"
                class="byte-item"
                :class="{ 
                  selected: selectedIndex === item.globalIndex,
                  gap: itemIndex > 0 && itemIndex % 2 === 0 
                }"
                @click="selectIndex(item.globalIndex)"
              >{{ item.hex }}</span>
            </td>
            <td class="ascii">
              <span 
                v-for="(item, itemIndex) in line.bytes" 
                :key="itemIndex"
                class="ascii-item"
                :class="{ selected: selectedIndex === item.globalIndex }"
                @click="selectIndex(item.globalIndex)"
              >{{ item.ascii }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  hex: {
    type: String,
    required: true
  }
})

const selectedIndex = ref(-1)

const selectIndex = (index) => {
  selectedIndex.value = index
}

const lines = computed(() => {
  const hexStr = props.hex.replace(/\s/g, '')
  const result = []
  
  let globalIndex = 0
  
  for (let i = 0; i < hexStr.length; i += 32) {
    const hexChunk = hexStr.substring(i, i + 32)
    const bytes = []
    
    for (let j = 0; j < hexChunk.length; j += 2) {
      const hexByte = hexChunk.substring(j, j + 2)
      
      // Convert to ASCII
      const charCode = parseInt(hexByte, 16)
      let asciiChar = '.'
      if (charCode >= 32 && charCode <= 126) {
        asciiChar = String.fromCharCode(charCode)
      }
      
      bytes.push({
        hex: hexByte,
        ascii: asciiChar,
        globalIndex: globalIndex++
      })
    }
    
    result.push({ bytes })
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
  vertical-align: top;
}

.hex-bytes {
  color: #9cdcfe;
  padding-right: 20px;
  font-weight: 500;
  vertical-align: top;
}

.ascii {
  color: #ce9178;
  vertical-align: top;
}

.byte-item, .ascii-item {
  display: inline-block;
  cursor: pointer;
  padding: 0 1px;
}

.byte-item:hover, .ascii-item:hover {
  background-color: #3a3d41;
}

.byte-item.selected, .ascii-item.selected {
  background-color: #264f78; /* VS Code text selection color */
  color: #ffffff;
}

.byte-item.gap {
  margin-left: 8px; /* Space between every 2 bytes */
}

td {
  padding: 2px 0;
  white-space: nowrap;
}
</style>
