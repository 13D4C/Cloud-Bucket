<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { fetchApi } from '$lib/api';
	import { Folder, FileText, Home, ChevronRight, Download, Upload, Plus, ArrowLeft } from 'lucide-svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { th } from 'date-fns/locale';
	import { fade } from 'svelte/transition';
	import * as tus from 'tus-js-client';

	interface FileItem {
		id: string;
		path: string;
		isDir: boolean;
		name: string;
		size?: number;
		modified: string;
	}

	let items: FileItem[] = [];
	let permission = 'read';
	let folderName = '';
	let sharedFolderId = '';
	let isLoading = true;
	let errorMessage = '';
	let currentPath = '';
	let uploadQueue: { id: number; file: File; progress: number; status: 'uploading' | 'preparing' | 'finalizing' | 'done' | 'error'; error?: string; }[] = [];
	let isUploading = false;

	$: folderId = $page.params.folderId;
	$: queryPath = $page.url.searchParams.get('path') || '/';

	$: breadcrumbs = [
		{ name: 'Shared Items', path: '/files/share' },
		{ name: folderName, path: `/files/share/folder/${folderId}` }
	].concat(
		queryPath === '/' ? [] : queryPath.split('/').filter(p => p).map((part, i, arr) => ({
			name: part,
			path: `/files/share/folder/${folderId}?path=/${arr.slice(0, i + 1).join('/')}`
		}))
	);

	async function fetchData() {
		isLoading = true;
		errorMessage = '';
		try {
			const res = await fetchApi(`/api/shared-folders/${folderId}/contents?path=${encodeURIComponent(queryPath)}`);
			if (!res.ok) {
				const errorData = await res.json();
				throw new Error(errorData.error || 'Failed to fetch shared folder contents');
			}
			const data = await res.json();
			items = data.items || [];
			permission = data.permission || 'read';
			folderName = data.folderName || 'Shared Folder';
			sharedFolderId = data.sharedFolderId || folderId;
		} catch (e: any) {
			errorMessage = e.message;
		} finally {
			isLoading = false;
		}
	}

	function formatBytes(bytes: number, decimals = 2) {
		if (!+bytes) return '0 Bytes';
		const k = 1024;
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${parseFloat((bytes / Math.pow(k, i)).toFixed(decimals))} ${['Bytes', 'KB', 'MB', 'GB', 'TB'][i]}`;
	}

	async function handleDownload(item: FileItem) {
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
			alert(`Could not download: ${error.message}`);
		}
	}

	// Upload functions (only available with write permission)
	function handleFileSelect(event: Event) {
		if (permission !== 'write') return;
		const input = event.target as HTMLInputElement;
		if (input.files) {
			startUploads(input.files);
		}
		input.value = '';
	}

async function startUploads(fileList: FileList) {
	isUploading = true;
	const newUploads = Array.from(fileList).map(file => ({
		id: Date.now() + Math.random(),
		file,
		progress: 0,
		status: 'preparing' as const
	}));
	uploadQueue = [...uploadQueue, ...newUploads];

	// Mark uploads as ready to start
	uploadQueue.forEach(item => {
		if (item.status === 'preparing') item.status = 'uploading';
	});
	uploadQueue = [...uploadQueue];

	const uploadPromises = newUploads.map(uploadItem => startSingleUpload(uploadItem));
	await Promise.allSettled(uploadPromises);
	await fetchData(); // Refresh folder contents

	// Clean up completed uploads after delay
	setTimeout(() => {
		uploadQueue = uploadQueue.filter(item => item.status !== 'done' && item.status !== 'error');
		if (uploadQueue.length === 0) {
			isUploading = false;
		}
	}, 5000);
}

function startSingleUpload(uploadItem: typeof uploadQueue[0]) {
	return new Promise<void>((resolve, reject) => {
		const upload = new tus.Upload(uploadItem.file, {
			// endpoint: `http://localhost:8080/uploads/`,
			endpoint: `/uploads/`,
			retryDelays: [0, 3000, 5000],
			metadata: {
				filename: uploadItem.file.name,
				filetype: uploadItem.file.type
			},
			onProgress: (bytes, total) => {
				const index = uploadQueue.findIndex(item => item.id === uploadItem.id);
				if (index !== -1) {
					uploadQueue[index].progress = (bytes / total) * 100;
					uploadQueue = [...uploadQueue];
				}
			},
			onError: (error) => {
				const index = uploadQueue.findIndex(item => item.id === uploadItem.id);
				if (index !== -1) {
					uploadQueue[index].status = 'error';
					uploadQueue[index].error = error.message;
					uploadQueue = [...uploadQueue];
				}
				reject(error);
			},
			onSuccess: async () => {
				let index = uploadQueue.findIndex(item => item.id === uploadItem.id);
				if (index !== -1) {
					uploadQueue[index].status = 'finalizing';
					uploadQueue = [...uploadQueue];
				}

				const uploadId = upload.url?.split('/').pop();
				if (!uploadId) {
					if (index !== -1) {
						uploadQueue[index].status = 'error';
						uploadQueue[index].error = 'Could not get upload ID';
						uploadQueue = [...uploadQueue];
					}
					reject(new Error('Upload ID missing'));
					return;
				}

				try {
					await fetchApi('/api/shared-folders/finalize-upload', {
						method: 'POST',
						body: JSON.stringify({
							uploadId,
							sharedFolderId: folderId,
							relativePath: queryPath === '/' ? '' : queryPath
						})
					});

					index = uploadQueue.findIndex(item => item.id === uploadItem.id);
					if (index !== -1) {
						uploadQueue[index].status = 'done';
						uploadQueue = [...uploadQueue];
					}
					resolve();
				} catch (finalizeError: any) {
					index = uploadQueue.findIndex(item => item.id === uploadItem.id);
					if (index !== -1) {
						uploadQueue[index].status = 'error';
						uploadQueue[index].error = finalizeError.message;
						uploadQueue = [...uploadQueue];
					}
					reject(finalizeError);
				}
			}
		});
		upload.start();
	});
}

	onMount(fetchData);
	$: if ($page.url) fetchData();
