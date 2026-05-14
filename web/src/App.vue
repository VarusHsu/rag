<script setup>
import { computed, ref } from 'vue'
import { createHttpClient, withRequestID } from './lib/http'

const apiBase = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

const activeTab = ref('login')
const loading = ref(false)
const uploading = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const lastRequestId = ref('')
const token = ref(localStorage.getItem('auth_token') || '')
const user = ref(loadUser())
const activePage = ref(token.value ? 'upload' : 'auth')

function clearAuthState() {
  token.value = ''
  user.value = null
  localStorage.removeItem('auth_token')
  localStorage.removeItem('auth_user')
  activePage.value = 'auth'
}

const http = createHttpClient(apiBase, {
  onUnauthorized: (err) => {
    clearAuthState()
    errorMsg.value = 'Login expired, please login again'
    lastRequestId.value = err?.requestId || ''
  }
})

http.useRequest((req) => {
  const next = withRequestID(req)
  if (token.value) {
    next.options.headers = {
      ...next.options.headers,
      Authorization: `Bearer ${token.value}`
    }
  }
  return next
})

const loginForm = ref({
  email: '',
  password: ''
})

const registerForm = ref({
  username: '',
  email: '',
  phone: '',
  password: ''
})

const selectedFile = ref(null)
const uploadInfo = ref(null)

const isLoggedIn = computed(() => Boolean(token.value))

function loadUser() {
  try {
    const raw = localStorage.getItem('auth_user')
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

function clearMessage() {
  errorMsg.value = ''
  successMsg.value = ''
  lastRequestId.value = ''
}

async function submitLogin() {
  clearMessage()
  loading.value = true

  try {
    const { data, requestId } = await http.post('/api/v1/auth/login', loginForm.value)
    lastRequestId.value = requestId || data?.request_id || ''

    token.value = data.token
    user.value = data.user
    localStorage.setItem('auth_token', data.token)
    localStorage.setItem('auth_user', JSON.stringify(data.user))
    successMsg.value = `Welcome back, ${data.user.username}!`
    activePage.value = 'upload'
  } catch (err) {
    errorMsg.value = err.message || 'Login failed'
    lastRequestId.value = err.requestId || ''
  } finally {
    loading.value = false
  }
}

async function submitRegister() {
  clearMessage()
  loading.value = true

  const payload = {
    ...registerForm.value,
    phone: registerForm.value.phone || null
  }

  try {
    const { data, requestId } = await http.post('/api/v1/auth/register', payload)
    lastRequestId.value = requestId || data?.request_id || ''

    token.value = data.token
    user.value = data.user
    localStorage.setItem('auth_token', data.token)
    localStorage.setItem('auth_user', JSON.stringify(data.user))
    successMsg.value = `Register success, ${data.user.username}`
    activePage.value = 'upload'
  } catch (err) {
    errorMsg.value = err.message || 'Register failed'
    lastRequestId.value = err.requestId || ''
  } finally {
    loading.value = false
  }
}

function onFileChange(event) {
  const file = event.target.files?.[0] || null
  selectedFile.value = file
  uploadInfo.value = null
}

async function submitUpload() {
  clearMessage()
  uploadInfo.value = null

  if (!selectedFile.value) {
    errorMsg.value = 'Please choose a file first'
    return
  }

  uploading.value = true
  try {
    const file = selectedFile.value
    const { data, requestId } = await http.post(
      '/api/v1/files/presign-upload',
      {
        file_name: file.name,
        content_type: file.type || 'application/octet-stream',
        file_size: file.size
      }
    )
    lastRequestId.value = requestId || ''

    const putRes = await fetch(data.upload_url, {
      method: data.upload_method || 'PUT',
      headers: {
        'Content-Type': file.type || 'application/octet-stream'
      },
      body: file
    })
    if (!putRes.ok) {
      throw new Error(`Object upload failed with status ${putRes.status}`)
    }

    uploadInfo.value = data
    successMsg.value = 'File uploaded successfully'
  } catch (err) {
    errorMsg.value = err.message || 'Upload failed'
    lastRequestId.value = err.requestId || lastRequestId.value
  } finally {
    uploading.value = false
  }
}

async function logout() {
  clearMessage()
  loading.value = true

  try {
    const { data, requestId } = await http.post('/api/v1/auth/logout', {})
    lastRequestId.value = requestId || data?.request_id || ''
    successMsg.value = data?.message || 'Logout success'
  } catch (err) {
    errorMsg.value = err.message || 'Logout failed'
    lastRequestId.value = err.requestId || ''
  } finally {
    selectedFile.value = null
    uploadInfo.value = null
    clearAuthState()
    loading.value = false
  }
}
</script>

<template>
  <main class="container">
    <section class="card">
      <h1>Gateway Auth</h1>
      <p class="sub">Vue + Gin + PostgreSQL</p>

      <p v-if="errorMsg" class="alert error">{{ errorMsg }}</p>
      <p v-if="successMsg" class="alert success">{{ successMsg }}</p>
      <p v-if="lastRequestId" class="trace-id">request_id: {{ lastRequestId }}</p>

      <template v-if="activePage === 'upload' && isLoggedIn && user">
        <div class="profile">
          <p><strong>User:</strong> {{ user.username }}</p>
          <p><strong>Email:</strong> {{ user.email }}</p>
          <p><strong>Role:</strong> {{ user.role }}</p>
          <p><strong>Status:</strong> {{ user.status }}</p>
        </div>

        <div class="upload-panel">
          <h3>Upload File</h3>
          <label>
            Choose File
            <input type="file" @change="onFileChange" />
          </label>
          <button class="btn" :disabled="uploading" @click="submitUpload">
            {{ uploading ? 'Uploading...' : 'Upload' }}
          </button>
        </div>

        <div v-if="uploadInfo" class="upload-result">
          <p><strong>File ID:</strong> {{ uploadInfo.file_id }}</p>
          <p><strong>Bucket:</strong> {{ uploadInfo.bucket }}</p>
          <p><strong>Object Key:</strong> {{ uploadInfo.object_key }}</p>
        </div>

        <button class="btn btn-danger" :disabled="loading" @click="logout">Logout</button>
      </template>

      <template v-else>
        <div class="tabs">
          <button
            class="tab"
            :class="{ active: activeTab === 'login' }"
            @click="activeTab = 'login'"
          >
            Login
          </button>
          <button
            class="tab"
            :class="{ active: activeTab === 'register' }"
            @click="activeTab = 'register'"
          >
            Register
          </button>
        </div>

        <form v-if="activeTab === 'login'" class="form" @submit.prevent="submitLogin">
          <label>
            Email
            <input v-model="loginForm.email" type="email" required />
          </label>

          <label>
            Password
            <input v-model="loginForm.password" type="password" required />
          </label>

          <button class="btn" type="submit" :disabled="loading">
            {{ loading ? 'Submitting...' : 'Login' }}
          </button>
        </form>

        <form v-else class="form" @submit.prevent="submitRegister">
          <label>
            Username
            <input v-model="registerForm.username" type="text" required />
          </label>

          <label>
            Email
            <input v-model="registerForm.email" type="email" required />
          </label>

          <label>
            Phone (optional)
            <input v-model="registerForm.phone" type="tel" />
          </label>

          <label>
            Password
            <input v-model="registerForm.password" type="password" minlength="8" required />
          </label>

          <button class="btn" type="submit" :disabled="loading">
            {{ loading ? 'Submitting...' : 'Register' }}
          </button>
        </form>
      </template>
    </section>
  </main>
</template>
