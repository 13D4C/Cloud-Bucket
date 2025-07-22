<script lang="ts">
    import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import * as tus from 'tus-js-client';
	import { Folder, FileText, UploadCloud, Trash2, Home, ChevronRight, Download, X, AlertCircle, Plus, CheckCircle, Loader } from 'lucide-svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { th } from 'date-fns/locale';
    import { fetchApi } from '$lib/api';
	import { onMount } from 'svelte';
    import { fly, fade } from 'svelte/transition';

	// --- State Variables ---
	let files: any[] = [];
	let folders: any[] = [];
	let error_message = '';
	let showCreateFolderModal = false;
	let newFolderName = '';
    
    // State สำหรับ Multi-file upload
    let isBatchUploading = false;
    let uploads: { id: number, file: File, progress: number, status: 'uploading' | 'processing' | 'success' | 'error', error?: string }[] = [];
    let overallProgress = 0;
    
    // State สำหรับ Drag & Drop
    let draggedItem: any = null;

	// --- Computed State from URL ---
	$: currentPath = $page.url.searchParams.get('path') || '';
	$: breadcrumbs = [{ name: 'My Files', path: '' }].concat(
		currentPath.split('/').filter(p => p).map((part, i, arr) => ({ name: part, path: arr.slice(0, i + 1).join('/') }))
	);

	// --- Data Fetching ---
	async function fetchItems() {
        error_message = '';
		try {
			const res = await fetchApi(`/api/files?path=${encodeURIComponent(currentPath)}`);
			if (!res.ok) {
                const errData = await res.json();
                throw new Error(errData.error || 'Failed to fetch items');
            }
			const allItems = await res.json() || [];
			folders = allItems.filter((item: any) => item.isDir).sort((a: any, b: any) => a.name.localeCompare(b.name));
			files = allItems.filter((item: any) => !item.isDir).sort((a: any, b: any) => (a.originalName || '').localeCompare(b.originalName || ''));
		} catch (error: any) {
			console.error("Fetch Items Error:", error);
			error_message = `Could not load file list: ${error.message}`;
		}
	}
	onMount(fetchItems);
	$: if ($page.url) { fetchItems(); }

	// --- Core Functions ---
	async function handleCreateFolder() {
		if (!newFolderName.trim()) return;
		try {
			await fetchApi(`/api/folders?path=${encodeURIComponent(currentPath)}`, {
				method: 'POST',
				body: JSON.stringify({ folderName: newFolderName.trim() })
			});
			showCreateFolderModal = false;
			newFolderName = '';
			await fetchItems();
		} catch (error: any) { alert(`Error creating folder: ${error.message}`); }
	}
	
    function handleFileSelect(event: Event) {
        const input = event.target as HTMLInputElement;
		if (input.files && input.files.length > 0) {
			startBatchUpload(input.files);
		}
		input.value = '';
	}

    function startBatchUpload(fileList: FileList) {
        isBatchUploading = true;
        uploads = Array.from(fileList).map((file, index) => ({
            id: Date.now() + index,
            file: file,
            progress: 0,
            status: 'uploading'
        }));
        overallProgress = 0;

        const uploadPromises = uploads.map(uploadItem => 
            new Promise<void>((resolve, reject) => {
                const upload = new tus.Upload(uploadItem.file, {
                    endpoint: `http://localhost:8080/uploads/`,
                    metadata: { filename: uploadItem.file.name, filetype: uploadItem.file.type },
                    onProgress: (bytes, total) => {
                        uploadItem.progress = (bytes / total) * 100;
                        const totalUploaded = uploads.reduce((sum, u) => sum + u.progress, 0);
                        overallProgress = totalUploaded / uploads.length;
                    },
                    onError: (error) => {
                        uploadItem.status = 'error';
                        uploadItem.error = error.message;
                        reject(error);
                    },
                    onSuccess: async () => {
                        uploadItem.status = 'processing';
                        const uploadId = upload.url?.split('/').pop();
                        if (!uploadId) {
                            const err = new Error('Could not get file ID to finalize.');
                            uploadItem.status = 'error';
                            uploadItem.error = err.message;
                            reject(err);
                            return;
                        }

                        try {
                            const res = await fetchApi(`/api/finalize-upload`, {
                                method: 'POST',
                                body: JSON.stringify({ uploadId, destinationPath: currentPath })
                            });
                            if (!res.ok) {
                                const errData = await res.json();
                                throw new Error(errData.error || "Server finalize error");
                            }
                            uploadItem.status = 'success';
                            resolve();
                        } catch (finalizeError: any) {
                            uploadItem.status = 'error';
                            uploadItem.error = finalizeError.message;
                            reject(finalizeError);
                        }
                    }
                });
                upload.start();
            })
        );

        Promise.allSettled(uploadPromises).then(() => {
            fetchItems();
            setTimeout(() => {
                const stillProcessing = uploads.some(u => u.status === 'uploading' || u.status === 'processing');
                if (!stillProcessing) {
                    isBatchUploading = false;
                }
            }, 5000);
        });
    }

	async function handleDelete(item: any) {
		const itemName = item.isDir ? item.name : item.originalName;
		if (!confirm(`Are you sure you want to move "${itemName}" to the trash?`)) return;
		try {
			await fetchApi(`/api/items/${item.path}`, { method: 'DELETE' });
			await fetchItems();
		} catch (error: any) {
			alert(`Could not move item to trash: ${error.message}`);
		}
	}

    function getDownloadUrl(item: any) {
        const token = localStorage.getItem('jwt_token');
        const endpoint = item.isDir ? 'download-folder' : 'download';
        return `http://localhost:8080/api/${endpoint}/${item.path}?token=${token}`;
    }

    function formatBytes(bytes: number, decimals = 2) {
		if (!+bytes) return '0 Bytes';
		const k = 1024;
		const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${parseFloat((bytes / Math.pow(k, i)).toFixed(decimals < 0 ? 0 : decimals))} ${sizes[i]}`;
	}

    // --- Drag & Drop Functions ---
    function handleDragStart(item: any) {
        draggedItem = item;
    }

    function handleDragEnd() {
        draggedItem = null;
    }

    async function handleDrop(destinationFolder: any) {
        if (draggedItem && draggedItem.path !== destinationFolder.path) {
            await handleMove(draggedItem, destinationFolder);
        }
        handleDragEnd();
    }

    async function handleMove(sourceItem: any, destinationFolder: any) {
		try {
			const res = await fetchApi(`/api/move`, {
				method: 'POST',
				body: JSON.stringify({ 
                    sourcePath: sourceItem.path, 
                    destinationFolder: destinationFolder.path 
                })
			});
			if (!res.ok) { 
                const errorData = await res.json();
                throw new Error(errorData.error || 'Failed to move item');
            }
            await fetchItems(); 
		} catch (error: any) { 
			alert(`Error moving item: ${error.message}`);
			await fetchItems();
		}
	}
</script>

<!-- New Folder Modal -->
{#if showCreateFolderModal}
	<div class="modal-backdrop" on:click={() => showCreateFolderModal = false}>
		<div class="modal-content" transition:fly={{ y: -20, duration: 300 }} on:click|stopPropagation>
			<h3>New Folder</h3>
			<form on:submit|preventDefault={handleCreateFolder}>
				<input type="text" bind:value={newFolderName} placeholder="Enter folder name..." required />
				<button type="submit">Create Folder</button>
			</form>
			<button class="close-modal-btn" on:click={() => showCreateFolderModal = false}><X size=20 /></button>
		</div>
	</div>
{/if}

<!-- Main Content Area -->
<div class="page-header">
    <div class="breadcrumbs">
        {#each breadcrumbs as crumb, i}
            <a href={crumb.path ? `/files?path=${crumb.path}` : '/files'}>
                {#if i === 0}<Home size=16/>{:else}<span>{crumb.name}</span>{/if}
            </a>
            {#if i < breadcrumbs.length - 1}<ChevronRight size=16 class="separator" />{/if}
        {/each}
    </div>
    <div class="actions">
		<button class="action-btn secondary" on:click={() => showCreateFolderModal = true}>
			<Plus size=16 /> New Folder
		</button>
		<label class="action-btn primary">
			<UploadCloud size=16/> Upload Files
			<input type="file" hidden multiple on:change={handleFileSelect}/>
		</label>
    </div>
</div>

{#if error_message}
    <div class="error-banner"><AlertCircle size=18/> {error_message}</div>
{/if}

<div class="file-grid-container">
    <div class="grid-header">
        <div class="header-name">Name</div>
        <div class="header-size">Size</div>
        <div class="header-modified">Last Modified</div>
    </div>

    <!-- Folders -->
    {#each folders as folder (folder.path)}
        <div 
            class="grid-row is-folder" 
            class:drop-target={draggedItem && draggedItem.path !== folder.path}
            on:click={() => goto(`/files?path=${folder.path}`)}
            on:dragover|preventDefault
            on:drop|preventDefault={() => handleDrop(folder)}
            transition:fly={{ y: 20, duration: 300 }}
        >
            <div class="item-name"><Folder size=20 color="#5DADE2" /><span>{folder.name}</span></div>
            <div class="item-size">--</div>
            <div class="item-modified">{formatDistanceToNow(new Date(folder.modified),{addSuffix:true,locale:th})}</div>
            <div class="row-actions">
				<a href={getDownloadUrl(folder)} class="action-icon" title="Download Folder as .zip" on:click|stopPropagation><Download size=18 /></a>
				<button class="action-icon" title="Move to Trash" on:click|preventDefault|stopPropagation={() => handleDelete(folder)}><Trash2 size=18 /></button>
            </div>
        </div>
    {/each}

    <!-- Files -->
    {#each files as file (file.path)}
        <div 
            class="grid-row"
            class:dragging={draggedItem?.path === file.path}
            draggable="true"
            on:dragstart={() => handleDragStart(file)}
            on:dragend={handleDragEnd}
            transition:fly={{ y: 20, duration: 300 }}
        >
            <div class="item-name"><FileText size=20 color="#6C757D" /><span>{file.originalName}</span></div>
            <div class="item-size">{formatBytes(file.size)}</div>
            <div class="item-modified">{formatDistanceToNow(new Date(file.modified),{addSuffix:true,locale:th})}</div>
            <div class="row-actions">
				<a href={getDownloadUrl(file)} download={file.originalName} class="action-icon" title="Download File" on:click|stopPropagation><Download size=18 /></a>
				<button class="action-icon" title="Move to Trash" on:click|preventDefault|stopPropagation={() => handleDelete(file)}><Trash2 size=18 /></button>
            </div>
        </div>
    {/each}
</div>

{#if files.length === 0 && folders.length === 0 && !isBatchUploading && !error_message}
    <div class="empty-state" in:fade>
		<svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round" class="feather feather-folder"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>
        <h3>This folder is empty</h3>
        <p>Drop files here or use the upload button to get started.</p>
    </div>
{/if}

<!-- Multi-file Upload Status Popup -->
{#if isBatchUploading}
<div class="upload-status-popup" transition:fly={{ y: 20, duration: 300 }}>
    <div class="popup-header">
        <h4>Uploading {uploads.length} items...</h4>
        <button class="close-popup-btn" on:click={() => isBatchUploading = false}><X size=18 /></button>
    </div>
    <div class="overall-progress-container">
        <div class="overall-progress-bar" style="width: {overallProgress}%"></div>
    </div>
    <div class="upload-list">
        {#each uploads as upload (upload.id)}
        <div class="upload-item">
            <div class="item-icon">
                {#if upload.status === 'uploading' || upload.status === 'processing'}
                    <Loader size=18 class="spinner" />
                {:else if upload.status === 'success'}
                    <CheckCircle size=18 color="#198754" />
                {:else if upload.status === 'error'}
                    <AlertCircle size=18 color="#DC3545" />
                {/if}
            </div>
            <div class="item-details">
                <div class="item-filename">{upload.file.name}</div>
                {#if upload.status === 'error'}
                    <div class="item-error-msg">{upload.error}</div>
                {:else}
                    <div class="item-progress-bar-container">
                        <div class="item-progress-bar" style="width: {upload.progress}%"></div>
                    </div>
                {/if}
            </div>
        </div>
        {/each}
    </div>
</div>
{/if}

<style>
/* --- Page Layout & Header --- */
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
.breadcrumbs { display: flex; align-items: center; gap: 0.25rem; font-size: 0.9rem; }
.breadcrumbs a { display: flex; align-items: center; gap: 0.5rem; text-decoration: none; color: #6C757D; padding: 0.25rem 0.5rem; border-radius: 6px; transition: background-color 0.2s; }
.breadcrumbs a:hover { background-color: #F1F5F9; }
.separator { color: #E5E7EB; }
.actions { display: flex; gap: 0.75rem; }
.action-btn { padding: 0.6rem 1rem; border: 1px solid transparent; border-radius: 8px; font-weight: 500; cursor: pointer; display: flex; align-items: center; gap: 0.5rem; transition: all 0.2s; }
.action-btn.primary { background-color: #0d6efd; color: white; }
.action-btn.primary:hover { background-color: #0b5ed7; }
.action-btn.secondary { background-color: white; color: #343A40; border-color: #DEE2E6; }
.action-btn.secondary:hover { border-color: #ADB5BD; background-color: #F8F9FA; }

/* --- File Grid / List --- */
.file-grid-container { border: 1px solid #E5E7EB; border-radius: 12px; overflow: hidden; }
.grid-header { display: grid; grid-template-columns: 3fr 1fr 1.5fr 100px; align-items: center; padding: 0.75rem 1.5rem; background-color: #F9FAFB; font-size: 0.8rem; font-weight: 500; color: #6C757D; text-transform: uppercase; letter-spacing: 0.05em; border-bottom: 1px solid #E5E7EB; }
.header-name { grid-column: 1; }
.header-size { grid-column: 2; text-align: right; }
.header-modified { grid-column: 3; text-align: right; }
.grid-row { display: grid; grid-template-columns: 3fr 1fr 1.5fr 100px; align-items: center; padding: 1rem 1.5rem; border-bottom: 1px solid #F3F4F6; transition: background-color 0.2s; text-decoration: none; color: inherit; }
.grid-row:last-child { border-bottom: none; }
.grid-row:hover { background-color: #F9FAFB; }
.grid-row.is-folder { cursor: pointer; }
.item-name { grid-column: 1; display: flex; align-items: center; gap: 1rem; font-weight: 500; color: #1F2937; }
.item-size { grid-column: 2; text-align: right; color: #4B5563; }
.item-modified { grid-column: 3; text-align: right; color: #4B5563; }

/* --- Row Actions (Appear on Hover) --- */
.row-actions { grid-column: 4; display: flex; justify-content: flex-end; gap: 0.5rem; opacity: 0; transition: opacity 0.2s ease-in-out; }
.grid-row:hover .row-actions { opacity: 1; }
.action-icon { background: none; border: none; padding: 0.25rem; color: #6C757D; cursor: pointer; border-radius: 4px; }
.action-icon:hover { color: #0d6efd; background-color: #E9ECEF; }

/* --- Drag and Drop Styles --- */
.dragging { opacity: 0.5; background-color: #E9ECEF; }
.drop-target { outline: 2px dashed #0d6efd; outline-offset: -2px; background-color: #E7F1FF; }

/* --- Other States --- */
.status-area { margin-bottom: 1rem; background-color: #F8F9FA; padding: 1rem; border-radius: 8px; }
.progress-bar-container { width: 100%; height: 8px; background-color: #E9ECEF; border-radius: 4px; overflow: hidden; }
.progress-bar { height: 100%; background-color: #0d6efd; border-radius: 4px; transition: width 0.3s ease; }
.empty-state { text-align: center; padding: 4rem 2rem; color: #6C757D; }
.empty-state svg { color: #ADB5BD; margin-bottom: 1rem; }
.empty-state h3 { margin: 0; font-size: 1.25rem; color: #343A40; }
.empty-state p { margin-top: 0.5rem; }
.error-banner { background-color: #fef2f2; color: #991b1b; border: 1px solid #fecaca; padding: 1rem; border-radius: 8px; margin-bottom: 1rem; display: flex; align-items: center; gap: 0.75rem; }

/* --- Modal --- */
.modal-backdrop { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(9, 30, 66, 0.7); display: grid; place-items: center; z-index: 100; }
.modal-content { position: relative; background: white; padding: 2rem; border-radius: 12px; width: 90%; max-width: 400px; }
.modal-content h3 { margin: 0 0 1.5rem 0; font-size: 1.25rem; }
.modal-content form { display: flex; flex-direction: column; gap: 1rem; }
.modal-content input { padding: 0.75rem; border-radius: 6px; border: 1px solid #DEE2E6; font-size: 1rem; }
.modal-content button { padding: 0.75rem; border-radius: 6px; border: none; background: #0d6efd; color: white; font-weight: 500; cursor: pointer; }
.close-modal-btn { position: absolute; top: 0.75rem; right: 0.75rem; background: none; border: none; cursor: pointer; color: #6C757D; }

/* --- Multi-file Upload Popup --- */
.upload-status-popup { position: fixed; bottom: 1.5rem; right: 1.5rem; width: 350px; background-color: white; border-radius: 12px; box-shadow: 0 10px 25px rgba(0, 0, 0, 0.1); border: 1px solid #E5E7EB; z-index: 1000; overflow: hidden; }
.popup-header { display: flex; justify-content: space-between; align-items: center; padding: 0.75rem 1rem; border-bottom: 1px solid #E5E7EB; }
.popup-header h4 { margin: 0; font-size: 0.9rem; font-weight: 600; }
.close-popup-btn { background: none; border: none; cursor: pointer; color: #6C757D; padding: 0.25rem; }
.overall-progress-container { width: 100%; height: 4px; }
.overall-progress-bar { height: 100%; background-color: #0d6efd; transition: width 0.3s ease; }
.upload-list { max-height: 200px; overflow-y: auto; padding: 0.5rem; }
.upload-item { display: flex; align-items: center; gap: 0.75rem; padding: 0.5rem; }
.item-icon .spinner { animation: spin 1.5s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
.item-details { flex-grow: 1; min-width: 0; }
.item-filename { font-size: 0.85rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.item-error-msg { font-size: 0.75rem; color: #DC3545; }
.item-progress-bar-container { width: 100%; height: 6px; background-color: #E9ECEF; border-radius: 3px; overflow: hidden; margin-top: 0.25rem; }
.item-progress-bar { height: 100%; background-color: #6C757D; border-radius: 3px; transition: width 0.3s ease; }
</style>