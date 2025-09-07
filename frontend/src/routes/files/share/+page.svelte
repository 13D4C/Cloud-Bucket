<script lang="ts">
    import { onMount } from 'svelte';
    import { fetchApi } from '$lib/api';
    import { Folder, FileText, RotateCcw, Trash2, AlertCircle } from 'lucide-svelte';
    import { formatDistanceToNow } from 'date-fns';
    import { th } from 'date-fns/locale';
    import { fade } from 'svelte/transition';

    let trashItems: any[] = [];
    let error_message = '';

    async function fetchTrashItems() {
        error_message = '';
        try {
            const res = await fetchApi('/api/trash');
            if (!res.ok) {
                const errData = await res.json();
                throw new Error(errData.error || 'Could not fetch trash items');
            }
            trashItems = await res.json() || [];
        } catch (e: any) {
            error_message = e.message;
        }
    }
    onMount(fetchTrashItems);

    async function handleRestore(item: any) {
        const itemName = item.originalName || item.name;
        if (!confirm(`Are you sure you want to restore "${itemName}"?`)) return;
        try {
            await fetchApi('/api/trash/restore', {
                method: 'POST',
                body: JSON.stringify({ path: item.path })
            });
            await fetchTrashItems(); // Refresh list after restoring
        } catch (e: any) { 
            alert(`Restore failed: ${e.message}`); 
        }
    }

    async function handlePermanentDelete(item: any) {
        const itemName = item.originalName || item.name;
        if (!confirm(`This will permanently delete "${itemName}". This action cannot be undone. Are you sure?`)) return;
        try {
            await fetchApi(`/api/trash/${item.path}`, { method: 'DELETE' });
            await fetchTrashItems(); // Refresh list after deleting
        } catch (e: any) { 
            alert(`Permanent delete failed: ${e.message}`); 
        }
    }
    
    function formatBytes(bytes: number, decimals = 2) {
		if (!+bytes) return '0 Bytes';
		const k = 1024;
		const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${parseFloat((bytes / Math.pow(k, i)).toFixed(decimals < 0 ? 0 : decimals))} ${sizes[i]}`;
	}
</script>

<div class="mb-6">
    <h1 class="text-3xl font-bold text-primary-50 m-0">Deleted Files</h1>
</div>
<p class="text-primary-300 mt-1 mb-8">Items in the trash can be restored or deleted forever.</p>

{#if error_message}
    <div class="bg-red-900/20 text-red-400 border border-red-800 p-4 rounded-lg mb-4 flex items-center gap-3">
        <AlertCircle size=18/> {error_message}
    </div>
{/if}

<div class="border border-primary-700 rounded-xl overflow-hidden bg-primary-800">
    <div class="grid grid-cols-2 px-6 py-3 bg-primary-900 font-medium text-primary-400 uppercase text-xs tracking-wider border-b border-primary-700">
        <div>Name</div>
        <div>Actions</div>
    </div>

    {#each trashItems as item (item.path)}
        <div class="grid grid-cols-2 items-center px-6 py-4 border-b border-primary-700 transition-colors duration-200 hover:bg-primary-700 last:border-0" transition:fade|local>
            <div class="flex items-center gap-4 font-medium text-primary-50 overflow-hidden" title={item.originalName || item.name}>
                {#if item.isDir}
                    <Folder size=20 color="#5DADE2" />
                {:else}
                    <FileText size=20 color="#6C757D" />
                {/if}
                <span class="whitespace-nowrap overflow-hidden text-ellipsis">{item.originalName || item.name}</span>
            </div>
            <div class="flex gap-4">
                <button 
                    class="px-4 py-2 border border-primary-600 rounded-lg font-medium cursor-pointer flex items-center gap-2 transition-all duration-200 bg-primary-800 text-primary-300 hover:border-green-400 hover:bg-green-900/20 hover:text-green-400" 
                    on:click={() => handleRestore(item)}
                >
                    <RotateCcw size=16 /> Restore
                </button>
                <button 
                    class="px-4 py-2 border border-primary-600 rounded-lg font-medium cursor-pointer flex items-center gap-2 transition-all duration-200 bg-primary-800 text-primary-300 hover:border-red-400 hover:bg-red-900/20 hover:text-red-400" 
                    on:click={() => handlePermanentDelete(item)}
                >
                    <Trash2 size=16 /> Delete Forever
                </button>
            </div>
        </div>
    {:else}
        <div class="text-center py-16 text-primary-400" transition:fade>
            <Trash2 size=48 />
            <h3 class="my-4 mb-2 text-primary-300">Your trash is empty</h3>
            <p>Items you delete will appear here.</p>
        </div>
    {/each}
</div>

