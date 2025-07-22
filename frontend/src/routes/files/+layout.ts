import { browser } from '$app/environment';
import { goto } from '$app/navigation';
export const load = async () => {
    if (browser && !localStorage.getItem('jwt_token')) {
        await goto('/');
    }
};