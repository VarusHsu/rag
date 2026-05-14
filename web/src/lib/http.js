const REQUEST_ID_HEADER = 'X-Request-Id'

function randomHex(bytes) {
  const arr = new Uint8Array(bytes)
  crypto.getRandomValues(arr)
  return Array.from(arr, (b) => b.toString(16).padStart(2, '0')).join('')
}

export function generateRequestId() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    return randomHex(16)
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export function createHttpClient(baseURL) {
  const requestInterceptors = []
  const responseInterceptors = []

  async function request(path, options = {}) {
    let req = {
      url: `${baseURL}${path}`,
      options: {
        method: options.method || 'GET',
        headers: {
          ...(options.headers || {})
        },
        body: options.body
      }
    }

    for (const interceptor of requestInterceptors) {
      req = (await interceptor(req)) || req
    }

    const response = await fetch(req.url, req.options)
    let data = null
    try {
      data = await response.json()
    } catch {
      data = null
    }

    let result = {
      response,
      data,
      requestId: response.headers.get(REQUEST_ID_HEADER) || req.options.headers[REQUEST_ID_HEADER] || ''
    }

    for (const interceptor of responseInterceptors) {
      result = (await interceptor(result)) || result
    }

    if (!result.response.ok) {
      const message = result.data?.error || `Request failed with status ${result.response.status}`
      const error = new Error(message)
      error.status = result.response.status
      error.requestId = result.requestId
      error.data = result.data
      throw error
    }

    return result
  }

  return {
    useRequest(interceptor) {
      requestInterceptors.push(interceptor)
    },
    useResponse(interceptor) {
      responseInterceptors.push(interceptor)
    },
    get(path, options = {}) {
      return request(path, { ...options, method: 'GET' })
    },
    post(path, body, options = {}) {
      return request(path, {
        ...options,
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(options.headers || {})
        },
        body: JSON.stringify(body)
      })
    }
  }
}

export function withRequestID(req) {
  const requestId = req.options.headers[REQUEST_ID_HEADER] || generateRequestId()
  return {
    ...req,
    options: {
      ...req.options,
      headers: {
        ...req.options.headers,
        [REQUEST_ID_HEADER]: requestId
      }
    }
  }
}

export { REQUEST_ID_HEADER }

