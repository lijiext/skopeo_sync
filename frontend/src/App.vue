<template>
  <div v-if="!token" class="auth-wrapper">
    <Login @login-success="onLoginSuccess" />
  </div>

  <div v-else class="app-container">
    <!-- 顶部导航栏 -->
    <div class="header">
      <h1 class="logo">Skopeo 同步管理系统</h1>
      <div class="actions">
        <el-button type="info" plain @click="openWebhookModal">通知设置</el-button>
        <el-button type="success" plain @click="openRegistryManager">仓库管理</el-button>
        <el-button type="primary" @click="openTaskModal">+ 批量新建同步</el-button>
        <el-button type="danger" text @click="handleLogout">退出登录</el-button>
      </div>
    </div>

    <div class="main-content">
      <!-- 数据看板 -->
      <el-row :gutter="20" class="dashboard" style="margin-bottom: 20px;">
        <el-col :span="12">
          <el-card shadow="hover" class="stat-card border-blue">
            <template #header>
              <div class="stat-header">总消耗流量</div>
            </template>
            <div class="stat-value text-blue">{{ totalTrafficMB.toFixed(2) }} MB</div>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="hover" class="stat-card border-green">
            <template #header>
              <div class="stat-header">活动任务数</div>
            </template>
            <div class="stat-value text-green">{{ activeTasks }}</div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 任务列表 -->
      <el-card shadow="never" class="task-list-card">
        <template #header>
          <div class="card-title">最近同步任务</div>
        </template>
        
        <el-table :data="tasks" stripe style="width: 100%" v-loading="loadingTasks">
          <el-table-column prop="ID" label="ID" width="80" align="center" />
          <el-table-column label="镜像映射关系" min-width="300">
            <template #default="{ row }">
              <div class="image-map">
                <div class="src-img" :title="getRegistryUrl(row.source_id) + '/' + row.image">
                  <el-tag size="small" type="info" effect="plain" style="margin-right: 5px;">源</el-tag>
                  {{ getRegistryUrl(row.source_id) }}/{{ row.image }}
                </div>
                <div class="dest-img" :title="getRegistryUrl(row.dest_id) + '/' + row.dest_image">
                  <el-tag size="small" type="primary" effect="plain" style="margin-right: 5px;">目标</el-tag>
                  {{ getRegistryUrl(row.dest_id) }}/{{ row.dest_image }}
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="150" align="center">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)" effect="dark">
                {{ row.status }}
              </el-tag>
              <div v-if="row.current_retry > 0" class="retry-text">
                重试: {{ row.current_retry }}/{{ row.retries }}
              </div>
            </template>
          </el-table-column>
          <el-table-column label="流量消耗" width="120" align="center">
            <template #default="{ row }">
              {{ row.traffic_bytes > 0 ? (row.traffic_bytes / 1048576).toFixed(2) + ' MB' : '-' }}
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="180" align="center">
            <template #default="{ row }">
              {{ new Date(row.CreatedAt).toLocaleString() }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <el-button 
                size="small" 
                type="primary" 
                link 
                @click="openLogModal(row)"
              >
                执行日志
              </el-button>
              <el-button 
                v-if="['failed', 'success'].includes(row.status)" 
                size="small" 
                type="warning" 
                link 
                @click="handleRetry(row.ID)"
              >
                重试
              </el-button>
              <el-button 
                v-if="row.status !== 'running'"
                size="small" 
                type="danger" 
                link 
                @click="handleDeleteTask(row.ID)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <!-- 弹窗: 仓库管理列表 -->
    <el-dialog v-model="showRegistryManagerModal" title="仓库管理" width="700px" destroy-on-close>
      <div style="margin-bottom: 15px;">
        <el-button type="primary" size="small" @click="openAddRegistry">+ 新增仓库配置</el-button>
      </div>
      <el-table :data="registries" stripe size="small" style="width: 100%">
        <el-table-column prop="ID" label="ID" width="60" />
        <el-table-column prop="name" label="名称" width="120" />
        <el-table-column prop="url" label="地址 (URL)" />
        <el-table-column prop="username" label="用户名" width="120" />
        <el-table-column label="操作" width="120" align="center">
          <template #default="{ row }">
            <el-button size="small" type="primary" link @click="openEditRegistry(row)">编辑</el-button>
            <el-button size="small" type="danger" link @click="handleDeleteRegistry(row.ID)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 弹窗: 添加/编辑仓库 -->
    <el-dialog v-model="showRegistryModal" :title="isEditRegistry ? '编辑仓库配置' : '添加仓库配置'" width="500px" destroy-on-close>
      <el-form :model="registryForm" label-position="top">
        <el-form-item label="仓库名称" required>
          <el-input v-model="registryForm.name" placeholder="例如: Docker Hub" />
        </el-form-item>
        <el-form-item label="仓库地址 (URL)" required>
          <el-input v-model="registryForm.url" placeholder="例如: docker.io" />
        </el-form-item>
        <el-form-item label="用户名 (可选, 公开源留空)">
          <el-input v-model="registryForm.username" placeholder="认证用户名" />
        </el-form-item>
        <el-form-item label="密码/Token (可选)">
          <el-input v-model="registryForm.password" type="password" placeholder="认证密码" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRegistryModal = false">取消</el-button>
        <el-button type="primary" @click="submitRegistry" :loading="submittingReg">保存</el-button>
      </template>
    </el-dialog>

    <!-- 弹窗: 批量新建任务 -->
    <el-dialog v-model="showTaskModal" title="批量新建同步任务" width="650px" destroy-on-close>
      <el-form :model="taskForm" label-position="top">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="源仓库" required>
              <el-select v-model="taskForm.source_id" placeholder="选择源仓库" style="width: 100%">
                <el-option v-for="r in registries" :key="r.ID" :label="`${r.name} (${r.url})`" :value="r.ID" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="目标仓库" required>
              <el-select v-model="taskForm.dest_id" placeholder="选择目标仓库" style="width: 100%">
                <el-option v-for="r in registries" :key="r.ID" :label="`${r.name} (${r.url})`" :value="r.ID" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="镜像列表 (每行一个，支持 '源 -> 目标' 显式映射语法)" required>
          <el-input 
            type="textarea" 
            v-model="taskForm.images" 
            :rows="5" 
            placeholder="library/nginx:stable-alpine3.23-perl&#10;library/alpine:latest -> myrepo/alpine:test"
          />
        </el-form-item>

        <!-- 实时解析预览 -->
        <div v-if="previewList.length > 0" class="preview-box">
          <div class="preview-header">解析预览 ({{ previewList.length }} 条)</div>
          <ul class="preview-list">
            <li v-for="(p, idx) in previewList" :key="idx" class="preview-item">
              <span class="preview-src" :title="p.src">源: {{ p.src }}</span>
              <span class="preview-dest" :title="p.dest">目标: {{ p.dest }}</span>
            </li>
          </ul>
        </div>

        <el-form-item label="单镜像最大重试次数" required style="margin-top: 15px;">
          <el-input-number v-model="taskForm.retries" :min="1" :max="10" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTaskModal = false">取消</el-button>
        <el-button type="primary" @click="submitTask" :loading="submittingTask">批量提交</el-button>
      </template>
    </el-dialog>

    <!-- 弹窗: Webhook 设置 -->
    <el-dialog v-model="showWebhookModal" title="全局通知设置" width="500px" destroy-on-close>
      <el-form :model="webhookForm" label-position="top">
        <el-form-item label="Webhook 平台类型">
          <el-select v-model="webhookForm.type" style="width: 100%">
            <el-option label="通用格式 (JSON)" value="general" />
            <el-option label="钉钉机器人 (Text)" value="dingtalk" />
            <el-option label="企业微信机器人 (Text)" value="wechat" />
            <el-option label="飞书机器人 (Text)" value="feishu" />
          </el-select>
        </el-form-item>
        <el-form-item label="Webhook URL">
          <el-input v-model="webhookForm.url" placeholder="留空则不发送通知" clearable />
          <div class="form-tip">系统将在每个同步任务处于终态（失败或成功）时触发 HTTP POST 推送通知。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showWebhookModal = false">取消</el-button>
        <el-button type="primary" @click="submitWebhook" :loading="submittingWebhook">保存配置</el-button>
      </template>
    </el-dialog>

    <!-- 弹窗: 日志终端 -->
    <el-dialog 
      v-model="showLogModal" 
      title="任务执行日志" 
      width="850px" 
      @close="closeLogModal"
      top="5vh"
    >
      <div class="log-terminal-container">
        <div class="log-terminal" ref="logContainerRef">
          {{ liveLog || '等待日志数据...' }}
        </div>
      </div>
      <template #footer>
        <el-button @click="showLogModal = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import axios from 'axios'
import Login from './components/Login.vue'

// ======= 基础状态 =======
const token = ref(localStorage.getItem('skopeo-token') || '')

const tasks = ref<any[]>([])
const registries = ref<any[]>([])
const loadingTasks = ref(false)
const totalTrafficMB = ref(0)
const activeTasks = ref(0)

// ======= 拦截器 =======
axios.interceptors.request.use(config => {
  if (token.value) {
    config.headers.Authorization = `Bearer ${token.value}`
  }
  return config
})

axios.interceptors.response.use(res => res, err => {
  if (err.response?.status === 401) {
    handleLogout()
  }
  return Promise.reject(err)
})

// ======= 方法: 登录与退出 =======
const onLoginSuccess = (newToken: string) => {
  token.value = newToken
  localStorage.setItem('skopeo-token', newToken)
  ElMessage.success('登录成功')
  initData()
}

const handleLogout = () => {
  token.value = ''
  localStorage.removeItem('skopeo-token')
  stopPolling()
}

// ======= 数据拉取 =======
let pollTimer: any = null

const fetchTasks = async () => {
  try {
    const res = await axios.get('/api/tasks')
    tasks.value = res.data || []
    
    let bytes = 0
    let active = 0
    tasks.value.forEach(t => {
      bytes += t.traffic_bytes || 0
      if (t.status === 'pending' || t.status === 'running') active++
    })
    totalTrafficMB.value = bytes / 1048576
    activeTasks.value = active
  } catch (error) {
    console.error('Fetch tasks error:', error)
  }
}

const fetchRegistries = async () => {
  try {
    const res = await axios.get('/api/registries')
    registries.value = res.data || []
  } catch (error) {
    console.error('Fetch registries error:', error)
  }
}

const startPolling = () => {
  if (pollTimer) clearInterval(pollTimer)
  fetchTasks()
  pollTimer = setInterval(fetchTasks, 3000)
}

const stopPolling = () => {
  if (pollTimer) clearInterval(pollTimer)
}

const initData = () => {
  if (!token.value) return
  fetchRegistries()
  startPolling()
}

onMounted(() => {
  initData()
})

onUnmounted(() => {
  stopPolling()
})

// 提取 registry url
const getRegistryUrl = (id: number) => {
  const reg = registries.value.find(r => r.ID === id)
  return reg ? reg.url : '未知'
}

const getStatusType = (status: string) => {
  switch (status) {
    case 'success': return 'success'
    case 'failed': return 'danger'
    case 'running': return 'primary'
    case 'pending': return 'warning'
    default: return 'info'
  }
}

// ======= 弹窗: 仓库管理 =======
const showRegistryManagerModal = ref(false)
const showRegistryModal = ref(false)
const isEditRegistry = ref(false)
const editRegistryId = ref<number | null>(null)
const submittingReg = ref(false)
const registryForm = reactive({ name: '', url: '', username: '', password: '' })

const openRegistryManager = () => {
  showRegistryManagerModal.value = true
}

const openAddRegistry = () => {
  isEditRegistry.value = false
  editRegistryId.value = null
  registryForm.name = ''
  registryForm.url = ''
  registryForm.username = ''
  registryForm.password = ''
  showRegistryModal.value = true
}

const openEditRegistry = (row: any) => {
  isEditRegistry.value = true
  editRegistryId.value = row.ID
  registryForm.name = row.name
  registryForm.url = row.url
  registryForm.username = row.username
  registryForm.password = '' // 编辑时不显示已有密码
  showRegistryModal.value = true
}

const handleDeleteRegistry = async (id: number) => {
  try {
    await axios.delete(`/api/registries/${id}`)
    ElMessage.success('仓库删除成功')
    fetchRegistries()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '删除失败')
  }
}

