<script lang="ts">
    import { fetchApi } from '$lib/api';
    import { onMount } from 'svelte';
    import { fly } from 'svelte/transition';
    import { FileText, Folder, RotateCcw, Trash2, AlertCircle } from 'lucide-svelte';
    import { formatDistanceToNow } from 'date-fns';
	import { th } from 'date-fns/locale';

    let trashedItems: any[] = [];
    let error_message = '';

    async function fetchTrashedItems() {
        try {
            const res = await fetchApi('/api/trash');
            if (!res.ok) throw new Error('Failed to load trash items');
            trashedItems = await res.json() || [];
        } catch (e: any) {
            error_message = e.message;
        }
    }
    onMount(fetchTrashedItems);

    async function handleRestore(item: any) {
        try {
            await fetchApi('/api/trash/restore', {
                method: 'POST',
                body: JSON.stringify({ path: item.path }) // ✅ Send path
            });
            trashedItems = trashedItems.filter(i => i.path !== item.path);
        } catch (e) {
            alert('Failed to restore item.');
        }
    }

    async function handlePermanentDelete(item: any) {
        if (!confirm('This action is permanent and cannot be undone. Are you sure?')) return;
        try {
            await fetchApi(`/api/trash/${item.path}`, { method: 'DELETE' }); // ✅ Send path
            trashedItems = trashedItems.filter(i => i.path !== item.path);
        } catch (e) {
            alert('Failed to permanently delete item.');
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

<h1>Trash</h1>
<p class="subtitle">Items in the trash will be permanently deleted after 7 days.</p>

{#if error_message}
    <div class="error-banner"><AlertCircle size=18/> {error_message}</div>
{/if}

<div class="trash-list-container">
    <div class="list-header">
        <div class="header-name">Name</div>
		<div class="header-size">Size</div>
        <div class="header-date">Date Deleted</div>
    </div>
    {#each trashedItems as item (item.path)}
        <div class="list-row" transition:fly={{ y: 20, duration: 300 }}>
            <div class="item-name">
                {#if item.isDir} <Folder size=20 color="#5DADE2" /> {:else} <FileText size=20 color="#6C757D" /> {/if}
                <span>{item.name}</span>
            </div>
			<div class="item-size">{item.isDir ? '--' : formatBytes(item.size)}</div>
            <div class="item-date">{formatDistanceToNow(new Date(item.modified),{addSuffix:true,locale:th})}</div>
            <div class="row-actions">
                <button class="action-icon" title="Restore" on:click={() => handleRestore(item)}><RotateCcw size=18 /></button>
                <button class="action-icon" title="Delete Permanently" on:click={() => handlePermanentDelete(item)}><Trash2 size=18 /></button>
            </div>
        </div>
    {:else}
        <div class="empty-state">Your trash is empty.</div>
    {/each}
</div>

<style>
    h1 { margin: 0; font-size: 1.75rem; }
    .subtitle { margin: 0.25rem 0 1.5rem 0; color: #6C757D; }
    .trash-list-container { border: 1px solid #E5E7EB; border-radius: 12px; overflow: hidden; }
    .list-header { display: grid; grid-template-columns: 2fr 1fr 1fr 150px; align-items: center; padding: 0.75rem 1.5rem; background-color: #F9FAFB; font-weight: 500; color: #6C757D; text-transform: uppercase; font-size: 0.8rem; }
    .list-header .header-name { grid-column: 1; }
    .list-header .header-size { grid-column: 2; text-align: right; }
    .list-header .header-date { grid-column: 3; }
    .list-row { display: grid; grid-template-columns: 2fr 1fr 1fr 150px; align-items: center; padding: 1rem 1.5rem; border-bottom: 1px solid #F3F4F6; }
    .list-row:last-child { border-bottom: none; }
    .item-name { grid-column: 1; display: flex; align-items: center; gap: 1rem; font-weight: 500; }
    .item-size { grid-column: 2; text-align: right; }
    .item-date { grid-column: 3; }
    .row-actions { grid-column: 4; display: flex; justify-content: flex-end; gap: 0.5rem; }
    .action-icon { background: none; border: none; padding: 0.25rem; color: #6C757D; cursor: pointer; }
    .action-icon:hover { color: #0d6efd; }
    .empty-state { text-align: center; padding: 4rem; color: #6C757D; }
    .error-banner { background-color: #fef2f2; color: #991b1b; border: 1px solid #fecaca; padding: 1rem; border-radius: 8px; margin-bottom: 1rem; }
</style>