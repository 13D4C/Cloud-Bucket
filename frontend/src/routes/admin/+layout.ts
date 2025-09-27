// // src/routes/admin/+layout.ts
import { goto } from '$app/navigation';
import { jwtToken } from '$lib/stores/auth';
import { get } from 'svelte/store';
import type { LayoutLoad } from './$types';
import { browser } from '$app/environment';

export const load: LayoutLoad = async () => {
    // Only run checks in the browser
    if (browser) {
        const token = get(jwtToken);
    
        if (!token) {
            await goto('/');
            return {};
        }
    
        try {
            const payload = JSON.parse(atob(token.split('.')[1]));
            const userRole = payload.role || 'User';
        
            if (userRole !== 'Admin') {
                await goto('/files');
                return {};
            }
        
            return {
                user: {
                    username: payload.sub,
                    role: userRole
                }
            };
        } catch (error) {
            console.error('Invalid token:', error);
            await goto('/');
            return {};
        }
    }

    return {};
};