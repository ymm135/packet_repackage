<template>
  <el-container class="app-container">
    <el-header class="app-header">
      <div class="header-content">
        <h1>Packet Repackage System</h1>
        <el-menu mode="horizontal" :default-active="activeRoute" router>
          <el-menu-item index="/network">Network Config</el-menu-item>
          <el-menu-item index="/nftrules">Firewall Rules</el-menu-item>
          <el-menu-item index="/rules">Rules</el-menu-item>
          <el-menu-item index="/test">Test Mode</el-menu-item>
          <el-menu-item index="/logs">Logs</el-menu-item>
        </el-menu>
      </div>
    </el-header>
    <el-main class="app-main">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const activeRoute = computed(() => route.path)
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

.app-container {
  height: 100%;
}

.app-header {
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  padding: 0;
}

.header-content {
  max-width: 1400px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  height: 100%;
  padding: 0 20px;
}

.header-content h1 {
  font-size: 20px;
  margin-right: 40px;
  color: #409eff;
}

.app-main {
  background: #f5f5f5;
  padding: 20px;
  overflow-y: auto;
}
</style>
