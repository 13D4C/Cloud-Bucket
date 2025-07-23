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

<div class="page-header">
    <h1>Deleted Files</h1>
</div>
<p class="subtitle">Items in the trash can be restored or deleted forever.</p>

{#if error_message}
    <div class="error-banner"><AlertCircle size=18/> {error_message}</div>
{/if}

<div class="trash-list-container">
    <div class="grid-header">
        <div class="header-name">Name</div>
        <div class="header-actions">Actions</div>
    </div>

    {#each trashItems as item (item.path)}
        <div class="list-row" transition:fade|local>
            <div class="item-name" title={item.originalName || item.name}>
                {#if item.isDir}
                    <Folder size=20 color="#5DADE2" />
                {:else}
                    <FileText size=20 color="#6C757D" />
                {/if}
                <span>{item.originalName || item.name}</span>
            </div>
            <div class="item-actions">
                <button class="action-btn restore" on:click={() => handleRestore(item)}>
                    <RotateCcw size=16 /> Restore
                </button>
                <button class="action-btn delete" on:click={() => handlePermanentDelete(item)}>
                    <Trash2 size=16 /> Delete Forever
                </button>
            </div>
        </div>
    {:else}
        <div class="empty-state" transition:fade>
            <Trash2 size=48 />
            <h3>Your trash is empty</h3>
            <p>Items you delete will appear here.</p>
        </div>
    {/each}
</div>

<style>
    .page-header h1 { font-size: 1.75rem; margin: 0; color: #1F2937; }
    .subtitle { color: #6C757D; margin-top: 0.25rem; margin-bottom: 2rem; }
    .trash-list-container { border: 1px solid #E5E7EB; border-radius: 12px; overflow: hidden; }
    .grid-header { display: grid; grid-template-columns: 1fr auto; padding: 0.75rem 1.5rem; background-color: #F9FAFB; font-weight: 500; color: #6C757D; text-transform: uppercase; font-size: 0.8rem; letter-spacing: 0.05em; border-bottom: 1px solid #E5E7EB; }
    .list-row { display: grid; grid-template-columns: 1fr auto; align-items: center; padding: 1rem 1.5rem; border-bottom: 1px solid #F3F4F6; transition: background-color 0.2s; }
    .list-row:last-child { border: none; }
    .list-row:hover { background-color: #F9FAFB; }
    .item-name { display: flex; align-items: center; gap: 1rem; font-weight: 500; color: #1F2937; overflow: hidden; }
    .item-name span { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    .item-actions { display: flex; gap: 1rem; }
    .action-btn { padding: 0.5rem 1rem; border: 1px solid #DEE2E6; border-radius: 8px; font-weight: 500; cursor: pointer; display: flex; align-items: center; gap: 0.5rem; transition: all 0.2s; }
    .action-btn.restore { background-color: white; color: #343A40; }
    .action-btn.restore:hover { border-color: #4ade80; background-color: #f0fdf4; color: #166534; }
    .action-btn.delete { background-color: white; color: #343A40; }
    .action-btn.delete:hover { border-color: #f87171; background-color: #fef2f2; color: #991b1b; }
    .empty-state { text-align: center; padding: 4rem; color: #6C757D; }
    .empty-state h3 { margin: 1rem 0 0.5rem 0; color: #343A40; }
    .error-banner { background-color: #fef2f2; color: #991b1b; border: 1px solid #fecaca; padding: 1rem; border-radius: 8px; margin-bottom: 1rem; display: flex; align-items: center; gap: 0.75rem; }
</style>