import { goto } from '$app/navigation';
import { jwtToken } from '$lib/stores/auth';
import { get } from 'svelte/store';
import type { LayoutLoad } from './$types';
import { browser } from '$app/environment';

export const load: LayoutLoad = async () => {
    // ตรวจสอบว่าโค้ดกำลังทำงานฝั่งเบราว์เซอร์หรือไม่
    if (browser) {
        // อ่านค่าล่าสุดจาก store
        const token = get(jwtToken);

        if (!token) {
            // ถ้าไม่มี token ใน store ให้กลับไปหน้า login
            // ใช้ await เพื่อให้แน่ใจว่าการ redirect เกิดขึ้นก่อนที่จะทำอย่างอื่น
            await goto('/');
        }
    }
    
    // ถ้ามี token หรือไม่ได้ทำงานฝั่งเบราว์เซอร์ ก็อนุญาตให้เข้าไปได้
    return {}; 
};