const submitRegistry = async () => {
  if (!registryForm.name || !registryForm.url) {
    ElMessage.warning('名称和URL为必填项')
    return
  }
  submittingReg.value = true
  try {
    if (isEditRegistry.value && editRegistryId.value) {
      await axios.put(`/api/registries/${editRegistryId.value}`, registryForm)
      ElMessage.success('仓库配置更新成功')
    } else {
      await axios.post('/api/registries', registryForm)
      ElMessage.success('仓库配置添加成功')
    }
    showRegistryModal.value = false
    fetchRegistries()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '保存失败')
  } finally {
    submittingReg.value = false
  }
}

// ======= 弹窗: 新建任务 =======
const showTaskModal = ref(false)
const submittingTask = ref(false)
const taskForm = reactive({ source_id: '', dest_id: '', images: '', retries: 3 })

const openTaskModal = () => {
  taskForm.source_id = ''
  taskForm.dest_id = ''
  taskForm.images = ''
  taskForm.retries = 3
  fetchRegistries()
  showTaskModal.value = true
}

const submitTask = async () => {
  if (!taskForm.source_id || !taskForm.dest_id || !taskForm.images) {
    ElMessage.warning('请填写完整的源、目标及镜像列表')
    return
  }
  submittingTask.value = true
  try {
    await axios.post('/api/tasks', {
      source_id: Number(taskForm.source_id),
      dest_id: Number(taskForm.dest_id),
      images: taskForm.images,
      retries: taskForm.retries
    })
    ElMessage.success('批量同步任务创建成功')
    showTaskModal.value = false
    taskForm.images = ''
    fetchTasks()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '任务创建失败')
  } finally {
    submittingTask.value = false
  }
}

