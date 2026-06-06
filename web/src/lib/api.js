import { generateUUID } from './utils';

// Helper to handle API responses and RFC 7807 problem details
async function handleResponse(response) {
  if (response.status === 204) {
    return null;
  }

  const contentType = response.headers.get('content-type');
  let data = null;
  if (contentType && contentType.includes('application/json')) {
    data = await response.json();
  } else if (contentType && contentType.includes('application/problem+json')) {
    data = await response.json();
  }

  if (!response.ok) {
    if (data && data.detail) {
      let errorMessage = data.detail;
      if (data.invalid_params && data.invalid_params.length > 0) {
        errorMessage += ':\n' + data.invalid_params.map(p => `- ${p.name}: ${p.reason}`).join('\n');
      }
      throw new Error(errorMessage);
    }
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }

  return data;
}

// Base fetch wrapper
async function apiFetch(endpoint, options = {}) {
  const defaultHeaders = {
    'Content-Type': 'application/json',
    'X-Correlation-Id': generateUUID(),
  };

  const config = {
    ...options,
    headers: {
      ...defaultHeaders,
      ...options.headers,
    },
    // This is critical: it ensures the session_token cookie is sent with the request to the proxy
    credentials: 'include',
  };

  // If we're passing FormData, let the browser set the Content-Type with the boundary
  if (options.body instanceof FormData) {
    delete config.headers['Content-Type'];
  }

  try {
    const response = await fetch(`/v1${endpoint}`, config);
    return await handleResponse(response);
  } catch (error) {
    console.error(`API Fetch Error (${endpoint}):`, error);
    throw error;
  }
}

// ---------------------------------------------------------------------------
// Authentication
// ---------------------------------------------------------------------------

export const login = (email, password) => {
  return apiFetch('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
};

export const signup = (email, password) => {
  return apiFetch('/auth/signup', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
};

export const logout = () => {
  return apiFetch('/auth/logout', {
    method: 'POST',
  });
};

export const getMe = () => {
  return apiFetch('/auth/me', {
    method: 'GET',
  });
};

// ---------------------------------------------------------------------------
// Accounts
// ---------------------------------------------------------------------------

export const getAccounts = (cursor = 0, limit = 20) => {
  const url = cursor ? `/accounts?cursor=${cursor}&page_size=${limit}` : `/accounts?page_size=${limit}`;
  return apiFetch(url, { method: 'GET' });
};

export const getAccount = (id) => {
  return apiFetch(`/accounts/${id}`, { method: 'GET' });
};

export const getAccountBalance = (id) => {
  return apiFetch(`/accounts/${id}/balance`, { method: 'GET' });
};

export const createAccount = (data) => {
  return apiFetch('/accounts', {
    method: 'POST',
    body: JSON.stringify(data),
  });
};

// ---------------------------------------------------------------------------
// Transactions
// ---------------------------------------------------------------------------

export const getTransactions = (cursor = '', limit = 20) => {
  const url = cursor ? `/transactions?cursor=${encodeURIComponent(cursor)}&page_size=${limit}` : `/transactions?page_size=${limit}`;
  return apiFetch(url, { method: 'GET' });
};

export const getTransaction = (id) => {
  return apiFetch(`/transactions/${id}`, { method: 'GET' });
};

export const createTransaction = (data, idempotencyKey) => {
  return apiFetch('/transactions', {
    method: 'POST',
    headers: {
      'Idempotency-Key': idempotencyKey || generateUUID(),
    },
    body: JSON.stringify(data),
  });
};

// ---------------------------------------------------------------------------
// Documents
// ---------------------------------------------------------------------------

export const uploadDocument = (transactionId, file) => {
  const formData = new FormData();
  formData.append('file', file);
  
  return apiFetch(`/transactions/${transactionId}/documents`, {
    method: 'POST',
    body: formData,
  });
};

export const getDocuments = (transactionId) => {
  return apiFetch(`/transactions/${transactionId}/documents`, { method: 'GET' });
};

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

export const getHealth = () => {
  return apiFetch('/health', { method: 'GET' });
};
