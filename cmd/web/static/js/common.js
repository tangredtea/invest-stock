/**
 * Invest 量化终端 - 核心 JavaScript
 */
(function() {
    'use strict';

    const CONFIG = {
        API_BASE_URL: '',
        TOAST_DURATION: 3000,
        THEME_KEY: 'theme',
        TOKEN_KEY: 'token',
        USER_KEY: 'user'
    };

    // ========== API 模块 ==========
    const API = {
        getToken() {
            return localStorage.getItem(CONFIG.TOKEN_KEY);
        },

        async request(method, url, data) {
            const headers = { 'Content-Type': 'application/json' };
            const token = this.getToken();
            if (token) headers['Authorization'] = 'Bearer ' + token;

            const opts = { method, headers };
            if (data) opts.body = JSON.stringify(data);

            const resp = await fetch(CONFIG.API_BASE_URL + url, opts);

            if (resp.status === 401) {
                localStorage.removeItem(CONFIG.TOKEN_KEY);
                localStorage.removeItem(CONFIG.USER_KEY);
                location.href = '/login';
                throw new Error('未登录');
            }

            const json = await resp.json();
            if (json.code !== 0 && json.code !== undefined) {
                throw new Error(json.message || '请求失败');
            }
            return json;
        },

        get(url) { return this.request('GET', url); },
        post(url, data) { return this.request('POST', url, data); },
        put(url, data) { return this.request('PUT', url, data); },
        del(url) { return this.request('DELETE', url); }
    };

    // ========== Toast 模块 ==========
    const Toast = {
        show(message, type = 'info') {
            const container = document.getElementById('toast-container');
            if (!container) return;
            const el = document.createElement('div');
            el.className = `toast toast-${type}`;
            el.textContent = message;
            container.appendChild(el);
            setTimeout(() => {
                el.classList.add('removing');
                setTimeout(() => el.remove(), 300);
            }, CONFIG.TOAST_DURATION);
        },
        success(msg) { this.show(msg, 'success'); },
        error(msg) { this.show(msg, 'error'); },
        warning(msg) { this.show(msg, 'warning'); },
        info(msg) { this.show(msg, 'info'); }
    };

    // ========== Theme 模块 ==========
    const Theme = {
        init() {
            const t = localStorage.getItem(CONFIG.THEME_KEY);
            if (t === 'dark' || (!t && window.matchMedia('(prefers-color-scheme:dark)').matches)) {
                document.documentElement.classList.add('dark');
            }
        },
        toggle() {
            const isDark = document.documentElement.classList.toggle('dark');
            localStorage.setItem(CONFIG.THEME_KEY, isDark ? 'dark' : 'light');
            return isDark;
        },
        isDark() {
            return document.documentElement.classList.contains('dark');
        }
    };

    // ========== AppLayout (Alpine.js) ==========
    function AppLayout() {
        return {
            sidebarOpen: false,
            sidebarCollapsed: false,
            isDark: Theme.isDark(),
            currentUser: null,

            init() {
                Theme.init();
                this.isDark = Theme.isDark();
                try {
                    this.currentUser = JSON.parse(localStorage.getItem(CONFIG.USER_KEY));
                } catch(e) {
                    this.currentUser = null;
                }
            },

            toggleTheme() {
                this.isDark = Theme.toggle();
            },

            logout() {
                localStorage.removeItem(CONFIG.TOKEN_KEY);
                localStorage.removeItem(CONFIG.USER_KEY);
                location.href = '/login';
            },

            isCurrentPath(path) {
                return location.pathname === path;
            }
        };
    }

    // ========== 全局导出 ==========
    window.API = API;
    window.Toast = Toast;
    window.Theme = Theme;
    window.AppLayout = AppLayout;

    // 页面加载完成后隐藏 loader
    document.addEventListener('DOMContentLoaded', () => {
        const loader = document.getElementById('page-loader');
        if (loader) {
            loader.style.opacity = '0';
            setTimeout(() => loader.remove(), 300);
        }
    });
})();
