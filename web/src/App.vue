<script setup>
import { computed, ref } from 'vue'

const apiBase = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

const activeTab = ref('login')
const loading = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const token = ref(localStorage.getItem('auth_token') || '')
const user = ref(loadUser())

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
}

async function submitLogin() {
  clearMessage()
  loading.value = true

  try {
    const res = await fetch(`${apiBase}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(loginForm.value)
    })

    const data = await res.json()
    if (!res.ok) {
      throw new Error(data.error || 'Login failed')
    }

    token.value = data.token
    user.value = data.user
    localStorage.setItem('auth_token', data.token)
    localStorage.setItem('auth_user', JSON.stringify(data.user))
    successMsg.value = `Welcome back, ${data.user.username}!`
  } catch (err) {
    errorMsg.value = err.message || 'Login failed'
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
    const res = await fetch(`${apiBase}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })

    const data = await res.json()
    if (!res.ok) {
      throw new Error(data.error || 'Register failed')
    }

    token.value = data.token
    user.value = data.user
    localStorage.setItem('auth_token', data.token)
    localStorage.setItem('auth_user', JSON.stringify(data.user))
    successMsg.value = `Register success, ${data.user.username}`
    activeTab.value = 'login'
  } catch (err) {
    errorMsg.value = err.message || 'Register failed'
  } finally {
    loading.value = false
  }
}

function logout() {
  token.value = ''
  user.value = null
  localStorage.removeItem('auth_token')
  localStorage.removeItem('auth_user')
  clearMessage()
}
</script>

<template>
  <main class="container">
    <section class="card">
      <h1>Gateway Auth</h1>
      <p class="sub">Vue + Gin + PostgreSQL</p>

      <p v-if="errorMsg" class="alert error">{{ errorMsg }}</p>
      <p v-if="successMsg" class="alert success">{{ successMsg }}</p>

      <template v-if="isLoggedIn && user">
        <div class="profile">
          <p><strong>User:</strong> {{ user.username }}</p>
          <p><strong>Email:</strong> {{ user.email }}</p>
          <p><strong>Role:</strong> {{ user.role }}</p>
          <p><strong>Status:</strong> {{ user.status }}</p>
        </div>
        <button class="btn" @click="logout">Logout</button>
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

