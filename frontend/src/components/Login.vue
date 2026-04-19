<template>
  <div class="login-container">
    <el-card class="login-card" shadow="always">
      <template #header>
        <h2 class="login-title">Skopeo 同步管理系统</h2>
      </template>
      <el-form :model="form" @submit.prevent="handleLogin" size="large">
        <el-form-item>
          <el-input 
            v-model="form.username" 
            placeholder="用户名" 
            clearable 
          >
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-input 
            v-model="form.password" 
            type="password" 
            placeholder="密码" 
            show-password 
            clearable
          >
          </el-input>
        </el-form-item>
        <el-form-item>
          <div style="display: flex; gap: 10px; width: 100%;">
            <el-input 
              v-model="form.captcha_value" 
              placeholder="请输入验证码" 
              clearable 
              style="flex: 1;"
            >
            </el-input>
            <img 
              v-if="captchaImg" 
              :src="captchaImg" 
              @click="fetchCaptcha" 
              class="captcha-img" 
              title="点击刷新验证码"
            />
          </div>
        </el-form-item>
        <el-button type="primary" native-type="submit" class="login-btn" :loading="loading">
          登 录
        </el-button>
      </el-form>
      <el-alert v-if="error" :title="error" type="error" show-icon style="margin-top: 20px;" :closable="false" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import axios from 'axios'

const emit = defineEmits(['login-success'])

const form = reactive({
  username: '',
  password: '',
  captcha_id: '',
  captcha_value: ''
})
const error = ref('')
const loading = ref(false)
const captchaImg = ref('')

const fetchCaptcha = async () => {
  try {
    const res = await axios.get('/api/captcha')
    if (res.data && res.data.captcha_id) {
      form.captcha_id = res.data.captcha_id
      captchaImg.value = res.data.captcha_img
    }
  } catch (err) {
    console.error('Failed to fetch captcha', err)
  }
}

onMounted(() => {
  fetchCaptcha()
})

const handleLogin = async () => {
  if (!form.username || !form.password || !form.captcha_value) {
    error.value = '请填写完整登录信息'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await axios.post('/api/login', form)
    if (res.data && res.data.token) {
      emit('login-success', res.data.token)
    }
  } catch (err: any) {
    error.value = err.response?.data?.error || '登录失败，请检查网络或后端状态'
    fetchCaptcha() // 登录失败刷新验证码
    form.captcha_value = ''
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: #f0f2f5;
  background-image: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
}

.login-card {
  width: 420px;
  border-radius: 8px;
}

.login-title {
  text-align: center;
  margin: 0;
  color: #303133;
  font-size: 22px;
}

.login-btn {
  width: 100%;
  margin-top: 10px;
}

.captcha-img {
  height: 40px;
  cursor: pointer;
  border-radius: 4px;
  border: 1px solid #dcdfe6;
}
</style>
