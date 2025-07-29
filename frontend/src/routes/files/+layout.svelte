<script lang="ts">
	import { goto } from '$app/navigation';
	import { Home, Trash2, Cloud } from 'lucide-svelte';
    import { page } from '$app/stores';

	function handleLogout() {
		localStorage.removeItem('jwt_token');
		goto('/');
	}

    $: isTrash = $page.url.searchParams.get('view') === 'trash';
</script>

<div class="app-container">
	<aside class="sidebar">
		<div class="logo">
			<Cloud size={28} />
			<span>IT-Cloud</span>
		</div>
		<nav class="main-nav">
			<a href="/files" class:active={!isTrash}>
				<Home size={20} />
				<span>My Files</span>
			</a>
			<a href="/files?view=trash" class:active={isTrash}>
				<Trash2 size={20} />
				<span>Trash</span>
			</a>
		</nav>
	</aside>

	<main class="main-content">
		<header class="main-header">
			<div class="header-actions">
				<button on:click={handleLogout} class="logout-btn">Logout</button>
			</div>
		</header>
		<div class="page-content">
			<slot />
		</div>
	</main>
</div>

<style>
    :root {
        --bg-sidebar: #1f2937;
        --bg-main: #111827;   
        --bg-header: #1f2937;
        --border-color: #374151;
        --text-primary: #f9fafb;
        --text-secondary: #d1d5db;
        --text-muted: #9ca3af;
        --accent-primary: #ef4444; 
        --accent-primary-hover: #dc2626; 
        --nav-hover-bg: #374151;
        --nav-active-bg: rgba(239, 68, 68, 0.15);
        --nav-active-text: var(--accent-primary);
        --nav-active-border: var(--accent-primary);
    }

    .app-container {
        display: flex;
        height: 100vh;
        background-color: var(--bg-main);
    }

    .sidebar {
        width: 240px;
        background-color: var(--bg-sidebar);
        border-right: 1px solid var(--border-color);
        display: flex;
        flex-direction: column;
        padding: 1.5rem 1rem;
        flex-shrink: 0;
    }

    .logo {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        font-size: 1.5rem;
        font-weight: 700;
        margin-bottom: 2rem;
        padding: 0 0.5rem;
        color: var(--text-primary);
    }

    .main-nav a {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        padding: 0.75rem 1rem;
        border-radius: 8px;
        text-decoration: none;
        color: var(--text-secondary);
        font-weight: 500;
        margin-bottom: 0.5rem;
        transition: background-color 0.2s ease, color 0.2s ease;
    }

    .main-nav a:hover {
        background-color: var(--nav-hover-bg);
        color: var(--text-primary);
    }

    .main-nav a.active {
        background-color: var(--nav-active-bg);
        color: var(--nav-active-text);
        box-shadow: inset 2px 0 0 0 var(--nav-active-border); /* Indicator line */
    }

    .main-content {
        flex: 1;
        display: flex;
        flex-direction: column;
        overflow-y: hidden;
    }

    .main-header {
        display: flex;
        justify-content: flex-end;
        align-items: center;
        padding: 1rem 2rem;
        background-color: var(--bg-header);
        border-bottom: 1px solid var(--border-color);
        height: 65px;
        flex-shrink: 0;
    }

    .logout-btn {
        background-color: var(--accent-primary);
        color: white;
        border: none;
        padding: 0.5rem 1rem;
        border-radius: 6px;
        cursor: pointer;
        font-weight: 500;
        transition: background-color 0.2s ease;
    }

    .logout-btn:hover {
        background-color: var(--accent-primary-hover);
    }

    .page-content {
        padding: 2rem;
        flex: 1;
        overflow-y: auto;
        scrollbar-color: var(--border-color) var(--bg-main);
    }
</style>