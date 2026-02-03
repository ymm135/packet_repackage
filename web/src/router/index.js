import { createRouter, createWebHistory } from 'vue-router'
import NetworkConfig from '@/views/NetworkConfig.vue'
import NFTRules from '@/views/NFTRules.vue'
import Rules from '@/views/Rules.vue'
import TestMode from '@/views/TestMode.vue'
import Logs from '@/views/Logs.vue'

const routes = [
    {
        path: '/',
        redirect: '/network'
    },
    {
        path: '/network',
        name: 'Network',
        component: NetworkConfig
    },
    {
        path: '/nftrules',
        name: 'NFTRules',
        component: NFTRules
    },
    {
        path: '/rules',
        name: 'Rules',
        component: Rules
    },
    {
        path: '/test',
        name: 'Test',
        component: TestMode
    },
    {
        path: '/logs',
        name: 'Logs',
        component: Logs
    }
]

const router = createRouter({
    history: createWebHistory(),
    routes
})

export default router