// 实时预览逻辑
const previewList = computed(() => {
  if (!taskForm.images || !taskForm.dest_id || !taskForm.source_id) return []
  
  const srcReg = registries.value.find(r => r.ID === Number(taskForm.source_id))
  const destReg = registries.value.find(r => r.ID === Number(taskForm.dest_id))
  const lines = taskForm.images.split('\n').map(l => l.trim()).filter(l => l !== '')
  
  return lines.map(line => {
    let src = '', dest = ''
    if (line.includes('->')) {
      const parts = line.split('->')
      src = parts[0].trim()
      dest = parts[1].trim()
    } else {
      src = line
      let shortName = src
      const idx = src.lastIndexOf('/')
      if (idx !== -1) {
        shortName = src.substring(idx + 1)
      }
      if (destReg && destReg.username) {
        dest = `${destReg.username}/${shortName}`
      } else {
        dest = src
      }
    }
    
    // 加上完整 registry url 进行展示
    const srcFull = srcReg ? `${srcReg.url}/${src}` : src
    const destFull = destReg ? `${destReg.url}/${dest}` : dest
    
    return { src: srcFull, dest: destFull }
  })
})

// ======= 弹窗: Webhook 设置 =======
const showWebhookModal = ref(false)
const submittingWebhook = ref(false)
const webhookForm = reactive({ type: 'general', url: '' })

