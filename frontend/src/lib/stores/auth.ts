import { writable } from 'svelte/store';
import { browser } from '$app/environment';

// ฟังก์ชันนี้จะพยายามอ่าน token จาก localStorage ตอนที่แอปเริ่มทำงานครั้งแรก
function createTokenStore() {
    const initialValue = browser ? window.localStorage.getItem('jwt_token') : null;
    const { subscribe, set } = writable<string | null>(initialValue);

    return {
        subscribe,
        // set function ของเราจะทำการอัปเดตทั้ง store และ localStorage
        set: (value: string | null) => {
            if (browser) {
                if (value) {
                    window.localStorage.setItem('jwt_token', value);
                } else {
                    window.localStorage.removeItem('jwt_token');
                }
            }
            set(value);
        }
    };
}

export const jwtToken = createTokenStore();