</script>

<div class="h-full text-primary-50 p-8" in:fade>
	<div class="flex justify-between items-center mb-6">
		<div class="flex items-center gap-1 text-sm">
			{#each breadcrumbs as crumb, i}
				<a href={crumb.path} class="flex items-center gap-2 text-primary-400 px-2 py-2 rounded-md hover:bg-primary-800 transition-colors">
					{#if i === 0}
						<ArrowLeft size=16/>
					{:else}
						<span>{crumb.name}</span>
					{/if}
				</a>
				{#if i < breadcrumbs.length - 1}
					<ChevronRight size=16 class="text-primary-600" />
				{/if}
			{/each}
		</div>
		
		{#if permission === 'write'}
			<div class="flex gap-3">
				<label class="flex items-center gap-2 px-5 py-3 rounded-lg font-medium cursor-pointer bg-accent-500 text-white hover:bg-accent-600 transition-all">
					<Upload size=16/> Upload Files
					<input type="file" class="hidden" on:change={handleFileSelect} multiple />
				</label>
			</div>
		{/if}
	</div>

	{#if errorMessage}
		<div class="bg-red-500 bg-opacity-10 text-red-300 border border-red-500 px-4 py-4 rounded-lg mb-6">
			{errorMessage}
		</div>
	{/if}

	{#if isLoading}
		<div class="text-center py-16 text-primary-400">Loading...</div>
	{:else}
		<div class="border border-primary-600 rounded-xl overflow-hidden bg-primary-800">
			<div class="grid grid-cols-[3fr_1fr_1.5fr_auto] px-6 py-3 bg-primary-900 text-xs uppercase text-primary-400">
				<div>Name</div>
				<div>Size</div>
				<div>Modified</div>
				<div>Actions</div>
			</div>
			
			{#each items as item (item.id)}
				<div class="grid grid-cols-[3fr_1fr_1.5fr_auto] items-center px-6 py-4 border-b border-primary-700 last:border-b-0 hover:bg-primary-700 transition-colors group">
					<div class="flex items-center gap-4 font-medium text-primary-50">
						{#if item.isDir}
							<Folder size=20 class="text-blue-400"/>
							<button 
								class="hover:text-accent-400 transition-colors" 
								on:click={() => goto(`/files/share/folder/${folderId}?path=${item.name}`)}
							>
								{item.name}
							</button>
						{:else}
							<FileText size=20 class="text-primary-400"/>
							<span>{item.name}</span>
						{/if}
					</div>
					
					<div class="text-primary-300">{item.isDir ? '--' : formatBytes(item.size ?? 0)}</div>
					
					<div class="text-primary-300">{formatDistanceToNow(new Date(item.modified), {locale: th, addSuffix: true})}</div>
					
					<div class="flex justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
						<button class="p-1 text-primary-400 hover:text-accent-500 transition-colors" on:click={() => handleDownload(item)} title="Download">
							<Download size=18/>
						</button>
					</div>
				</div>
			{:else}
				<div class="text-center py-16 text-primary-400">
					<Folder size=48 class="mx-auto mb-4" />
					<h3 class="text-xl font-semibold text-primary-50">Folder is empty</h3>
				</div>
			{/each}
		</div>
	{/if}
</div>