const openWebhookModal = async () => {
  showWebhookModal.value = true
  try {
    const [resUrl, resType] = await Promise.all([
      axios.get('/api/config?key=webhook_url'),
      axios.get('/api/config?key=webhook_type')
    ])
    webhookForm.url = resUrl.data?.value || ''
    webhookForm.type = resType.data?.value || 'general'
  } catch (error) {
    console.error('Failed to load webhook config', error)
  }
}

const submitWebhook = async () => {
  submittingWebhook.value = true
  try {
    await Promise.all([
      axios.post('/api/config', { key: 'webhook_url', value: webhookForm.url }),
      axios.post('/api/config', { key: 'webhook_type', value: webhookForm.type })
    ])
    ElMessage.success('通知设置保存成功')
    showWebhookModal.value = false
  } catch (error: any) {
    ElMessage.error('保存失败')
  } finally {
    submittingWebhook.value = false
  }
}

// ======= 操作: 重试 =======
const handleRetry = async (id: number) => {
  try {
    await axios.post(`/api/tasks/${id}/retry`)
    ElMessage.success('任务已加入等待队列')
    fetchTasks()
  } catch (error) {
    ElMessage.error('重试请求失败')
  }
}

const handleDeleteTask = async (id: number) => {
  try {
    await axios.delete(`/api/tasks/${id}`)
    ElMessage.success('任务删除成功')
    fetchTasks()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '任务删除失败')
  }
}

