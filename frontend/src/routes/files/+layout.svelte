<script lang="ts">
    import { UploadCloud, Folder, Trash2 } from 'lucide-svelte';
    import { goto } from '$app/navigation';
    import { page } from '$app/stores';
</script>

<div class="app-container">
    <header class="app-header">
        <div class="logo">
            <UploadCloud size={24} />
            <span>MyCloud</span>
        </div>
        <button class="logout-btn" on:click={() => {localStorage.removeItem('jwt_token'); goto('/');}}>
            Logout
        </button>
    </header>

    <aside class="app-sidebar">
        <nav>
            <ul>
                <li class:active={$page.url.pathname === '/files'}>
                    <a href="/files"><Folder size={18} /><span>My Files</span></a>
                </li>
                <li class:active={$page.url.pathname.startsWith('/files/trash')}>
                    <a href="/files/trash"><Trash2 size={18} /><span>Trash</span></a>
                </li>
            </ul>
        </nav>
    </aside>

    <main class="app-content">
        <!-- Page content (e.g., file list or trash list) goes here -->
        <slot />
    </main>
</div>

<style>
    :root {
        --bg-dark: #0f172a;
        --bg-sidebar: #f8fafc;
        --bg-content: #ffffff;
        --bg-hover: #f1f5f9;
        --border-color: #e2e8f0;
        --text-primary: #1e293b;
        --text-secondary: #64748b;
        --accent-color: #0d6efd;
    }
    
    .app-container {
        display: grid;
        grid-template-columns: 240px 1fr;
        grid-template-rows: 60px 1fr;
        height: 100vh;
        grid-template-areas:
            'header header'
            'sidebar content';
    }

    .app-header {
        grid-area: header;
        background-color: var(--bg-dark);
        color: white;
        display: flex;
        align-items: center;
        padding: 0 1.5rem;
        justify-content: space-between;
        border-bottom: 1px solid #334155;
    }

    .logo {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        font-size: 1.125rem;
        font-weight: 600;
    }

    .logout-btn {
        background: var(--accent-color);
        color: white;
        border: none;
        padding: 0.5rem 1rem;
        border-radius: 6px;
        cursor: pointer;
        font-weight: 500;
        transition: background-color 0.2s;
    }

    .logout-btn:hover {
        background-color: #0b5ed7;
    }

    .app-sidebar {
        grid-area: sidebar;
        background-color: var(--bg-sidebar);
        border-right: 1px solid var(--border-color);
        padding: 1rem;
    }

    .app-sidebar nav {
        width: 100%;
    }

    .app-sidebar ul {
        list-style: none;
        padding: 0;
        margin: 0;
    }
    
    .app-sidebar li {
        margin-bottom: 0.25rem;
    }

    .app-sidebar li a {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        padding: 0.75rem 1rem;
        border-radius: 8px;
        font-weight: 500;
        color: var(--text-secondary);
        text-decoration: none;
        transition: all 0.2s;
    }

    .app-sidebar li:hover a {
        background-color: var(--bg-hover);
        color: var(--text-primary);
    }
    
    .app-sidebar li.active a {
        background-color: #E7F1FF;
        color: var(--accent-color);
    }

    .app-content {
        grid-area: content;
        padding: 2rem;
        overflow-y: auto;
        background-color: var(--bg-content);
    }
</style>