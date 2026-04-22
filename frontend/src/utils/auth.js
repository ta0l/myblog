// frontend/src/utils/auth.js

// 内存中的鉴权缓存状态
let authCache = {
    isValid: false,
    lastCheckTime: 0
};

// 缓存有效期（例如：5分钟，单位毫秒）
// 在这 5 分钟内切换任何页面，都不会重复发起 HTTP 请求
const CACHE_DURATION = 5 * 60 * 1000; 

/**
 * 简易解析 JWT 载荷 (Payload) 
 * 浏览器端无需密钥即可解开 JWT 的中间段以读取公共信息（如过期时间）
 */
function parseJwtPayload(token) {
    try {
        const base64Url = token.split('.')[1];
        const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
        const jsonPayload = decodeURIComponent(window.atob(base64).split('').map(function(c) {
            return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
        }).join(''));
        return JSON.parse(jsonPayload);
    } catch (e) {
        return null;
    }
}

export async function checkAuth() {
    const token = localStorage.getItem('token');
    
    // 1. 如果根本没有 Token，直接判定为未登录
    if (!token) {
        authCache.isValid = false;
        return { isValid: false, token: null };
    }

    // 2. 本地解析 JWT，判断是否过期 (零延迟，无网络开销)
    const payload = parseJwtPayload(token);
    if (!payload || !payload.exp) {
        localStorage.removeItem('token');
        return { isValid: false, token: null };
    }

    const currentUnixTime = Math.floor(Date.now() / 1000); // 转换为秒
    if (currentUnixTime >= payload.exp) {
        console.log("🔒 Token 本地校验已过期");
        localStorage.removeItem('token');
        authCache.isValid = false;
        return { isValid: false, token: null };
    }

    // 3. 检查内存缓存是否还在有效期内
    const now = Date.now();
    if (authCache.isValid && (now - authCache.lastCheckTime < CACHE_DURATION)) {
        console.log("⚡ 使用本地鉴权缓存，跳过网络请求");
        return { isValid: true, token };
    }

    // 4. 缓存过期或首次加载，向服务器发起真实的静默校验
    console.log("🌐 向服务器发起 Token 真实校验...");
    try {
        const res = await fetch('http://localhost:8888/api/auth/verify', {
            method: 'GET',
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (res.ok) {
            // 校验成功，更新缓存状态和时间
            authCache = {
                isValid: true,
                lastCheckTime: now
            };
            return { isValid: true, token };
        } else {
            // 服务器说 Token 假冒或失效，立即清理
            localStorage.removeItem('token');
            authCache.isValid = false;
            return { isValid: false, token: null };
        }
    } catch (error) {
        console.error("鉴权服务连接失败:", error);
        // 如果是网络断开，为了安全起见，暂时视为未登录，但不清空 Token
        return { isValid: false, token: null };
    }
}

/**
 * 供退出登录时调用的清理函数
 */
export function clearAuth() {
    localStorage.removeItem('token');
    authCache.isValid = false;
    authCache.lastCheckTime = 0;
}