<script lang="ts">
    import { UploadCloud, Folder, Trash2 } from 'lucide-svelte';
    import { goto } from '$app/navigation';
    import { page } from '$app/stores';
    import { jwtToken } from '$lib/stores/auth';

    function handleLogout() {
        jwtToken.set(null);
        goto('/');
    }
</script>

<div class="app-container">
    <header class="app-header">
        <div class="logo">
            <UploadCloud size={24} />
            <span>MyCloud</span>
        </div>
        <div class="user-menu">
            <button on:click={handleLogout}>Logout</button>
        </div>
    </header>

    <aside class="app-sidebar">
        <nav>
            <ul>
                <!-- ถ้าไม่มี ?view=trash, หรือ path ไม่ใช่ /trash, เมนูนี้จะ active -->
                <li class:active={!$page.url.searchParams.get('view') && $page.url.pathname.startsWith('/files')}>
                    <a href="/files"><Folder size={18} /><span>My Files</span></a>
                </li>
                <!-- ถ้ามี ?view=trash, เมนูนี้จะ active -->
                <li class:active={$page.url.searchParams.get('view') === 'trash'}>
                    <a href="/files?view=trash"><Trash2 size={18} /><span>Trash</span></a>
                </li>
            </ul>
        </nav>
    </aside>

    <main class="app-content">
        <slot />
    </main>
</div>

<style>
    :root { --bg-dark: #0f172a; --bg-sidebar: #f8fafc; --bg-content: #ffffff; --bg-hover: #f1f5f9; --border-color: #e2e8f0; --text-primary: #1e293b; --text-secondary: #64748b; --accent-color: #3b82f6; --font-sans: system-ui, sans-serif; }
    * { box-sizing: border-box; }
    :global(body) { margin: 0; font-family: var(--font-sans); }
    .app-container { display: grid; grid-template-columns: 240px 1fr; grid-template-rows: 60px 1fr; height: 100vh; grid-template-areas: 'header header' 'sidebar content'; }
    .app-header { grid-area: header; display: flex; align-items: center; justify-content: space-between; padding: 0 1.5rem; background: var(--bg-dark); color: white; border-bottom: 1px solid #334155;}
    .logo { display: flex; align-items: center; gap: 0.75rem; font-weight: 600; }
    .user-menu button { background: var(--accent-color); color: white; border: none; padding: 0.5rem 1rem; border-radius: 6px; cursor: pointer; font-weight: 500;}
    .app-sidebar { grid-area: sidebar; padding: 1rem; background: var(--bg-sidebar); border-right: 1px solid var(--border-color); }
    .app-sidebar nav ul { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 0.25rem; }
    .app-sidebar nav li { transition: background-color 0.2s; border-radius: 8px; color: var(--text-secondary); }
    .app-sidebar nav li.active { background-color: var(--bg-hover); font-weight: 500; color: var(--text-primary); }
    .app-sidebar nav li:not(.active):hover { background-color: #f8fafc; }
    .app-sidebar nav li a { display: flex; align-items: center; gap: 0.75rem; padding: 0.75rem 1rem; text-decoration: none; color: inherit; border-radius: 8px; }
    .app-content { grid-area: content; padding: 2rem; overflow-y: auto; background: var(--bg-content); }
</style>