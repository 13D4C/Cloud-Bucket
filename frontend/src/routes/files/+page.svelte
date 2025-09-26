<script lang="ts">
    import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import * as tus from 'tus-js-client';
	import { Folder, FileText, UploadCloud, Home, ChevronRight, Download, X, AlertCircle, Plus, Clock, CheckCircle, XCircle, Upload, CornerLeftUp, Share } from 'lucide-svelte';
    import { Trash2 } from 'lucide-svelte'; // Correctly placed Trash2 import
	import { formatDistanceToNow } from 'date-fns';
	import { th } from 'date-fns/locale';
    import { fetchApi } from '$lib/api';
	import { afterUpdate } from 'svelte';
    import { fly, fade } from 'svelte/transition';

	interface FileItem {
        id: string;
		path: string;
		isDir: boolean;
		name: string; // Standardized to use 'name' for display
		size?: number;
		modified: string;
	}

	let files: FileItem[] = [];
	let folders: FileItem[] = [];
	let items: FileItem[] = [];
	let recentFiles: FileItem[] = [];
	let error_message = '';
	let showCreateFolderModal = false;
	let newFolderName = '';
    let draggedItem: FileItem | null = null;
    let isDraggingOver = false;
    let uploadQueue: { id: number; file: File; progress: number; status: 'uploading' | 'preparing' | 'finalizing' | 'done' | 'error'; error?: string; path?: string; }[] = [];
    let totalUploadProgress = 0;
    let isUploading = false;
    let selectedItems = new Set<string>();
    let selectAllCheckbox: HTMLInputElement;

    // --- Sharing State ---
    let showShareModal = false;
    let shareModalItem: FileItem | null = null;
    let shareUsername = '';
    let sharePermission = 'read';
    let isSharing = false;

    // --- Quota State ---
    let quotaInfo = { quotaLimit: 0, quotaUsed: 0, quotaAvailable: 0 };

	// --- Computed State ---
	$: currentPath = $page.url.searchParams.get('path') || '';
	$: inSelectionMode = selectedItems.size > 0;
    $: allSelected = items.length > 0 && selectedItems.size === items.length;
	$: breadcrumbs = [{ name: 'My Files', path: '/files' }].concat(
		currentPath.split('/').filter(p => p).map((part, i, arr) => ({ name: part, path: `/files?path=/${arr.slice(0, i + 1).join('/')}` }))
	);
    $: {
        if (files) {
            recentFiles = [...files]
                .sort((a: FileItem, b: FileItem) => new Date(b.modified).getTime() - new Date(a.modified).getTime())
                .slice(0, 4);
        }
    }
    $: {
        if (uploadQueue.length > 0) {
            const total = uploadQueue.reduce((acc, curr) => acc + curr.progress, 0);
            totalUploadProgress = total / uploadQueue.length;
        } else {
            totalUploadProgress = 0;
        }
    }

    afterUpdate(() => {
        if (selectAllCheckbox) {
            selectAllCheckbox.indeterminate = selectedItems.size > 0 && !allSelected;
        }
    });

	// --- Data Fetching ---
	async function fetchData() {
        error_message = '';
		try {
			const endpoint = `/api/files?path=${encodeURIComponent(currentPath)}`;
			const res = await fetchApi(endpoint);
			if (!res.ok) {
                const errData = await res.json();
                throw new Error(errData.error || 'Failed to fetch items');
            }
			const allItems: FileItem[] = (await res.json()) || [];
            // --- MODIFIED: Standardized sorting to use `name` property ---
			folders = allItems.filter((item: FileItem) => item.isDir).sort((a: FileItem, b: FileItem) => a.name.localeCompare(b.name));
			files = allItems.filter((item: FileItem) => !item.isDir).sort((a: FileItem, b: FileItem) => a.name.localeCompare(b.name));
            items = [...folders, ...files];
		} catch (error: any) {
			console.error("Fetch Error:", error);
			error_message = `Could not load items: ${error.message}`;
		}
	}

    async function fetchQuotaInfo() {
        try {
            const res = await fetchApi('/api/quota');
            if (res.ok) {
                quotaInfo = await res.json();
            }
        } catch (error: any) {
            console.error("Quota fetch error:", error);
        }
    }

	$: if ($page.url) { fetchData(); fetchQuotaInfo(); selectedItems = new Set(); error_message = ''; }

	// --- Handlers for Selection ---
    function toggleSelect(id: string) {
        const newSelectedItems = new Set(selectedItems);
        if (newSelectedItems.has(id)) {
            newSelectedItems.delete(id);
        } else {
            newSelectedItems.add(id);
        }
        selectedItems = newSelectedItems;
    }
    async function handleBulkDelete() {
        if (!confirm(`Are you sure you want to move ${selectedItems.size} items to the trash?`)) return;
        const itemsById = new Map(items.map(item => [item.id, item]));
        const file_ids: number[] = [];
        const folder_ids: string[] = [];

        for (const id of selectedItems) {
            const item = itemsById.get(id);
            if (item) {
                if (item.isDir) {
                    folder_ids.push(id);
                } else {
                    // Backend expects file IDs as integers
                    file_ids.push(parseInt(id, 10));
                }
            }
        }
        try {
            await fetchApi('/api/items/bulk-delete', {
                method: 'POST',
                body: JSON.stringify({ file_ids, folder_ids })
            });
            selectedItems = new Set();
            await fetchData();
            await fetchQuotaInfo(); // Refresh quota after deletion
        } catch (e: any) {
            alert(`Error moving items to trash: ${e.message}`);
        }
    }
    function toggleSelectAll() {
        if (allSelected) { selectedItems = new Set(); } else { selectedItems = new Set(items.map(item => item.id)); }
    }
    async function handleBulkDownload() {
        if (selectedItems.size === 0) return alert("Please select items to download.");
        try {
            const res = await fetchApi('/api/items/bulk-download', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ paths: Array.from(selectedItems) }) });
            if (!res.ok) {
                const errorData = await res.json();
                throw new Error(errorData.error || `Server responded with status ${res.status}`);
            }
            const blob = await res.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.style.display = 'none';
            a.href = url;
            // The backend now sets the filename via Content-Disposition, but this is a good fallback.
            a.download = `IT-Cloud-Download.zip`; 
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            a.remove();
            selectedItems = new Set();
        } catch (error: any) {
            console.error('Bulk download failed:', error);
            alert(`Could not download items: ${error.message}`);
        }
    }

    // --- Sharing Functions ---
    function openShareModal(item: FileItem) {
        shareModalItem = item;
        shareUsername = '';
        sharePermission = 'read';
        showShareModal = true;
    }

    function closeShareModal() {
        showShareModal = false;
        shareModalItem = null;
        shareUsername = '';
        sharePermission = 'read';
        isSharing = false;
    }

    async function handleShare() {
        if (!shareModalItem || !shareUsername.trim()) {
            alert('Please enter a username to share with.');
            return;
        }

        isSharing = true;
        try {
            const res = await fetchApi('/api/share', {
                method: 'POST',
                body: JSON.stringify({
                    itemId: shareModalItem.id,
                    itemType: shareModalItem.isDir ? 'folder' : 'file',
                    shareWithUsername: shareUsername.trim(),
                    permission: sharePermission
                })
            });

            if (!res.ok) {
                const errorData = await res.json();
                throw new Error(errorData.error || 'Failed to share item');
            }

            alert(`Successfully shared "${shareModalItem.name}" with ${shareUsername}`);
            closeShareModal();
        } catch (error: any) {
            alert(`Error sharing item: ${error.message}`);
        } finally {
            isSharing = false;
        }
    }

    async function handleBulkShare() {
        if (selectedItems.size === 0) return alert("Please select items to share.");
        if (!shareUsername.trim()) {
            alert('Please enter a username to share with.');
            return;
        }

        isSharing = true;
        const itemsById = new Map(items.map(item => [item.id, item]));
        const promises: Promise<any>[] = [];

        for (const id of selectedItems) {
            const item = itemsById.get(id);
            if (item) {
                promises.push(
                    fetchApi('/api/share', {
                        method: 'POST',
                        body: JSON.stringify({
                            itemId: item.id,
                            itemType: item.isDir ? 'folder' : 'file',
                            shareWithUsername: shareUsername.trim(),
                            permission: sharePermission
                        })
                    })
                );
            }
        }

        try {
            const results = await Promise.allSettled(promises);
            let successCount = 0;
            let failCount = 0;

            for (const result of results) {
                if (result.status === 'fulfilled' && result.value.ok) {
                    successCount++;
                } else {
                    failCount++;
                }
            }

            if (successCount > 0) {
                alert(`Successfully shared ${successCount} items with ${shareUsername}${failCount > 0 ? `. ${failCount} items failed to share.` : ''}`);
            } else {
                alert(`Failed to share items with ${shareUsername}`);
            }

            selectedItems = new Set();
            closeShareModal();
        } catch (error: any) {
            alert(`Error sharing items: ${error.message}`);
        } finally {
            isSharing = false;
        }
    }

	// --- Drag & Drop Upload Handlers ---
    function handleUploadDragOver(event: DragEvent) { event.preventDefault(); if (!draggedItem) { isDraggingOver = true; } }
    function handleUploadDragLeave() { isDraggingOver = false; }
    async function handleUploadDrop(event: DragEvent) { 
        event.preventDefault(); 
        isDraggingOver = false; 
        if (!draggedItem && event.dataTransfer?.files) { 
            const fileArray = Array.from(event.dataTransfer.files);
            await startMultipleUploadsFromArray(fileArray); 
        } 
    }

	// --- Upload Logic ---
    
    // Check user quota and validate files before upload
    async function checkQuotaAndValidateFiles(fileList: FileList): Promise<{ valid: boolean; error?: string }> {
        try {
            const res = await fetchApi('/api/quota');
            if (!res.ok) {
                return { valid: false, error: 'Could not check quota limits' };
            }
            const quota = await res.json();
            
            const totalFileSize = Array.from(fileList).reduce((sum, file) => sum + file.size, 0);
            const availableSpace = quota.quotaLimit - quota.quotaUsed;
            
            if (totalFileSize > availableSpace) {
                const formatBytes = (bytes: number) => {
                    if (bytes === 0) return '0 B';
                    const k = 1024;
                    const sizes = ['B', 'KB', 'MB', 'GB'];
                    const i = Math.floor(Math.log(bytes) / Math.log(k));
                    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
                };
                
                return { 
                    valid: false, 
                    error: `Not enough storage space. Files need ${formatBytes(totalFileSize)} but only ${formatBytes(availableSpace)} available.` 
                };
            }
            
            return { valid: true };
        } catch (error: any) {
            return { valid: false, error: `Error checking quota: ${error.message}` };
        }
    }

    function handleFileSelect(event: Event) { 
        const input = event.target as HTMLInputElement; 
        if (input.files && input.files.length > 0) { 
            // Create a copy of the files array before clearing the input
            const fileArray = Array.from(input.files);
            input.value = ''; // Clear input immediately after copying files
            startMultipleUploadsFromArray(fileArray); 
        } else {
            input.value = '';
        }
    }
    function handleFolderSelect(event: Event) { 
        const input = event.target as HTMLInputElement; 
        if (input.files && input.files.length > 0) { 
            const fileArray = Array.from(input.files);
            input.value = '';
            startMultipleUploadsFromArray(fileArray); 
        } else {
            input.value = '';
        }
    }
    async function startMultipleUploadsFromArray(fileArray: File[]) {
        // Convert array to FileList-like object for quota checking
        const fileListLike = {
            length: fileArray.length,
            item: (index: number) => fileArray[index] || null,
            [Symbol.iterator]: () => fileArray[Symbol.iterator](),
            ...fileArray.reduce((obj, file, index) => ({ ...obj, [index]: file }), {})
        } as FileList;
        
        // Check quota limits before starting upload
        const validation = await checkQuotaAndValidateFiles(fileListLike);
        if (!validation.valid) {
            error_message = validation.error || 'Upload validation failed';
            return;
        }

        isUploading = true; 
        const newUploads = fileArray.map(file => {
            return { id: Date.now() + Math.random(), file, progress: 0, status: 'preparing' as const, path: (file as any).webkitRelativePath || file.name };
        });
        uploadQueue = [...uploadQueue, ...newUploads];
        const dirPaths = new Set<string>();
        newUploads.forEach(upload => {
            const pathParts = upload.path.split('/');
            if (pathParts.length > 1) {
                for (let i = 1; i < pathParts.length; i++) {
                    dirPaths.add(pathParts.slice(0, i).join('/'));
                }
            }
        });
        if (dirPaths.size > 0) {
            const createFolderPromises = Array.from(dirPaths).map(path => fetchApi(`/api/folders/structure`, { method: 'POST', body: JSON.stringify({ path: `${currentPath}/${path}`.replace(/^\//, '') }) }));
            await Promise.allSettled(createFolderPromises);
        }
        uploadQueue.forEach(item => { if (item.status === 'preparing') item.status = 'uploading'; });
        uploadQueue = [...uploadQueue];
        const uploadPromises = newUploads.map(uploadItem => startSingleUpload(uploadItem));
        await Promise.allSettled(uploadPromises);
        await fetchData();
        await fetchQuotaInfo(); // Refresh quota after uploads
        setTimeout(() => {
            uploadQueue = uploadQueue.filter(item => item.status !== 'done' && item.status !== 'error');
            if (uploadQueue.length === 0) { isUploading = false; }
        }, 5000);
    }

	async function startMultipleUploads(fileList: FileList) {
        // Convert FileList to array and delegate to the array version
        const fileArray = Array.from(fileList);
        return startMultipleUploadsFromArray(fileArray);
    }
    function startSingleUpload(uploadItem: typeof uploadQueue[0]) {
        return new Promise<void>((resolve, reject) => {
            const pathParts = (uploadItem.path || '').split('/');
            pathParts.pop();
            const folderPath = pathParts.join('/');
            // const destinationPath = currentPath ? `/${currentPath}/${folderPath}`.replace(/^\//, '') : `/${folderPath}`;
            let destinationPath = '';
            if (currentPath && folderPath) {
                destinationPath = `${currentPath}/${folderPath}`;
            } else if (currentPath) {
                destinationPath = `${currentPath}`;
            } else if (folderPath) {
                destinationPath = `/${folderPath}`;
            }
            const upload = new tus.Upload(uploadItem.file, {
                endpoint: `http://localhost:8080/uploads/`,
                retryDelays: [0, 3000, 5000],
                metadata: { filename: uploadItem.file.name, filetype: uploadItem.file.type },
                onProgress: (bytes, total) => {
                    const index = uploadQueue.findIndex(item => item.id === uploadItem.id);
                    if (index !== -1) { uploadQueue[index].progress = (bytes / total) * 100; uploadQueue = [...uploadQueue]; }
                },
                onError: (error) => {
                    const index = uploadQueue.findIndex(item => item.id === uploadItem.id);
                    if (index !== -1) { uploadQueue[index].status = 'error'; uploadQueue[index].error = error.message; uploadQueue = [...uploadQueue]; }
                    reject(error);
                },
                onSuccess: async () => {
                    let index = uploadQueue.findIndex(item => item.id === uploadItem.id);
                    if (index !== -1) { uploadQueue[index].status = 'finalizing'; uploadQueue = [...uploadQueue]; }
                    const uploadId = upload.url?.split('/').pop();
                    if (!uploadId) {
                        if (index !== -1) { uploadQueue[index].status = 'error'; uploadQueue[index].error = 'Could not get finalize ID.'; uploadQueue = [...uploadQueue]; }
                        reject(new Error('Finalize ID missing')); return;
                    }
                    try {
                        await fetchApi(`/api/finalize-upload`, { method: 'POST', body: JSON.stringify({ uploadId, destinationPath: destinationPath || '/' }) });
                        index = uploadQueue.findIndex(item => item.id === uploadItem.id);
                        if (index !== -1) { uploadQueue[index].status = 'done'; uploadQueue = [...uploadQueue]; }
                        resolve();
                    } catch (finalizeError: any) {
                        index = uploadQueue.findIndex(item => item.id === uploadItem.id);
                        if (index !== -1) { uploadQueue[index].status = 'error'; uploadQueue[index].error = finalizeError.message; uploadQueue = [...uploadQueue]; }
                        reject(finalizeError);
                    }
                }
            });
            upload.start();
        });
    }

	// --- Other Handlers ---
    function handleDragStart(item: FileItem) { draggedItem = item; }
    function handleDragEnd() { draggedItem = null; }
    async function handleDrop(destinationFolder: FileItem) { if (draggedItem && draggedItem.path !== destinationFolder.path) { await handleMove(draggedItem, destinationFolder); } handleDragEnd(); }
    async function handleMove(sourceItem: FileItem, destinationFolder: FileItem) { await fetchApi(`/api/move`,{method:'POST',body:JSON.stringify({sourcePath:sourceItem.path,destinationFolder:destinationFolder.path})}); await fetchData(); }
	async function handleCreateFolder() {
        if (!newFolderName.trim()) return;
        // --- CRITICAL CHANGE: Switched from query param to body for 'path' ---
        await fetchApi(`/api/folders`, {
            method: 'POST',
            body: JSON.stringify({
                folderName: newFolderName.trim(),
                path: currentPath
            })
        });
        showCreateFolderModal=false;
        newFolderName='';
        await fetchData();
    }
    async function handleMoveUp(item: FileItem) {
        if (!currentPath) return;
        const pathParts = currentPath.split('/');
        pathParts.pop();
        const destinationFolder = pathParts.join('/');
        if (!confirm(`Move "${item.name}" up one level?`)) return;
        try {
            await fetchApi('/api/move', { method: 'POST', body: JSON.stringify({ sourcePath: item.path, destinationFolder: destinationFolder }) });
            await fetchData();
        } catch (e: any) {
            alert(`Error moving item: ${e.message}`);
        }
    }
    async function handleDownload(item: FileItem) {
        try {
            const endpoint = item.isDir ? 'download-folder' : 'download';
            // Remove the full URL construction - let fetchApi handle the base URL
            const downloadUrl = `/api/${endpoint}/${item.path}`;
            const res = await fetchApi(downloadUrl, {});
            
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

    function formatBytes(bytes: number, decimals=2) { if(!+bytes)return"0 Bytes";const k=1024,i=Math.floor(Math.log(bytes)/Math.log(k));return`${parseFloat((bytes/Math.pow(k,i)).toFixed(decimals))} ${["Bytes","KB","MB","GB","TB"][i]}` }
</script>

<!-- New Folder Modal -->
{#if showCreateFolderModal}
	<div class="fixed inset-0 bg-black bg-opacity-80 flex items-center justify-center z-50" role="dialog" aria-modal="true" tabindex="0" on:keydown={(e) => { if (e.key === 'Escape') showCreateFolderModal = false; }}>
        <div class="relative bg-primary-800 p-8 rounded-xl w-11/12 max-w-md border border-primary-700 text-primary-50" transition:fly={{ y: -20, duration: 300 }}>
            <h3 class="text-xl font-semibold mb-6">New Folder</h3>
            <form on:submit|preventDefault={handleCreateFolder} class="flex flex-col gap-4">
                <input type="text" bind:value={newFolderName} placeholder="Enter folder name..." required
                    class="px-3 py-3 rounded-lg border border-primary-600 bg-primary-900 text-primary-50 text-base focus:border-accent-500 focus:outline-none" />
                <button type="submit" class="px-3 py-3 rounded-lg bg-accent-500 text-white font-medium hover:bg-accent-600 transition-colors">
                    Create Folder
                </button>
            </form>
            <button type="button" class="absolute top-3 right-3 p-1 text-primary-400 hover:text-primary-200" on:click={() => showCreateFolderModal = false}>
                <X size=20 />
            </button>
        </div>
	</div>
{/if}

<!-- Share Modal -->
{#if showShareModal}
	<div class="fixed inset-0 bg-black bg-opacity-80 flex items-center justify-center z-50" role="dialog" aria-modal="true" tabindex="0" on:keydown={(e) => { if (e.key === 'Escape') closeShareModal(); }}>
        <div class="relative bg-primary-800 p-8 rounded-xl w-11/12 max-w-md border border-primary-700 text-primary-50" transition:fly={{ y: -20, duration: 300 }}>
            <h3 class="text-xl font-semibold mb-6">
                {#if shareModalItem}
                    Share "{shareModalItem.name}"
                {:else}
                    Share Selected Items ({selectedItems.size})
                {/if}
            </h3>
            <form on:submit|preventDefault={shareModalItem ? handleShare : handleBulkShare} class="flex flex-col gap-4">
                <input type="text" bind:value={shareUsername} placeholder="Enter username..." required
                    class="px-3 py-3 rounded-lg border border-primary-600 bg-primary-900 text-primary-50 text-base focus:border-accent-500 focus:outline-none" />
                
                <div>
                    <label class="text-sm text-primary-300 gap-2 flex flex-col">
                        <div>Permission:</div>
                        <select bind:value={sharePermission} class="px-3 py-3 rounded-lg border border-primary-600 bg-primary-900 text-primary-50 text-base focus:border-accent-500 focus:outline-none">
                            <option value="read">Read Only</option>
                            {#if shareModalItem?.isDir || selectedItems.size > 0}
                                <option value="write">Read & Write</option>
                            {/if}
                        </select>
                    </label>
                </div>
                
                <button type="submit" class="px-3 py-3 rounded-lg bg-accent-500 text-white font-medium hover:bg-accent-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed" disabled={isSharing}>
                    {#if isSharing}
                        Sharing...
                    {:else}
                        <div class="flex items-center gap-2 justify-center">
                            <Share size=16 />
                            Share {shareModalItem ? 'Item' : `${selectedItems.size} Items`}
                        </div>
                    {/if}
                </button>
            </form>
            <button type="button" class="absolute top-3 right-3 p-1 text-primary-400 hover:text-primary-200" on:click={closeShareModal}>
                <X size=20 />
            </button>
        </div>
	</div>
{/if}

<!-- Selection Action Bar -->
{#if inSelectionMode}
    <div class="fixed bottom-8 left-1/2 transform -translate-x-1/2 max-w-2xl bg-primary-800 space-x-4 text-white rounded-xl  px-6 py-4 flex justify-between items-center z-50 shadow-2xl border border-primary-600" transition:fly={{ y: 20, duration: 300 }}>
        <div class="flex items-center gap-4">
            <span class="font-medium">{selectedItems.size} selected</span>
            <button class="text-primary-400 hover:text-primary-200 transition-colors" on:click={() => selectedItems = new Set()}>
				Deselect all
			</button>
        </div>
        <div class="flex gap-3">
            <button class="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary-700 text-white font-medium border border-primary-600 hover:bg-primary-600 transition-colors" on:click={() => { shareModalItem = null; showShareModal = true; }}>
				<Share size=16/> Share
			</button>
            <button class="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary-700 text-white font-medium border border-primary-600 hover:bg-primary-600 transition-colors" on:click={handleBulkDownload}>
				<Download size=16/> Download
			</button>
            <button class="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary-700 text-white font-medium border border-primary-600 hover:bg-accent-500 hover:border-accent-500 transition-colors" on:click={handleBulkDelete}>
				<Trash2 size=16/> Move to Trash
			</button>
        </div>
    </div>
{/if}

<!-- Main Content Area -->
<div class="relative h-full text-primary-50" on:dragover={handleUploadDragOver} on:dragleave={handleUploadDragLeave} on:drop={handleUploadDrop} role="region" aria-label="Files main content area">
    {#if isDraggingOver}
        <div class="absolute -inset-8 bg-primary-900 bg-opacity-95 border-4 border-dashed border-accent-500 rounded-xl flex items-center justify-center z-50 pointer-events-none">
			<div class="flex flex-col items-center gap-4 text-accent-500 text-2xl font-medium">
				<UploadCloud size=64 />
				<span>Drop files to upload</span>
			</div>
		</div>
    {/if}
    
	<div in:fade|local>
		<div class="flex justify-between items-center mb-6">
			<div class="flex items-center gap-1 text-sm">
				{#each breadcrumbs as crumb, i}
					<a href={crumb.path} class="flex items-center gap-2 text-primary-400 px-2 py-2 rounded-md hover:bg-primary-800 transition-colors">
						{#if i === 0}
							<Home size=16/>
						{:else}
							<span>{crumb.name}</span>
						{/if}
					</a>
					{#if i < breadcrumbs.length - 1}
						<ChevronRight size=16 class="text-primary-600" />
					{/if}
				{/each}
			</div>
			<div class="flex gap-3">
				<button class="flex items-center gap-2 px-5 py-3 rounded-lg font-medium bg-primary-700 text-primary-50 border border-primary-600 hover:bg-primary-600 hover:border-primary-500 hover:-translate-y-0.5 transition-all" on:click={() => showCreateFolderModal = true}>
					<Plus size=16/> New Folder
				</button>
				<label class="flex items-center gap-2 px-5 py-3 rounded-lg font-medium cursor-pointer bg-primary-700 text-primary-50 border border-primary-600 hover:bg-primary-600 hover:border-primary-500 hover:-translate-y-0.5 transition-all">
					<Upload size=16/> Upload Folder
					<input type="file" class="hidden" on:change={handleFolderSelect} webkitdirectory />
				</label>
				<label class="flex items-center gap-2 px-5 py-3 rounded-lg font-medium cursor-pointer bg-accent-500 text-white hover:bg-accent-600 hover:-translate-y-0.5 hover:shadow-lg hover:shadow-accent-500/20 transition-all">
					<UploadCloud size=16/> Upload Files
					<input type="file" class="hidden" on:change={handleFileSelect} multiple />
				</label>
			</div>
		</div>
		
		<!-- Storage Quota Display -->
		{#if quotaInfo.quotaLimit > 0}
			<div class="mb-6 bg-primary-800 border border-primary-600 rounded-lg p-4">
				<div class="flex justify-between items-center mb-2">
					<span class="text-sm text-primary-300">Storage Usage</span>
					<span class="text-sm text-primary-400">
						{formatBytes(quotaInfo.quotaUsed)} / {formatBytes(quotaInfo.quotaLimit)}
					</span>
				</div>
				<div class="w-full h-2 bg-primary-700 rounded-full overflow-hidden">
					<div 
						class="h-full transition-all duration-300 {quotaInfo.quotaUsed / quotaInfo.quotaLimit > 0.9 ? 'bg-red-500' : quotaInfo.quotaUsed / quotaInfo.quotaLimit > 0.7 ? 'bg-yellow-500' : 'bg-accent-500'}"
						style="width: {Math.min(100, (quotaInfo.quotaUsed / quotaInfo.quotaLimit) * 100)}%"
					></div>
				</div>
			</div>
		{/if}
		
		{#if error_message}
			<div class="bg-accent-500 bg-opacity-10 text-accent-300 border border-accent-500 px-4 py-4 rounded-lg mb-6 flex items-center gap-3">
				<AlertCircle size=20 />
				{error_message}
			</div>
		{/if}
		
		{#if isUploading && uploadQueue.length > 0}
			<div class="mb-8 bg-primary-800 border border-primary-600 rounded-xl shadow-lg">
				<div class="px-6 py-4 border-b border-primary-700">
					<span class="font-medium">Uploading {uploadQueue.length} items</span>
					<div class="w-full h-2 bg-primary-700 rounded-full overflow-hidden mt-3">
						<div class="h-full bg-accent-500 rounded-full transition-all duration-300" style="width: {totalUploadProgress}%"></div>
					</div>
				</div>
				<div class="max-h-48 overflow-y-auto p-2 scrollbar-thin scrollbar-track-primary-800 scrollbar-thumb-primary-600">
					{#each uploadQueue as upload (upload.id)}
						<div class="grid grid-cols-[auto_1fr_auto] items-center gap-4 px-4 py-3">
							<div class="flex-shrink-0">
								{#if upload.status==='uploading'||upload.status==='finalizing'||upload.status==='preparing'}
									<UploadCloud size=20 class="animate-spin text-accent-400" />
								{:else if upload.status==='done'}
									<CheckCircle size=20 class="text-green-500" />
								{:else if upload.status==='error'}
									<XCircle size=20 class="text-red-400" />
								{/if}
							</div>
							<div class="min-w-0">
								<span class="block text-sm text-primary-300 truncate" title={upload.path}>{upload.path}</span>
								{#if upload.status!=='error'}
									<div class="w-full h-1.5 bg-primary-700 rounded-full overflow-hidden mt-1">
										<div class="h-full rounded-full transition-all duration-300 {upload.status === 'done' ? 'bg-green-500' : 'bg-red-400'}" style="width: {upload.progress}%"></div>
									</div>
								{:else}
									<span class="text-xs text-red-400 truncate" title={upload.error}>{upload.error}</span>
								{/if}
							</div>
							<div class="text-sm font-medium text-primary-300">
								{#if upload.status==='done'}
									Done
								{:else if upload.status==='error'}
									Failed
								{:else if upload.status==='preparing'}
									Preparing...
								{:else}
									{Math.round(upload.progress)}%
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
		
		{#if recentFiles.length > 0 && currentPath === ''}
			<h2 class="flex items-center gap-3 text-xl font-semibold mt-8 mb-4 text-primary-50">
				<Clock size=20 />
				<span>Recent Files</span>
			</h2>
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
				{#each recentFiles as file (file.id)}
					<!-- --- MODIFIED: Using file.name instead of file.originalName --- -->
					<div class="bg-primary-800 border border-primary-600 rounded-xl p-4 grid grid-cols-[auto_1fr_auto] items-center gap-4 overflow-hidden hover:border-primary-500 transition-colors group">
						<div class="flex-shrink-0">
							<FileText size=28 class="text-primary-400" />
						</div>
						<div class="min-w-0 flex flex-col">
							<span class="font-medium truncate text-primary-50" title={file.name}>{file.name}</span>
							<span class="text-xs text-primary-400">{formatBytes(file.size ?? 0)}</span>
						</div>
						<div class="flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
							<button class="p-1 text-primary-400 hover:text-accent-500 transition-colors" on:click|preventDefault|stopPropagation={() => handleDownload(file)} title="Download file">
								<Download size=18 />
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
		
		<h2 class="text-xl font-semibold mb-4 text-primary-50">All Files & Folders</h2>
		<div class="border border-primary-600 rounded-xl overflow-hidden bg-primary-800">
			<div class="grid grid-cols-[auto_3fr_1fr_1.5fr_auto] px-6 py-3 bg-primary-900 text-xs uppercase text-primary-400">
				<div class="pr-4 flex items-center">
					<input type="checkbox" on:change={toggleSelectAll} bind:this={selectAllCheckbox} checked={allSelected} 
						   class="w-4 h-4 cursor-pointer bg-primary-700 border border-primary-600 rounded appearance-none checked:bg-accent-500 checked:border-accent-500 transition-colors" />
				</div>
				<div>Name</div>
				<div>Size</div>
				<div>Modified</div>
				<div></div>
			</div>
			
			{#each folders as folder (folder.id)}
				<div class="grid grid-cols-[auto_3fr_1fr_1.5fr_auto] items-center px-6 py-4 border-b border-primary-700 last:border-b-0 hover:bg-primary-700 transition-colors cursor-pointer group {selectedItems.has(folder.id) ? 'bg-primary-600' : ''} {draggedItem && draggedItem.id !== folder.id ? 'outline-2 outline-dashed outline-accent-500' : ''}" 
					 on:click={()=>goto(`/files?path=${folder.path}`)} 
					 on:dragover|preventDefault 
					 on:drop|preventDefault={()=>handleDrop(folder)} 
					 role="row" 
					 tabindex="0" 
					 aria-label={`Open folder ${folder.name}`} 
					 on:keydown={(e) => { if (e.key === 'Enter') goto(`/files?path=${folder.path}`); }}>
					

					<button aria-label={`Select folder ${folder.name}`} class="pr-4 flex items-center" on:click|stopPropagation>
						<input type="checkbox"
								id="folder-checkbox"
							   checked={selectedItems.has(folder.id)} 
							   on:change={()=>toggleSelect(folder.id)}
							   class="w-4 h-4 cursor-pointer bg-primary-700 border border-primary-600 rounded appearance-none checked:bg-accent-500 checked:border-accent-500 transition-colors"/>
					</button>
					
					<div class="flex items-center gap-4 font-medium text-primary-50">
						<Folder size=20 class="text-blue-400"/>
						<span>{folder.name}</span>
					</div>
					
					<div class="text-primary-300">--</div>
					
					<div class="text-primary-300">{formatDistanceToNow(new Date(folder.modified), {locale:th, addSuffix:true})}</div>
					
					<div class="flex justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
						{#if currentPath}
							<button class="p-1 text-primary-400 hover:text-accent-500 transition-colors" on:click|preventDefault|stopPropagation={() => handleMoveUp(folder)} title="Move Up">
								<CornerLeftUp size=18/>
							</button>
						{/if}
						<button class="p-1 text-primary-400 hover:text-accent-500 transition-colors" on:click|preventDefault|stopPropagation={() => openShareModal(folder)} title="Share folder">
							<Share size=18/>
						</button>
						<button class="p-1 text-primary-400 hover:text-accent-500 transition-colors" on:click|preventDefault|stopPropagation={() => handleDownload(folder)} title="Download folder">
							<Download size=18/>
						</button>
					</div>
				</div>
			{/each}
			
			{#each files as file (file.id)}
				<!-- Add tabindex="0" to make the interactive row focusable -->
				<div class="grid grid-cols-[auto_3fr_1fr_1.5fr_auto] items-center px-6 py-4 border-b border-primary-700 last:border-b-0 hover:bg-primary-700 transition-colors group {selectedItems.has(file.id) ? 'bg-primary-600' : ''} {draggedItem?.id === file.id ? 'opacity-50' : ''}"
					draggable="true"
					on:dragstart={()=>handleDragStart(file)}
					on:dragend={handleDragEnd}
					role="row"
					tabindex="0">
					<label class="pr-4 flex items-center cursor-pointer">
						<input type="checkbox" 
							checked={selectedItems.has(file.id)} 
							on:change={()=>toggleSelect(file.id)}
							class="w-4 h-4 cursor-pointer bg-primary-700 border border-primary-600 rounded appearance-none checked:bg-accent-500 checked:border-accent-500 transition-colors"/>
					</label>
					
					<div class="flex items-center gap-4 font-medium text-primary-50">
						<FileText size=20 class="text-primary-400"/>
						<span>{file.name}</span>
					</div>
					
					<div class="text-primary-300">{formatBytes(file.size ?? 0)}</div>
					
					<div class="text-primary-300">{formatDistanceToNow(new Date(file.modified ?? Date.now()), {locale:th, addSuffix:true})}</div>
					
					<div class="flex justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
						{#if currentPath}
							<button class="text-primary-400 hover:text-accent-500 transition-colors" on:click|preventDefault|stopPropagation={() => handleMoveUp(file)} title="Move Up">
								<CornerLeftUp size=18/>
							</button>
						{/if}
						<button class=" text-primary-400 hover:text-accent-500 transition-colors cursor-pointer" on:click|preventDefault|stopPropagation={() => openShareModal(file)} title="Share file">
							<Share size=20/>
						</button>
						<button class=" text-primary-400 hover:text-accent-500 transition-colors cursor-pointer" on:click|preventDefault|stopPropagation={() => handleDownload(file)} title="Download file">
							<Download size=20/>
						</button>
					</div>
				</div>
			{/each}
		</div>
		
		{#if folders.length === 0 && files.length === 0 && !isUploading} 
			<div class="text-center py-16 text-primary-400">
				<Folder size=48 class="mx-auto mb-4" />
				<h3 class="text-xl font-semibold text-primary-50">Folder is empty</h3>
			</div> 
		{/if}
	</div>
</div>