// ======= 弹窗: 日志终端 =======
const showLogModal = ref(false)
const logIsLive = ref(false)
const liveLog = ref('')
const logContainerRef = ref<HTMLElement | null>(null)
let wsConnection: WebSocket | null = null

const scrollToBottom = () => {
  nextTick(() => {
    if (logContainerRef.value) {
      logContainerRef.value.scrollTop = logContainerRef.value.scrollHeight
    }
  })
}

watch(liveLog, () => {
  scrollToBottom()
})

const openLogModal = (task: any) => {
  liveLog.value = ''
  showLogModal.value = true

  if (task.status === 'success' || task.status === 'failed') {
    logIsLive.value = false
    liveLog.value = '正在拉取日志...'
    axios.get(`/api/tasks/${task.ID}/logs`).then(res => {
      liveLog.value = res.data.logs || '暂无日志数据'
    }).catch(() => {
      liveLog.value = '拉取日志失败'
    })
  } else {
    logIsLive.value = true
    liveLog.value = '连接日志流...'
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsHost = window.location.host
    const wsUrl = `${protocol}//${wsHost}/api/ws/logs?task_id=${task.ID}`
    
    wsConnection = new WebSocket(wsUrl)
    wsConnection.onmessage = (e) => {
      liveLog.value = e.data
    }
    wsConnection.onerror = () => {
      liveLog.value += '\n[WebSocket 连接出错，请检查网络或后端是否存活]'
    }
    wsConnection.onclose = () => {
      liveLog.value += '\n[日志流结束]'
    }
  }
}

const closeLogModal = () => {
  if (wsConnection) {
    wsConnection.close()
    wsConnection = null
  }
  liveLog.value = ''
}
</script>

<style scoped>
.app-container {
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.logo {
  margin: 0;
  color: #303133;
  font-size: 26px;
}

.stat-card {
  border-left: 4px solid transparent;
}
.border-blue { border-left-color: #409eff; }
.border-green { border-left-color: #67c23a; }

.stat-header {
  font-size: 16px;
  font-weight: 600;
  color: #606266;
}
.stat-value {
  font-size: 28px;
  font-weight: bold;
}
.text-blue { color: #409eff; }
.text-green { color: #67c23a; }

.card-title {
  font-size: 18px;
  font-weight: 600;
}

.image-map {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.src-img, .dest-img {
  font-size: 13px;
  word-break: break-all;
}

.retry-text {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 5px;
  line-height: 1.4;
}

.preview-box {
  background-color: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 10px;
  max-height: 150px;
  overflow-y: auto;
  margin-bottom: 10px;
}
.preview-header {
  font-size: 13px;
  font-weight: bold;
  color: #606266;
  margin-bottom: 8px;
}
.preview-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.preview-item {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  border-bottom: 1px dashed #ebeef5;
  padding: 4px 0;
}
.preview-src {
  color: #606266;
  width: 50%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding-right: 10px;
}
.preview-dest {
  color: #409eff;
  width: 50%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-terminal-container {
  background-color: #1e1e1e;
  border-radius: 6px;
  padding: 15px;
}
.log-terminal {
  color: #d4d4d4;
  font-family: 'Courier New', Courier, monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  height: 60vh;
  overflow-y: auto;
}

/* 覆盖滚动条样式使其更好看 */
.log-terminal::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}
.log-terminal::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 4px;
}
.log-terminal::-webkit-scrollbar-track {
  background: #1e1e1e;
}
</style>
