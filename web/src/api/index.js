import axios from 'axios'

const api = axios.create({
    baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
    timeout: 10000
})

// Network APIs
export const networkAPI = {
    listInterfaces: () => api.get('/interfaces'),
    getInterface: (name) => api.get(`/interfaces/${name}`),
    configureVLAN: (data) => api.post('/vlan', data),
    getVLANConfig: (interface_name) => api.get(`/vlan/${interface_name}`),
    addVLANIP: (data) => api.post('/vlan/ip', data),
    removeVLANIP: (interface_name) => api.delete(`/vlan/ip/${interface_name}`),
    setInterfaceStatus: (data) => api.post('/interface/status', data)
}

// Field APIs
export const fieldAPI = {
    list: () => api.get('/fields'),
    get: (id) => api.get(`/fields/${id}`),
    create: (data) => api.post('/fields', data),
    update: (id, data) => api.put(`/fields/${id}`, data),
    delete: (id) => api.delete(`/fields/${id}`)
}

// Rule APIs
export const ruleAPI = {
    list: () => api.get('/rules'),
    get: (id) => api.get(`/rules/${id}`),
    create: (data) => api.post('/rules', data),
    update: (id, data) => api.put(`/rules/${id}`, data),
    delete: (id) => api.delete(`/rules/${id}`),
    toggle: (id) => api.post(`/rules/${id}/toggle`)
}

// NFT Rule APIs
export const nftRuleAPI = {
    list: () => api.get('/nftrules'),
    get: (id) => api.get(`/nftrules/${id}`),
    create: (data) => api.post('/nftrules', data),
    update: (id, data) => api.put(`/nftrules/${id}`, data),
    delete: (id) => api.delete(`/nftrules/${id}`),
    toggle: (id) => api.post(`/nftrules/${id}/toggle`),
    apply: () => api.post('/nftrules/apply')
}

// Test API
export const testAPI = {
    test: (data) => api.post('/test', data)
}

// Log APIs
export const logAPI = {
    list: (params) => api.get('/logs', { params }),
    get: (id) => api.get(`/logs/${id}`),
    clear: (days) => api.delete('/logs', { params: { days } })
}

export default api
