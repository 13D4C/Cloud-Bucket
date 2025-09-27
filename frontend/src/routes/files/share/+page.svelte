<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi } from '$lib/api';
	import { Folder, FileText, Users, UserX, AlertCircle, Download  } from 'lucide-svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { th } from 'date-fns/locale';
    import { fade } from 'svelte/transition';
	import { goto } from '$app/navigation';

	// --- TYPE DEFINITIONS ---
	interface ItemInfo {
		id: string;
		name: string;
		size: number;
		modified: string;
		isDir: boolean;
		path: string;
	}
	interface SharedWithMeItem extends ItemInfo {
		ownerName: string;
		permission: string;
	}
	interface SharedByMeItem {
		id: string;
		name: string;
		isDir: boolean;
		sharedWith: {
			userId: number;
			username: string;
			permission: string;
		}[];
	}

	// --- STATE ---
	let activeTab: 'withMe' | 'byMe' = 'withMe';
	let sharedWithMe: SharedWithMeItem[] = [];
	let sharedByMe: SharedByMeItem[] = [];
	let isLoading = true;
	let errorMessage = '';

	// --- DATA FETCHING ---
	async function fetchData() {
		isLoading = true;
		errorMessage = '';
		try {
			const res = await fetchApi('/api/share-info');
			if (!res.ok) {
				const errorData = await res.json();
				throw new Error(errorData.error || 'Failed to load sharing information.');
			}
			const data = await res.json();
			sharedWithMe = data.sharedWithMe || [];
			sharedByMe = data.sharedByMe || [];
		} catch (e: any) {
			errorMessage = e.message;
		} finally {
			isLoading = false;
		}
	}

	// --- ACTIONS ---
	async function handleUnshare(item: SharedByMeItem, userIdToUnshare: number) {
		if (!confirm(`Are you sure you want to stop sharing "${item.name}" with this user?`)) {
			return;
		}
		try {
			const res = await fetchApi('/api/unshare', {
				method: 'POST',
				body: JSON.stringify({
					itemId: item.id,
					itemType: item.isDir ? 'folder' : 'file',
					shareWithUserId: userIdToUnshare
				})
			});
			if (!res.ok) {
				const errorData = await res.json();
				throw new Error(errorData.error || 'Unshare operation failed.');
			}
			// Refresh data to show the change
			await fetchData();
		} catch (e: any) {
			alert(`Error: ${e.message}`);
		}
	}

	function formatBytes(bytes: number, decimals = 2) {
		if (!+bytes) return '0 Bytes';
		const k = 1024;
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${parseFloat((bytes / Math.pow(k, i)).toFixed(decimals))} ${['Bytes', 'KB', 'MB', 'GB', 'TB'][i]}`;
	}

	async function handleDownloadShared(item: SharedWithMeItem) {
		try {
			const endpoint = item.isDir ? `shared-folders/${item.id}/download` : `shared-files/${item.id}/download`;
			const res = await fetchApi(`/api/${endpoint}`, {});
			
			if (!res.ok) {
				let errorMessage = `Server responded with status ${res.status}`;
				try {
					const errorData = await res.json();
					errorMessage = errorData.error || errorMessage;
				} catch (e) {
					errorMessage = await res.text();
				}
				throw new Error(errorMessage);
			}
			
			const blob = await res.blob();
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.style.display = 'none';
			a.href = url;
			a.download = item.name + (item.isDir ? '.zip' : '');
			document.body.appendChild(a);
			a.click();
			window.URL.revokeObjectURL(url);
			a.remove();
		} catch (error: any) {
			console.error('Download failed:', error);
			alert(`Could not download: ${error.message}`);
		}
	}

	onMount(fetchData);
</script>

<div class="h-full text-primary-50 p-8" in:fade>
	<div class="flex justify-between items-center mb-6">
		<h1 class="text-3xl font-bold">Shared Items</h1>
	</div>

	<!-- TABS -->
	<div class="flex border-b border-primary-700 mb-6">
		<button
			class="px-6 py-3 text-lg font-medium transition-colors cursor-pointer"
			class:text-accent-400={activeTab === 'withMe'}
			class:border-b-2={activeTab === 'withMe'}
			class:border-accent-400={activeTab === 'withMe'}
			class:text-primary-400={activeTab !== 'withMe'}
			on:click={() => (activeTab = 'withMe')}>Shared with Me</button
		>
		<button
			class="px-6 py-3 text-lg font-medium transition-colors cursor-pointer"
			class:text-accent-400={activeTab === 'byMe'}
			class:border-b-2={activeTab === 'byMe'}
			class:border-accent-400={activeTab === 'byMe'}
			class:text-primary-400={activeTab !== 'byMe'}
			on:click={() => (activeTab = 'byMe')}>Shared by Me</button
		>
	</div>

	<!-- ERROR MESSAGE -->
	{#if errorMessage}
		<div class="bg-red-500 bg-opacity-10 text-red-300 border border-red-500 px-4 py-4 rounded-lg mb-6 flex items-center gap-3">
			<AlertCircle size={20} />
			{errorMessage}
		</div>
	{/if}

	<!-- LOADING INDICATOR -->
	{#if isLoading}
		<div class="text-center py-16 text-primary-400">Loading shared items...</div>
	{/if}

	<!-- CONTENT -->
{#if !isLoading && !errorMessage}
	{#if activeTab === 'withMe'}
		<div class="border border-primary-600 rounded-xl overflow-hidden bg-primary-800">
			<div class="grid grid-cols-[3fr_1fr_1fr_1.5fr_auto] px-6 py-3 bg-primary-900 text-xs uppercase text-primary-400">
				<div>Name</div>
				<div>Owner</div>
				<div>Size</div>
				<div>Date Shared</div>
				<div>Actions</div>
			</div>
			{#each sharedWithMe as item (item.id)}
				<div class="grid grid-cols-[3fr_1fr_1fr_1.5fr_auto] items-center px-6 py-4 border-b border-primary-700 last:border-b-0 hover:bg-primary-700 transition-colors group">
					<div class="flex items-center gap-4 font-medium text-primary-50">
						{#if item.isDir}
							<Folder size={20} class="text-blue-400" />
						{:else}
							<FileText size={20} class="text-primary-400" />
						{/if}
						<button 
							class="text-left hover:text-accent-400 transition-colors" 
							on:click={() => item.isDir ? goto(`/files/share/folder/${item.id}`) : null}
							class:cursor-pointer={item.isDir}
						>
							{item.name}
						</button>
					</div>
					<div class="text-primary-300">{item.ownerName}</div>
					<div class="text-primary-300">{item.isDir ? '--' : formatBytes(item.size ?? 0)}</div>
					<div class="text-primary-300">{formatDistanceToNow(new Date(item.modified), { locale: th, addSuffix: true })}</div>
					<div class="flex justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
						<button 
							class="p-1 text-primary-400 hover:text-accent-500 transition-colors" 
							on:click={() => handleDownloadShared(item)} 
							title={`Download ${item.isDir ? 'folder' : 'file'}`}
						>
							<Download size={18} />
						</button>
					</div>
				</div>
			{:else}
				<div class="text-center py-16 text-primary-400">
					<Users size={48} class="mx-auto mb-4" />
					<h3 class="text-xl font-semibold text-primary-50">Nothing is shared with you</h3>
				</div>
			{/each}
		</div>
		{:else if activeTab === 'byMe'}
			<div class="border border-primary-600 rounded-xl overflow-hidden bg-primary-800">
				<div class="grid grid-cols-[2fr_3fr] px-6 py-3 bg-primary-900 text-xs uppercase text-primary-400">
					<div>Item Name</div>
					<div>Shared With</div>
				</div>
				{#each sharedByMe as item (item.id)}
					<div class="grid grid-cols-[2fr_3fr] items-start px-6 py-4 border-b border-primary-700 last:border-b-0">
						<div class="flex items-center gap-4 font-medium text-primary-50 pt-2">
							{#if item.isDir}
								<Folder size={20} class="text-blue-400" />
							{:else}
								<FileText size={20} class="text-primary-400" />
							{/if}
							<span>{item.name}</span>
						</div>
						<div class="flex flex-col gap-2">
							{#each item.sharedWith as user}
								<div class="flex justify-between items-center bg-primary-700 p-2 rounded-lg">
									<div>
										<span class="font-medium">{user.username}</span>
										<span class="text-xs text-primary-400 ml-2 bg-primary-600 px-2 py-0.5 rounded-full">{user.permission}</span>
									</div>
									<button on:click={() => handleUnshare(item, user.userId)} title="Stop sharing" class="p-1 text-primary-400 hover:text-red-400 transition-colors">
										<UserX size={18} />
									</button>
								</div>
							{/each}
						</div>
					</div>
				{:else}
					<div class="text-center py-16 text-primary-400">
						<Users size={48} class="mx-auto mb-4" />
						<h3 class="text-xl font-semibold text-primary-50">You haven't shared any items</h3>
					</div>
				{/each}
			</div>
		{/if}
	{/if}
</div>