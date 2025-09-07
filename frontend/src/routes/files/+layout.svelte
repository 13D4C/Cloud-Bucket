<script lang="ts">
	import { goto } from '$app/navigation';
	import { Home, Trash2, Cloud, Folder } from 'lucide-svelte';
    import { page } from '$app/stores';

	function handleLogout() {
		localStorage.removeItem('jwt_token');
		goto('/');
	}

    $: isTrash = $page.url.searchParams.get('view') === 'trash';
    $: isShare = $page.url.searchParams.get('view') === 'share';
</script>

<div class="flex h-screen bg-primary-900">
	<aside class="w-60 bg-primary-800 border-r border-primary-700 flex flex-col p-6 flex-shrink-0">
		<div class="flex items-center gap-3 text-2xl font-bold mb-8 px-2 text-primary-50">
			<Cloud size={28} />
			<span>IT-Cloud</span>
		</div>
		<nav class="space-y-2">
			<a 
				href="/files" 
				class="flex items-center gap-3 px-4 py-3 rounded-lg text-primary-300 font-medium transition-all duration-200 hover:bg-primary-700 hover:text-primary-50 {!isTrash && !isShare ? 'bg-accent-500/15 text-accent-500 shadow-inner shadow-accent-500/20 border-l-2 border-accent-500' : ''}"
			>
				<Home size={20} />
				<span>My Files</span>
			</a>
			<a 
				href="/files?view=trash" 
				class="flex items-center gap-3 px-4 py-3 rounded-lg text-primary-300 font-medium transition-all duration-200 hover:bg-primary-700 hover:text-primary-50 {isTrash ? 'bg-accent-500/15 text-accent-500 shadow-inner shadow-accent-500/20 border-l-2 border-accent-500' : ''}"
			>
				<Trash2 size={20} />
				<span>Trash</span>
			</a>
            <a 
				href="/files?view=share" 
				class="flex items-center gap-3 px-4 py-3 rounded-lg text-primary-300 font-medium transition-all duration-200 hover:bg-primary-700 hover:text-primary-50 {isShare ? 'bg-accent-500/15 text-accent-500 shadow-inner shadow-accent-500/20 border-l-2 border-accent-500' : ''}"
			>
				<Folder size={20} />
				<span>share</span>
			</a>
		</nav>
	</aside>

	<main class="flex-1 flex flex-col overflow-y-hidden">
		<header class="flex justify-end items-center p-8 bg-primary-800 border-b border-primary-700 h-20 flex-shrink-0">
			<div class="flex items-center">
				<button 
					on:click={handleLogout} 
					class="bg-accent-500 text-white border-none px-4 py-2 rounded-md cursor-pointer font-medium transition-colors duration-200 hover:bg-accent-600"
				>
					Logout
				</button>
			</div>
		</header>
		<div class="p-8 flex-1 overflow-y-auto scrollbar-thin scrollbar-track-primary-900 scrollbar-thumb-primary-700">
			<slot />
		</div>
	</main>
</div>

