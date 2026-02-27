<template>
  <div class="login-page">
    <div class="login-container glass-card">
      <div class="login-header">
        <div class="logo-icon">
          <el-icon :size="32"><Platform /></el-icon>
        </div>
        <h1>AI Gateway</h1>
        <p>统一AI服务管理平台</p>
      </div>

      <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" class="login-form">
        <el-form-item prop="username">
          <el-input
            v-model="loginForm.username"
            placeholder="用户名"
            prefix-icon="User"
            size="large"
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="密码"
            prefix-icon="Lock"
            size="large"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item>
          <div class="login-options">
            <el-checkbox v-model="loginForm.remember">记住我</el-checkbox>
            <el-link type="primary">忘记密码?</el-link>
          </div>
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            class="login-btn login-button"
            :loading="loading"
            @click="handleLogin"
            native-type="submit"
          >
            登录
          </el-button>
        </el-form-item>
      </el-form>

      <div class="login-footer">
        <p>AI Gateway v1.0.0</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useUserStore } from '@/store/user'
import { request } from '@/api/request'
import { LOGIN_SUCCESS_REDIRECT } from '@/constants/navigation'

const router = useRouter()
const userStore = useUserStore()
const loginFormRef = ref<FormInstance>()
const loading = ref(false)

const loginForm = reactive({
  username: '',
  password: '',
  remember: false
})

const loginRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ]
}

const handleLogin = async () => {
  const valid = await loginFormRef.value?.validate().catch(() => false)
  if (!valid) return

  loading.value = true

  try {
    const res = await request.post<{ token: string; user: { id: string; username: string; role: string } }>('/auth/login', {
      username: loginForm.username,
      password: loginForm.password
    })

    // 保存 token
    userStore.setToken(res.token)
    userStore.setUserInfo({
      id: Number(res.user.id),
      username: res.user.username,
      email: '',
      role: res.user.role
    })

    ElMessage.success('登录成功')
    router.push(LOGIN_SUCCESS_REDIRECT)
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.error?.message || error?.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="scss">
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--color-primary) 0%, #5856D6 50%, #AF52DE 100%);
  position: relative;
  overflow: hidden;

  // 背景装饰
  &::before {
    content: '';
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    background: radial-gradient(circle, rgba(255,255,255,0.1) 0%, transparent 60%);
    animation: rotate 30s linear infinite;
  }

  @keyframes rotate {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .login-container {
    width: 420px;
    padding: 48px 40px;
    background: var(--bg-primary);
    border-radius: var(--border-radius-xl);
    box-shadow: 0 25px 80px rgba(0, 0, 0, 0.25);
    position: relative;
    z-index: 1;

    .login-header {
      text-align: center;
      margin-bottom: 36px;

      .logo-icon {
        width: 72px;
        height: 72px;
        margin: 0 auto 20px;
        background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
        border-radius: var(--border-radius-xl);
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
        box-shadow: 0 8px 24px rgba(0, 122, 255, 0.3);
      }

      h1 {
        margin: 0 0 8px;
        font-size: var(--font-size-2xl);
        font-weight: var(--font-weight-bold);
        color: var(--text-primary);
      }

      p {
        color: var(--text-secondary);
        margin: 0;
        font-size: var(--font-size-md);
      }
    }

    .login-form {
      .login-options {
        width: 100%;
        display: flex;
        justify-content: space-between;
        align-items: center;
      }

      .login-btn {
        width: 100%;
        height: 48px;
        font-size: var(--font-size-lg);
        font-weight: var(--font-weight-medium);
        border-radius: var(--border-radius-lg);
      }
    }

    .login-footer {
      text-align: center;
      margin-top: 32px;
      color: var(--text-tertiary);
      font-size: var(--font-size-sm);
    }
  }
}

// 暗色模式适配
[data-theme="dark"] {
  .login-page {
    .login-container {
      background: var(--bg-secondary);
      border: 1px solid var(--border-primary);
    }
  }
}
</style>
