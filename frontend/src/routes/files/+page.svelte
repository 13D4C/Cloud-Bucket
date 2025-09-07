<script lang="ts">
    import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import * as tus from 'tus-js-client';
	import { Folder, FileText, UploadCloud, Home, Trash2, ChevronRight, Download, X, AlertCircle, Plus, Clock, CheckCircle, XCircle, RotateCcw, Upload, CornerLeftUp } from 'lucide-svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { th } from 'date-fns/locale';
    import { fetchApi } from '$lib/api';
	import { afterUpdate } from 'svelte';
    import { fly, fade } from 'svelte/transition';

	interface FileItem {
		path: string;
		isDir: boolean;
		name: string;
		originalName?: string;
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
    let draggedItem: any = null;
    let isDraggingOver = false;
    let uploadQueue: { id: number; file: File; progress: number; status: 'uploading' | 'preparing' | 'finalizing' | 'done' | 'error'; error?: string; path?: string; }[] = [];
    let totalUploadProgress = 0;
    let isUploading = false;
    let selectedItems = new Set<string>();
    let selectAllCheckbox: HTMLInputElement;

	// --- Computed State ---
	$: currentPath = $page.url.searchParams.get('path') || '';
	$: inSelectionMode = selectedItems.size > 0;
    $: allSelected = items.length > 0 && selectedItems.size === items.length;
	$: breadcrumbs = [{ name: 'My Files', path: '/files' }].concat(
		currentPath.split('/').filter(p => p).map((part, i, arr) => ({ name: part, path: `/files?path=${arr.slice(0, i + 1).join('/')}` }))
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
			folders = allItems.filter((item: FileItem) => item.isDir).sort((a: FileItem, b: FileItem) => a.name.localeCompare(b.name));
			files = allItems.filter((item: FileItem) => !item.isDir).sort((a: FileItem, b: FileItem) => (a.originalName || '').localeCompare(b.originalName || ''));
            items = [...folders, ...files];
		} catch (error: any) {
			console.error("Fetch Error:", error);
			error_message = `Could not load items: ${error.message}`;
		}
	}
	$: if ($page.url) { fetchData(); selectedItems = new Set(); }

	// --- Handlers for Selection ---
    function toggleSelect(path: string) {
        const newSelectedItems = new Set(selectedItems);
        if (newSelectedItems.has(path)) {
            newSelectedItems.delete(path);
        } else {
            newSelectedItems.add(path);
        }
        selectedItems = newSelectedItems;
    }
    async function handleBulkDelete() {
        if (!confirm(`Are you sure you want to move ${selectedItems.size} items to the trash?`)) return;
        try {
            await fetchApi('/api/items/bulk-delete', {
                method: 'POST',
                body: JSON.stringify({ paths: Array.from(selectedItems) })
            });
            selectedItems = new Set();
            await fetchData();
        } catch (e: any) {
            alert(`Error moving items to trash: ${e.message}`);
        }
    }
    function toggleSelectAll() {
        if (allSelected) {
            selectedItems = new Set();
        } else {
            selectedItems = new Set(items.map(item => item.path));
        }
    }
    async function handleBulkDownload() {
        if (selectedItems.size === 0) return alert("Please select items to download.");
        try {
            const res = await fetchApi('/api/items/bulk-download', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ paths: Array.from(selectedItems) })
            });
            if (!res.ok) {
                const errorData = await res.json();
                throw new Error(errorData.error || `Server responded with status ${res.status}`);
            }
            const blob = await res.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.style.display = 'none';
            a.href = url;
            a.download = `IT-Cloud-${new Date().toISOString().slice(0, 10)}.zip`;
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

	// --- Drag & Drop Upload Handlers ---
    function handleUploadDragOver(event: DragEvent) {
        event.preventDefault();
        if (!draggedItem) { isDraggingOver = true; }
    }
    function handleUploadDragLeave() { isDraggingOver = false; }
    function handleUploadDrop(event: DragEvent) {
        event.preventDefault();
        isDraggingOver = false;
        if (!draggedItem && event.dataTransfer?.files) {
            startMultipleUploads(event.dataTransfer.files);
        }
    }

	// --- Upload Logic ---
	function handleFileSelect(event: Event) {
        const input = event.target as HTMLInputElement;
		if (input.files && input.files.length > 0) { startMultipleUploads(input.files); }
		input.value = '';
	}
    function handleFolderSelect(event: Event) {
        const input = event.target as HTMLInputElement;
		if (input.files && input.files.length > 0) { startMultipleUploads(input.files); }
		input.value = '';
    }
	async function startMultipleUploads(fileList: FileList) {
        isUploading = true;
        const newUploads = Array.from(fileList).map(file => ({
            id: Date.now() + Math.random(), file, progress: 0, status: 'preparing' as const, path: (file as any).webkitRelativePath || file.name
        }));
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
            const createFolderPromises = Array.from(dirPaths).map(path => 
                fetchApi(`/api/folders/structure`, { method: 'POST', body: JSON.stringify({ path: `${currentPath}/${path}`.replace(/^\//, '') }) })
            );
            await Promise.allSettled(createFolderPromises);
        }
        uploadQueue.forEach(item => { if (item.status === 'preparing') item.status = 'uploading'; });
        uploadQueue = [...uploadQueue];
        const uploadPromises = newUploads.map(uploadItem => startSingleUpload(uploadItem));
        await Promise.allSettled(uploadPromises);
        await fetchData();
        setTimeout(() => {
            uploadQueue = uploadQueue.filter(item => item.status !== 'done' && item.status !== 'error');
            if (uploadQueue.length === 0) { isUploading = false; }
        }, 5000);
    }
    function startSingleUpload(uploadItem: typeof uploadQueue[0]) {
        return new Promise((resolve, reject) => {
            const pathParts = (uploadItem.path || '').split('/');
            pathParts.pop();
            const folderPath = pathParts.join('/');
            const destinationPath = currentPath ? `${currentPath}/${folderPath}`.replace(/^\//, '') : folderPath;
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
                        reject(new Error('Finalize ID missing'));
                        return;
                    }
                    try {
                        await fetchApi(`/api/finalize-upload`, { method: 'POST', body: JSON.stringify({ uploadId, destinationPath }) });
                        index = uploadQueue.findIndex(item => item.id === uploadItem.id);
                        if (index !== -1) { uploadQueue[index].status = 'done'; uploadQueue = [...uploadQueue]; }
                        resolve(true);
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
    function handleDragStart(item: any) { draggedItem = item; }
    function handleDragEnd() { draggedItem = null; }
    async function handleDrop(destinationFolder: any) { if (draggedItem && draggedItem.path !== destinationFolder.path) { await handleMove(draggedItem, destinationFolder); } handleDragEnd(); }
    async function handleMove(sourceItem: any, destinationFolder: any) { await fetchApi(`/api/move`,{method:'POST',body:JSON.stringify({sourcePath:sourceItem.path,destinationFolder:destinationFolder.path})}); await fetchData(); }
	async function handleCreateFolder() { if (!newFolderName.trim()) return; await fetchApi(`/api/folders?path=${encodeURIComponent(currentPath)}`,{method:'POST',body:JSON.stringify({folderName:newFolderName.trim()})}); showCreateFolderModal=false; newFolderName=''; await fetchData(); }
    async function handleMoveUp(item: any) {
        if (!currentPath) return;
        const pathParts = currentPath.split('/');
        pathParts.pop();
        const destinationFolder = pathParts.join('/');
        const itemName = item.originalName || item.name;
        if (!confirm(`Move "${itemName}" up one level?`)) return;
        try {
            await fetchApi('/api/move', {
                method: 'POST',
                body: JSON.stringify({
                    sourcePath: item.path,
                    destinationFolder: destinationFolder 
                })
            });
            await fetchData();
        } catch (e: any) {
            alert(`Error moving item: ${e.message}`);
        }
    }
    async function handleDownload(item: any) {
        try {
            const token = localStorage.getItem('jwt_token');
            const endpoint = item.isDir ? 'download-folder' : 'download';
            const downloadUrl = `http://localhost:8080/api/${endpoint}/${item.path}?token=${token}`;
            const res = await fetch(downloadUrl);
            if (!res.ok) {
                let errorMessage = `Server responded with status ${res.status}`;
                try { const errorData = await res.json(); errorMessage = errorData.error || errorData.message || errorMessage; } catch (e) { errorMessage = await res.text(); }
                throw new Error(errorMessage);
            }
            const blob = await res.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.style.display = 'none';
            a.href = url;
            a.download = item.originalName || `${item.name}.zip`;
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
				{#each recentFiles as file (file.path)}
					<div class="bg-primary-800 border border-primary-600 rounded-xl p-4 grid grid-cols-[auto_1fr_auto] items-center gap-4 overflow-hidden hover:border-primary-500 transition-colors">
						<div class="flex-shrink-0">
							<FileText size=28 class="text-primary-400" />
						</div>
						<div class="min-w-0 flex flex-col">
							<span class="font-medium truncate text-primary-50" title={file.originalName}>{file.originalName}</span>
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
			
			{#each folders as folder (folder.path)}
				<div class="grid grid-cols-[auto_3fr_1fr_1.5fr_auto] items-center px-6 py-4 border-b border-primary-700 last:border-b-0 hover:bg-primary-700 transition-colors cursor-pointer {selectedItems.has(folder.path) ? 'bg-primary-600' : ''} {draggedItem && draggedItem.path !== folder.path ? 'outline-2 outline-dashed outline-accent-500' : ''}" 
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
							   checked={selectedItems.has(folder.path)} 
							   on:change={()=>toggleSelect(folder.path)}
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
						<button class="p-1 text-primary-400 hover:text-accent-500 transition-colors" on:click|preventDefault|stopPropagation={() => handleDownload(folder)} title="Download folder">
							<Download size=18/>
						</button>
					</div>
				</div>
			{/each}
			
			{#each files as file (file.path)}
				<!-- Add tabindex="0" to make the interactive row focusable -->
				<div class="grid grid-cols-[auto_3fr_1fr_1.5fr_auto] items-center px-6 py-4 border-b border-primary-700 last:border-b-0 hover:bg-primary-700 transition-colors {selectedItems.has(file.path) ? 'bg-primary-600' : ''} {draggedItem?.path === file.path ? 'opacity-50' : ''}"
					draggable="true"
					on:dragstart={()=>handleDragStart(file)}
					on:dragend={handleDragEnd}
					role="row"
					tabindex="0">
					<label class="pr-4 flex items-center cursor-pointer">
						<input type="checkbox" 
							checked={selectedItems.has(file.path)} 
							on:change={()=>toggleSelect(file.path)}
							class="w-4 h-4 cursor-pointer bg-primary-700 border border-primary-600 rounded appearance-none checked:bg-accent-500 checked:border-accent-500 transition-colors"/>
					</label>
					
					<div class="flex items-center gap-4 font-medium text-primary-50">
						<FileText size=20 class="text-primary-400"/>
						<span>{file.originalName}</span>
					</div>
					
					<div class="text-primary-300">{formatBytes(file.size ?? 0)}</div>
					
					<div class="text-primary-300">{formatDistanceToNow(new Date(file.modified ?? Date.now()), {locale:th, addSuffix:true})}</div>
					
					<div class="flex justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
						{#if currentPath}
							<button class="p-1 text-primary-400 hover:text-accent-500 transition-colors" on:click|preventDefault|stopPropagation={() => handleMoveUp(file)} title="Move Up">
								<CornerLeftUp size=18/>
							</button>
						{/if}
						<button class="p-1 text-primary-400 hover:text-accent-500 transition-colors" on:click|preventDefault|stopPropagation={() => handleDownload(file)} title="Download file">
							<Download size=18/>
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