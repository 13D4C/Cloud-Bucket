import { goto } from '$app/navigation';
import { browser } from '$app/environment';

const API_BASE_URL = 'http://localhost:8080';

export async function fetchApi(path: string, options: RequestInit = {}): Promise<Response> {
    if (!browser) throw new Error('fetchApi can only be called on the client');
    const token = localStorage.getItem('jwt_token');
    if (!token) { await goto('/'); throw new Error('No token found'); }

    const headers = new Headers(options.headers || {});
    headers.set('Authorization', `Bearer ${token}`);
    if (options.body && !headers.has('Content-Type')) {
        headers.set('Content-Type', 'application/json');
    }
    const response = await fetch(`${API_BASE_URL}${path}`, { ...options, headers });
    if (response.status === 401) {
        localStorage.removeItem('jwt_token');
        await goto('/');
        throw new Error('Unauthorized');
    }
    return response;
}