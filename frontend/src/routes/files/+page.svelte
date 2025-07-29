<script lang="ts">
    import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import * as tus from 'tus-js-client';
	import { Folder, FileText, UploadCloud, Trash2, Home, ChevronRight, Download, X, AlertCircle, Plus, Clock, CheckCircle, XCircle, RotateCcw, Upload, CornerLeftUp } from 'lucide-svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { th } from 'date-fns/locale';
    import { fetchApi } from '$lib/api';
	import { onMount, afterUpdate } from 'svelte';
    import { fly, fade } from 'svelte/transition';

	let files: any[] = [];
	let folders: any[] = [];
	let items: any[] = [];
	let recentFiles: any[] = [];
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
	$: isTrashView = $page.url.searchParams.get('view') === 'trash';
	$: currentPath = isTrashView ? '' : ($page.url.searchParams.get('path') || '');
	$: inSelectionMode = selectedItems.size > 0;
    $: allSelected = items.length > 0 && selectedItems.size === items.length;
	$: breadcrumbs = [{ name: 'My Files', path: '/files' }].concat(
		currentPath.split('/').filter(p => p).map((part, i, arr) => ({ name: part, path: `/files?path=${arr.slice(0, i + 1).join('/')}` }))
	);
    $: {
        if (files) {
            recentFiles = [...files]
                .sort((a, b) => new Date(b.modified).getTime() - new Date(a.modified).getTime())
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
			const endpoint = isTrashView ? '/api/trash' : `/api/files?path=${encodeURIComponent(currentPath)}`;
			const res = await fetchApi(endpoint);
			if (!res.ok) {
                const errData = await res.json();
                throw new Error(errData.error || 'Failed to fetch items');
            }
			const allItems = await res.json() || [];
			folders = allItems.filter(item => item.isDir).sort((a,b) => a.name.localeCompare(b.name));
			files = allItems.filter(item => !item.isDir).sort((a,b) => (a.originalName || '').localeCompare(b.originalName || ''));
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
    function toggleSelectAll() {
        if (allSelected) {
            selectedItems = new Set();
        } else {
            selectedItems = new Set(items.map(item => item.path));
        }
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
        if (!draggedItem && !isTrashView) { isDraggingOver = true; }
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
        if (isTrashView) return;
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
	async function handleSoftDelete(item: any) { if (!confirm(`Move "${item.originalName||item.name}" to trash?`)) return; await fetchApi(`/api/items/${item.path}`,{method:'DELETE'}); await fetchData(); }
	async function handleRestore(item: any) { if (!confirm(`Restore "${item.originalName||item.name}"?`)) return; await fetchApi('/api/trash/restore',{method:'POST',body:JSON.stringify({path:item.path})}); await fetchData(); }
	async function handlePermanentDelete(item: any) { if (!confirm(`Permanently delete "${item.originalName||item.name}"? This cannot be undone.`)) return; await fetchApi(`/api/trash/${item.path}`,{method:'DELETE'}); await fetchData(); }
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

<!-- HTML Template เหมือนเดิมทั้งหมด -->
<!-- New Folder Modal -->
{#if showCreateFolderModal}
	<div class="modal-backdrop" on:click={() => showCreateFolderModal = false}>
		<div class="modal-content" transition:fly={{ y: -20, duration: 300 }} on:click|stopPropagation>
			<h3>New Folder</h3>
			<form on:submit|preventDefault={handleCreateFolder}>
				<input type="text" bind:value={newFolderName} placeholder="Enter folder name..." required />
				<button type="submit" class="primary">Create Folder</button>
			</form>
			<button class="close-modal-btn" on:click={() => showCreateFolderModal = false}><X size=20 /></button>
		</div>
	</div>
{/if}

<!-- Selection Action Bar -->
{#if inSelectionMode && !isTrashView}
    <div class="selection-bar" transition:fly={{ y: 20, duration: 300 }}>
        <div class="selection-info">
            <span>{selectedItems.size} selected</span>
            <button class="deselect-btn" on:click={() => selectedItems = new Set()}>Deselect all</button>
        </div>
        <div class="selection-actions">
            <button class="action-btn-sm" on:click={handleBulkDownload}><Download size=16/> Download</button>
            <button class="action-btn-sm delete" on:click={handleBulkDelete}><Trash2 size=16/> Move to Trash</button>
        </div>
    </div>
{/if}

<!-- Main Content Area -->
<div class="main-content-area" on:dragover={handleUploadDragOver} on:dragleave={handleUploadDragLeave} on:drop={handleUploadDrop}>
    {#if isDraggingOver}
        <div class="dropzone-overlay"><div class="dropzone-message"><UploadCloud size=64 /><span>Drop files to upload</span></div></div>
    {/if}
    {#if isTrashView}
        <div in:fade|local>
            <div class="page-header"><h1>Trash</h1></div>
            <p class="subtitle">Items in trash can be restored or deleted forever.</p>
            {#if error_message}<div class="error-banner">{error_message}</div>{/if}
            <div class="file-grid-container is-trash">
                <div class="grid-header"><div>Name</div><div>Actions</div></div>
                {#each items as item (item.path)}
                    <div class="grid-row" transition:fade|local>
                        <div class="item-name">
                            {#if item.isDir}<Folder size=20 color="#4f86c2"/>{:else}<FileText size=20 color="#9ca3af"/>{/if}
                            <span>{item.originalName || item.name}</span>
                        </div>
                        <div class="row-actions">
                            <button class="action-icon" title="Restore" on:click={() => handleRestore(item)}><RotateCcw size=18/></button>
                            <button class="action-icon delete" title="Delete Forever" on:click={() => handlePermanentDelete(item)}><Trash2 size=18/></button>
                        </div>
                    </div>
                {:else}
                    <div class="empty-state"><Trash2 size=48/><h3>Trash is empty</h3></div>
                {/each}
            </div>
        </div>
    {:else}
        <div in:fade|local>
            <div class="page-header">
                <div class="breadcrumbs">{#each breadcrumbs as crumb, i}<a href={crumb.path}>{#if i === 0}<Home size=16/>{:else}<span>{crumb.name}</span>{/if}</a>{#if i < breadcrumbs.length - 1}<ChevronRight size=16 class="separator"/>{/if}{/each}</div>
                <div class="actions">
                    <button class="action-btn secondary" on:click={() => showCreateFolderModal = true}><Plus size=16/> New Folder</button>
                    <label class="action-btn secondary"><Upload size=16/> Upload Folder<input type="file" hidden on:change={handleFolderSelect} webkitdirectory /></label>
                    <label class="action-btn primary"><UploadCloud size=16/> Upload Files<input type="file" hidden on:change={handleFileSelect} multiple /></label>
                </div>
            </div>
            {#if error_message}<div class="error-banner">{error_message}</div>{/if}
            {#if isUploading && uploadQueue.length > 0}
                <div class="upload-status-area">
                    <div class="upload-header"><span>Uploading {uploadQueue.length} items</span><div class="total-progress-bar-container"><div class="total-progress-bar" style="width: {totalUploadProgress}%"></div></div></div>
                    <div class="upload-list">{#each uploadQueue as upload (upload.id)}<div class="upload-item"><div class="upload-icon">{#if upload.status==='uploading'||upload.status==='finalizing'||upload.status==='preparing'}<UploadCloud size=20 class="spinner"/>{:else if upload.status==='done'}<CheckCircle size=20 color="#22c55e"/>{:else if upload.status==='error'}<XCircle size=20 color="#ef4444"/>{/if}</div><div class="upload-details"><span class="upload-name" title={upload.path}>{upload.path}</span>{#if upload.status!=='error'}<div class="item-progress-bar-container"><div class="item-progress-bar" style="width: {upload.progress}%" class:done={upload.status==='done'}></div></div>{:else}<span class="upload-error-text" title={upload.error}>{upload.error}</span>{/if}</div><div class="upload-progress-text">{#if upload.status==='done'}Done{:else if upload.status==='error'}Failed{:else if upload.status==='preparing'}Preparing...{:else}{Math.round(upload.progress)}%{/if}</div></div>{/each}</div>
                </div>
            {/if}
            {#if recentFiles.length > 0 && currentPath === ''}
                <h2 class="section-title"><Clock size=20 /><span>Recent Files</span></h2>
                <div class="recents-grid">{#each recentFiles as file (file.path)}<div class="recent-card"><div class="card-icon"><FileText size=28 color="#9ca3af"/></div><div class="card-details"><span class="card-name" title={file.originalName}>{file.originalName}</span><span class="card-meta">{formatBytes(file.size)}</span></div><div class="card-actions"><button class="action-icon" on:click|preventDefault|stopPropagation={() => handleDownload(file)} title="Download file"><Download size=18 /></button><button class="action-icon delete" on:click|preventDefault|stopPropagation={()=>handleSoftDelete(file)}><Trash2 size=18 /></button></div></div>{/each}</div>
            {/if}
            <h2 class="section-title">All Files & Folders</h2>
            <div class="file-grid-container">
                <div class="grid-header">
                    <div class="header-select"><input type="checkbox" on:change={toggleSelectAll} bind:this={selectAllCheckbox} checked={allSelected} /></div>
                    <div>Name</div><div>Size</div><div>Modified</div><div></div>
                </div>
                {#each folders as folder (folder.path)}
                    <div class="grid-row is-folder" class:selected={selectedItems.has(folder.path)} class:drop-target={draggedItem&&draggedItem.path!==folder.path} on:click={()=>goto(`/files?path=${folder.path}`)} on:dragover|preventDefault on:drop|preventDefault={()=>handleDrop(folder)}>
                        <div class="row-select" on:click|stopPropagation><input type="checkbox" checked={selectedItems.has(folder.path)} on:change={()=>toggleSelect(folder.path)}/></div>
                        <div class="item-name"><Folder size=20 color="#4f86c2"/><span>{folder.name}</span></div>
                        <div class="item-size">--</div><div class="item-modified">{formatDistanceToNow(new Date(folder.modified),{locale:th,addSuffix:true})}</div>
                        <div class="row-actions">
                            {#if currentPath}<button class="action-icon" on:click|preventDefault|stopPropagation={() => handleMoveUp(folder)} title="Move Up"><CornerLeftUp size=18/></button>{/if}
                            <button class="action-icon" on:click|preventDefault|stopPropagation={() => handleDownload(folder)} title="Download folder"><Download size=18/></button>
                            <button class="action-icon delete" on:click|preventDefault|stopPropagation={()=>handleSoftDelete(folder)}><Trash2 size=18/></button>
                        </div>
                    </div>
                {/each}
                {#each files as file (file.path)}
                    <div class="grid-row" class:selected={selectedItems.has(file.path)} class:dragging={draggedItem?.path===file.path} draggable="true" on:dragstart={()=>handleDragStart(file)} on:dragend={handleDragEnd}>
                        <div class="row-select" on:click|stopPropagation><input type="checkbox" checked={selectedItems.has(file.path)} on:change={()=>toggleSelect(file.path)}/></div>
                        <div class="item-name"><FileText size=20 color="#9ca3af"/><span>{file.originalName}</span></div>
                        <div class="item-size">{formatBytes(file.size)}</div><div class="item-modified">{formatDistanceToNow(new Date(file.modified),{locale:th,addSuffix:true})}</div>
                        <div class="row-actions">
                            {#if currentPath}<button class="action-icon" on:click|preventDefault|stopPropagation={() => handleMoveUp(file)} title="Move Up"><CornerLeftUp size=18/></button>{/if}
                            <button class="action-icon" on:click|preventDefault|stopPropagation={() => handleDownload(file)} title="Download file"><Download size=18/></button>
                            <button class="action-icon delete" on:click|preventDefault|stopPropagation={()=>handleSoftDelete(file)}><Trash2 size=18/></button>
                        </div>
                    </div>
                {/each}
            </div>
            {#if folders.length === 0 && files.length === 0 && !isUploading} 
                <div class="empty-state"><Folder size=48/><h3>Folder is empty</h3></div> 
            {/if}
        </div>
    {/if}
</div>

<style>
 .main-content-area { 
        position: relative; 
        height: 100%; 
        color: var(--text-primary); 
    }
    .dropzone-overlay { 
        position: absolute; 
        inset: -2rem; 
        background-color: rgba(17, 24, 39, 0.95); 
        border: 3px dashed var(--accent-primary); 
        border-radius: 12px; 
        display: grid; 
        place-items: center; 
        z-index: 99; 
        pointer-events: none; 
    }
    .dropzone-message { 
        display: flex; 
        flex-direction: column; 
        align-items: center; 
        gap: 1rem; 
        color: var(--accent-primary); 
        font-size: 1.5rem; 
        font-weight: 500; 
    }
    
    .page-header { 
        display: flex; 
        justify-content: space-between; 
        align-items: center; 
        margin-bottom: 1.5rem; 
    }
    h1 { 
        font-size: 1.75rem; 
        margin: 0; 
        color: var(--text-primary); 
    }
    .subtitle { 
        color: var(--text-secondary); 
        margin-top: 0.25rem; 
        margin-bottom: 2rem; 
    }
    
    .breadcrumbs { 
        display: flex; 
        align-items: center; 
        gap: 0.25rem; 
        font-size: 0.9rem; 
    }
    .breadcrumbs a { 
        display: flex; 
        align-items: center; 
        gap: 0.5rem; 
        text-decoration: none; 
        color: var(--text-muted); 
        padding: 0.25rem 0.5rem; 
        border-radius: 6px; 
        transition: background-color 0.2s ease;
    }
    .breadcrumbs a:hover { 
        background-color: var(--bg-secondary); 
    }
    .separator { 
        color: var(--border-color); 
    }
    
    .actions { 
        display: flex; 
        gap: 0.75rem; 
    }

    .action-btn { 
        padding: 0.6rem 1.2rem; 
        border-radius: 8px; 
        font-weight: 500; 
        cursor: pointer; 
        display: flex; 
        align-items: center; 
        gap: 0.6rem;
        border: none;
        transition: all 0.2s ease-in-out;
    }
    .action-btn.primary { 
        background-color: var(--accent-primary); 
        color: white; 
    }
    .action-btn.primary:hover { 
        background-color: var(--accent-primary-hover); 
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(239, 68, 68, 0.2);
    }
    .action-btn.secondary { 
        background-color: var(--bg-tertiary); 
        color: var(--text-primary); 
        border: 1px solid var(--border-color);
    }
    .action-btn.secondary:hover { 
        background-color: var(--border-color);
        border-color: var(--text-muted);
        transform: translateY(-2px);
    }
    
    .upload-status-area { 
        margin-bottom: 2rem; 
        background-color: var(--bg-secondary); 
        border: 1px solid var(--border-color); 
        border-radius: 12px; 
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2); 
    }
    .upload-header { 
        padding: 1rem 1.5rem; 
        border-bottom: 1px solid var(--border-color); 
    }
    .upload-header span { 
        font-weight: 500; 
    }
    .total-progress-bar-container { 
        width: 100%; 
        height: 8px; 
        background-color: var(--bg-tertiary); 
        border-radius: 4px; 
        overflow: hidden; 
        margin-top: 0.75rem; 
    }
    .total-progress-bar { 
        height: 100%; 
        background-color: var(--accent-primary); 
        border-radius: 4px; 
        transition: width 0.3s ease; 
    }
    .upload-list { 
        max-height: 200px; 
        overflow-y: auto; 
        padding: 0.5rem; 
        scrollbar-color: var(--border-color) var(--bg-secondary); 
    }
    .upload-item { 
        display: grid; 
        grid-template-columns: auto 1fr auto; 
        align-items: center; 
        gap: 1rem; 
        padding: 0.75rem 1rem; 
    }
    .upload-icon .spinner { 
        animation: spin 1.5s linear infinite; 
    }
    @keyframes spin { 
        from { transform: rotate(0deg); } 
        to { transform: rotate(360deg); } 
    }
    .upload-details { 
        overflow: hidden; 
    }
    .upload-name { 
        white-space: nowrap; 
        overflow: hidden; 
        text-overflow: ellipsis; 
        font-size: 0.9rem; 
        color: var(--text-secondary); 
    }
    .item-progress-bar-container { 
        width: 100%; 
        height: 5px; 
        background-color: var(--bg-tertiary); 
        border-radius: 2.5px; 
        overflow: hidden; 
        margin-top: 0.25rem; 
    }
    .item-progress-bar { 
        height: 100%; 
        background-color: #f87171; 
        border-radius: 2.5px; 
        transition: width 0.3s ease; 
    }
    .item-progress-bar.done { 
        background-color: #22c55e; 
    }
    .upload-error-text { 
        font-size: 0.8rem; 
        color: var(--error-color); 
    }
    .upload-progress-text { 
        font-size: 0.85rem; 
        font-weight: 500; 
        color: var(--text-secondary); 
    }

    .section-title { 
        display: flex; 
        align-items: center; 
        gap: 0.75rem; 
        font-size: 1.25rem; 
        font-weight: 600; 
        margin: 2rem 0 1rem 0; 
        color: var(--text-primary); 
    }
    
    .recents-grid { 
        display: grid; 
        grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
        gap: 1rem; 
        margin-bottom: 2rem; 
    }
    .recent-card { 
        background: var(--bg-secondary); 
        border: 1px solid var(--border-color); 
        border-radius: 12px; 
        padding: 1rem; 
        display: grid;
        grid-template-columns: auto 1fr auto;
        align-items: center; 
        gap: 1rem; 
        position: relative; 
        overflow: hidden;
    }
    .card-icon {
        flex-shrink: 0;
    }
    .card-details { 
        min-width: 0;
        display: flex;
        flex-direction: column;
    }
    .card-name { 
        font-weight: 500; 
        white-space: nowrap; 
        overflow: hidden; 
        text-overflow: ellipsis; 
        color: var(--text-primary); 
    }
    .card-meta { 
        font-size: 0.8rem; 
        color: var(--text-muted); 
    }
    .card-actions { 
        display: flex; 
        gap: 0.5rem; 
        opacity: 0;
        transition: opacity 0.2s ease-in-out;
    }
    .recent-card:hover .card-actions { 
        opacity: 1; 
    }

    .file-grid-container { 
        border: 1px solid var(--border-color); 
        border-radius: 12px; 
        overflow: hidden; 
        background-color: var(--bg-secondary); 
    }
    .grid-header { 
        display: grid; 
        grid-template-columns: auto 3fr 1fr 1.5fr auto; 
        padding: 0.75rem 1.5rem; 
        background-color: var(--bg-primary); 
        font-size: 0.8rem; 
        text-transform: uppercase; 
        color: var(--text-muted); 
    }
    .grid-row { 
        display: grid; 
        grid-template-columns: auto 3fr 1fr 1.5fr auto; 
        align-items: center; 
        padding: 1rem 1.5rem; 
        border-bottom: 1px solid var(--border-color); 
    }
    .grid-row:last-child { 
        border-bottom: none; 
    }
    .grid-row:hover { 
        background-color: var(--bg-tertiary); 
    }
    .grid-row.is-folder { 
        cursor: pointer; 
    }
    .item-name { 
        display: flex; 
        align-items: center; 
        gap: 1rem; 
        font-weight: 500; 
        color: var(--text-primary); 
    }
    .item-size, .item-modified { 
        color: var(--text-secondary); 
    }
    .row-actions { 
        opacity: 0; 
        display: flex; 
        justify-content: flex-end; 
        gap: 0.5rem; 
    }
    .grid-row:hover .row-actions { 
        opacity: 1; 
    }
    .action-icon { 
        background: none; 
        border: none; 
        cursor: pointer; 
        padding: 0.25rem; 
        color: var(--text-muted); 
        transition: color 0.2s ease;
    }
    .action-icon:hover { 
        color: var(--accent-primary); 
    }
    .action-icon.delete:hover { 
        color: #ef4444; 
    }
    .dragging { 
        opacity: 0.5; 
    }
    .drop-target { 
        outline: 2px dashed var(--accent-primary); 
    }
    .empty-state { 
        text-align: center; 
        padding: 4rem; 
        color: var(--text-muted); 
    }
    .empty-state h3 { 
        color: var(--text-primary); 
    }

    .selection-bar { 
        position: fixed; 
        bottom: 2rem; 
        left: 50%; 
        transform: translateX(-50%); 
        width: auto; 
        max-width: 600px; 
        background-color: var(--bg-secondary); 
        color: white; 
        border-radius: 12px; 
        padding: 1rem 1.5rem; 
        display: flex; 
        justify-content: space-between; 
        align-items: center; 
        z-index: 100; 
        box-shadow: 0 8px 24px rgba(0,0,0,0.4); 
        border: 1px solid var(--border-color); 
    }
    .selection-info { 
        display: flex; 
        align-items: center; 
        gap: 1rem; 
    }
    .deselect-btn { 
        background: none; 
        border: none; 
        color: var(--text-muted); 
        cursor: pointer; 
        transition: color 0.2s ease;
    }
    .deselect-btn:hover {
        color: var(--text-primary);
    }
    .selection-actions { 
        display: flex; 
        gap: 0.75rem; 
    }
    .action-btn-sm { 
        padding: 0.5rem 1rem; 
        border-radius: 8px; 
        font-weight: 500; 
        cursor: pointer; 
        display: flex; 
        gap: 0.5rem; 
        background-color: var(--bg-tertiary); 
        color: white; 
        border: 1px solid var(--border-color); 
        transition: all 0.2s ease;
    }
    .action-btn-sm:hover {
        background-color: var(--border-color);
    }
    .action-btn-sm.delete:hover { 
        background-color: var(--accent-primary); 
        border-color: var(--accent-primary); 
    }
    
    .header-select, .row-select { 
        padding-right: 1rem; 
        display: flex; 
        align-items: center; 
    }
    input[type="checkbox"] { 
        width: 16px; 
        height: 16px; 
        cursor: pointer; 
        background-color: var(--bg-tertiary); 
        border: 1px solid var(--border-color); 
        border-radius: 4px; 
        appearance: none; 
        -webkit-appearance: none; 
        transition: all 0.2s ease;
    }
    input[type="checkbox"]:checked { 
        background-color: var(--accent-primary); 
        border-color: var(--accent-primary); 
    }
    .grid-row.selected { 
        background-color: var(--nav-active-bg) !important; 
    }
    
    .file-grid-container.is-trash .grid-header, .file-grid-container.is-trash .grid-row { 
        grid-template-columns: 1fr auto; 
    }
    .file-grid-container.is-trash .row-actions { 
        opacity: 1; 
    }
    
    .modal-backdrop { 
        position: fixed; 
        top: 0; 
        left: 0; 
        width: 100%; 
        height: 100%; 
        background: rgba(17, 24, 39, 0.8); 
        display: grid; 
        place-items: center; 
        z-index: 100; 
    }
    .modal-content { 
        position: relative; 
        background: var(--bg-secondary); 
        padding: 2rem; 
        border-radius: 12px; 
        width: 90%; 
        max-width: 400px; 
        border: 1px solid var(--border-color); 
        color: var(--text-primary); 
    }
    .modal-content h3 { 
        margin: 0 0 1.5rem 0; 
        font-size: 1.25rem; 
    }
    .modal-content form { 
        display: flex; 
        flex-direction: column; 
        gap: 1rem; 
    }
    .modal-content input { 
        padding: 0.75rem; 
        border-radius: 6px; 
        border: 1px solid var(--border-color); 
        font-size: 1rem; 
        background-color: var(--bg-main); 
        color: var(--text-primary); 
    }
    .modal-content input:focus { 
        border-color: var(--accent-primary); 
        outline: none; 
    }
    .modal-content button.primary { 
        padding: 0.75rem; 
        border-radius: 6px; 
        border: none; 
        background: var(--accent-primary); 
        color: white; 
        font-weight: 500; 
        cursor: pointer; 
        transition: background-color 0.2s ease;
    }
    .modal-content button.primary:hover {
        background-color: var(--accent-primary-hover);
    }
    .close-modal-btn { 
        position: absolute; 
        top: 0.75rem; 
        right: 0.75rem; 
        background: none; 
        border: none; 
        cursor: pointer; 
        color: var(--text-muted); 
    }

    .error-banner { 
        background-color: rgba(239, 68, 68, 0.1); 
        color: #f87171; 
        border: 1px solid var(--accent-primary); 
        padding: 1rem; 
        border-radius: 8px; 
        margin-bottom: 1.5rem; 
        display: flex; 
        align-items: center; 
        gap: 0.75rem; 
    }   
</style>