<!-- src/routes/admin/accounts/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchApi } from '$lib/api';
  import { 
    Search, 
    Edit2, 
    Trash2, 
    Save,
    X,
    Plus,
    Key,
    Mail,
    Phone,
    User,
    HardDrive
  } from 'lucide-svelte';
  import { styles, cn } from '$lib/components/ui/styles';

  interface User {
    id: number;
    username: string;
    email: string;
    phone: string;
    role: string;
    status: string;
    quotaLimit: number;
    quotaUsed: number;
    createdAt: string;
    lastLogin: string | null;
  }

  let users: User[] = [];
  let filteredUsers: User[] = [];
  let searchQuery = '';
  let loading = true;
  let error = '';
  
  // Edit modal state
  let showEditModal = false;
  let editingUser: User | null = null;
  let editForm = {
    email: '',
    phone: '',
    role: '',
    status: '',
    quotaLimit: 0,
    password: ''
  };

  // Create user modal state
  let showCreateModal = false;
  let createForm = {
    username: '',
    email: '',
    phone: '',
    password: '',
    confirmPassword: '',
    role: 'User',
    quotaLimit: 5368709120 // 5GB default
  };

  onMount(() => {
    fetchUsers();
  });

  async function fetchUsers() {
    try {
      loading = true;
      const response = await fetchApi('/api/admin/users');
      if (!response.ok) throw new Error('Failed to fetch users');
      users = await response.json();
      filterUsers();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load users';
    } finally {
      loading = false;
    }
  }

  function filterUsers() {
    if (searchQuery === '') {
      filteredUsers = users;
    } else {
      const query = searchQuery.toLowerCase();
      filteredUsers = users.filter(user => 
        user.username.toLowerCase().includes(query) ||
        user.email.toLowerCase().includes(query) ||
        user.phone.includes(query)
      );
    }
  }

  function openEditModal(user: User) {
    editingUser = user;
    editForm = {
      email: user.email,
      phone: user.phone,
      role: user.role,
      status: user.status,
      quotaLimit: user.quotaLimit,
      password: ''
    };
    showEditModal = true;
  }

  async function saveUserChanges() {
    if (!editingUser) return;
    
    try {
      const updates: any = {};
      if (editForm.email !== editingUser.email) updates.email = editForm.email;
      if (editForm.phone !== editingUser.phone) updates.phone = editForm.phone;
      if (editForm.role !== editingUser.role) updates.role = editForm.role;
      if (editForm.status !== editingUser.status) updates.status = editForm.status;
      if (editForm.quotaLimit !== editingUser.quotaLimit) updates.quotaLimit = editForm.quotaLimit;
      if (editForm.password) updates.password = editForm.password;

      const response = await fetchApi(`/api/admin/users/${editingUser.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates)
      });

      if (!response.ok) throw new Error('Failed to update user');
      
      showEditModal = false;
      await fetchUsers();
    } catch (err) {
      alert('Failed to update user: ' + (err instanceof Error ? err.message : 'Unknown error'));
    }
  }

  async function deleteUser(user: User) {
    if (!confirm(`Are you sure you want to permanently delete user "${user.username}"? This action cannot be undone.`)) {
      return;
    }

    try {
      const response = await fetchApi(`/api/admin/users/${user.id}`, {
        method: 'DELETE'
      });

      if (!response.ok) throw new Error('Failed to delete user');
      await fetchUsers();
    } catch (err) {
      alert('Failed to delete user: ' + (err instanceof Error ? err.message : 'Unknown error'));
    }
  }

  function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
  }

  function formatDate(dateString: string | null): string {
    if (!dateString) return 'Never';
    return new Date(dateString).toLocaleDateString('en-US', { 
      year: 'numeric', 
      month: 'short', 
      day: 'numeric' 
    });
  }

  $: searchQuery, filterUsers();
</script>

<div class="p-6 space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-3xl font-bold text-primary-100">User Accounts</h1>
      <p class="text-primary-400">Manage individual user accounts and permissions</p>
    </div>
    
    <button 
      on:click={() => showCreateModal = true}
      class="flex items-center justify-center gap-2 px-4 py-2 text-sm font-semibold bg-accent-600 text-primary-50 hover:bg-accent-700 rounded-md transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-primary-900 focus-visible:ring-accent-500"
    >
      <Plus class="w-5 h-5" />
      Create User
    </button>
  </div>

  <!-- Search Bar -->
  <div class="bg-primary-800 p-4 rounded-lg border border-primary-700">
    <div class="relative">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-primary-400" />
      <input
        type="text"
        placeholder="Search by username, email, or phone..."
        bind:value={searchQuery}
        class="w-full md:w-96 bg-primary-900 border border-primary-700 rounded-md pl-10 pr-3 py-2 text-primary-100 placeholder-primary-500 focus:border-accent-500 focus:ring-0"
      />
    </div>
  </div>

  <!-- Users Grid -->
  {#if loading}
    <div class="flex items-center justify-center py-12">
      <p class="text-primary-400">Loading users...</p>
    </div>
  {:else if error}
    <div class="p-4 rounded-md bg-accent-900/50 text-accent-300 border border-accent-800">
      {error}
    </div>
  {:else if filteredUsers.length === 0}
    <div class="flex flex-col items-center justify-center py-12 text-center">
      <User class="w-12 h-12 text-primary-600 mb-3" />
      <h3 class="text-lg font-semibold text-primary-300">No Users Found</h3>
      <p class="text-primary-400">The search did not return any results.</p>
    </div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
      {#each filteredUsers as user (user.id)}
        <div class="bg-primary-800 p-5 rounded-lg border border-primary-700 hover:border-primary-600 transition-colors flex flex-col">
          <!-- User Header -->
          <div class="flex items-start justify-between mb-4">
            <div class="flex items-center gap-4">
              <div class="w-12 h-12 bg-accent-500/10 rounded-full flex items-center justify-center">
                <span class="text-xl font-bold text-accent-400">
                  {user.username.charAt(0).toUpperCase()}
                </span>
              </div>
              <div>
                <h3 class="font-semibold text-primary-100">{user.username}</h3>
                <span class= {cn(
                  'px-2.5 py-0.5 text-xs font-semibold rounded-full',
                  user.role === 'Admin' ? 'bg-accent-500/20 text-accent-300' : 'bg-primary-700/80 text-primary-200'
                )}>
                  {user.role}
                </span>
              </div>
            </div>
            <span class={cn(
              'text-xs px-2 py-1 rounded-full font-medium',
              user.status === 'active' ? 'bg-green-500/10 text-green-400' : 'bg-primary-700 text-primary-300'
            )}>
              {user.status}
            </span>
          </div>

          <!-- User Details -->
          <div class="space-y-3 mb-4 text-sm">
            <div class="flex items-center gap-3">
              <Mail class="w-4 h-4 text-primary-500" />
              <span class="text-primary-300 truncate">{user.email}</span>
            </div>
            <div class="flex items-center gap-3">
              <Phone class="w-4 h-4 text-primary-500" />
              <span class="text-primary-300">{user.phone || 'Not provided'}</span>
            </div>
            <div class="flex items-center gap-3">
              <HardDrive class="w-4 h-4 text-primary-500" />
              <span class="text-primary-300">
                {formatBytes(user.quotaUsed)} / <span class="text-primary-400">{formatBytes(user.quotaLimit)}</span>
              </span>
            </div>
          </div>

          <!-- Storage Progress Bar -->
          <div class="mb-4">
            <div class="w-full h-1.5 bg-primary-700 rounded-full overflow-hidden">
              <div 
                class="h-full bg-accent-500 transition-all duration-300"
                style="width: {Math.min(100, (user.quotaUsed / user.quotaLimit) * 100)}%"
              ></div>
            </div>
          </div>

          <!-- Dates -->
          <div class="text-xs text-primary-500 mb-5">
            <p>Created: {formatDate(user.createdAt)}</p>
            <p>Last login: {formatDate(user.lastLogin)}</p>
          </div>

          <!-- Actions -->
          <div class="flex gap-2">
            <button
              on:click={() => openEditModal(user)}
              class={cn(styles.button.base, styles.button.secondary, styles.button.size.sm, 'flex-1 cursor-pointer')}
            >
              <Edit2 class="w-4 h-4" />
              Edit
            </button>
            <button
              on:click={() => deleteUser(user)}
              class={cn(styles.button.base, styles.button.danger, styles.button.size.sm, 'flex-1 cursor-pointer')}
            >
              <Trash2 class="w-4 h-4" />
              Delete
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Edit User Modal -->
{#if showEditModal && editingUser}
  <div 
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-primary-900/80 backdrop-blur-sm"
    on:click={() => showEditModal = false}
    tabindex="0"
    role="button"
    aria-label="Close edit modal"
    on:keydown={(e) => {
      if (e.key === 'Escape') {
        showEditModal = false;
      }
    }}
  >
<div 
  class="bg-primary-800 rounded-lg shadow-xl w-full max-w-2xl border border-primary-700" 
  on:click|stopPropagation
  on:keydown={(e) => {
    if (e.key === 'Escape') {
      showEditModal = false;
    }
  }}
  role="dialog"
  aria-modal="true"
  aria-labelledby="edit-modal-title"
  tabindex="-1"
>
  <div class="flex items-center justify-between p-4 border-b border-primary-700">
    <h2 id="edit-modal-title" class="text-xl font-semibold text-primary-100">Edit User: {editingUser.username}</h2>
    <button 
      on:click={() => showEditModal = false}
      class="p-1 text-primary-400 hover:text-primary-100 hover:bg-primary-700 rounded-full transition-colors"
      aria-label="Close dialog"
    >
      <X class="w-5 h-5" />
    </button>
  </div>
  
    <div class="p-6">
      <div class="space-y-4">
        <!-- Form fields -->
        <div>
          <label for="edit-email" class="block text-sm font-medium text-primary-300 mb-1">Email</label>
          <input id="edit-email" type="email" bind:value={editForm.email} class="w-full bg-primary-900 border border-primary-700 rounded-md px-3 py-2 text-primary-100 placeholder-primary-500 focus:border-accent-500 focus:ring-accent-500" />
        </div>
        
        <div>
          <label for="edit-phone" class="block text-sm font-medium text-primary-300 mb-1">Phone</label>
          <input id="edit-phone" type="tel" bind:value={editForm.phone} class="w-full bg-primary-900 border border-primary-700 rounded-md px-3 py-2 text-primary-100 placeholder-primary-500 focus:border-accent-500 focus:ring-accent-500" />
        </div>
        
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="edit-role" class="block text-sm font-medium text-primary-300 mb-1">Role</label>
            <select id="edit-role" bind:value={editForm.role} class="w-full bg-primary-900 border border-primary-700 rounded-md px-3 py-2 text-primary-100 focus:border-accent-500 focus:ring-accent-500">
              <option value="User">User</option>
              <option value="Admin">Admin</option>
            </select>
          </div>
          
          <div>
            <label for="edit-status" class="block text-sm font-medium text-primary-300 mb-1">Status</label>
            <select id="edit-status" bind:value={editForm.status} class="w-full bg-primary-900 border border-primary-700 rounded-md px-3 py-2 text-primary-100 focus:border-accent-500 focus:ring-accent-500">
              <option value="active">Active</option>
              <option value="disabled">Disabled</option>
            </select>
          </div>
        </div>
        
        <div>
          <label for="edit-quota" class="block text-sm font-medium text-primary-300 mb-1">Storage Quota (bytes)</label>
          <input id="edit-quota" type="number" bind:value={editForm.quotaLimit} class="w-full bg-primary-900 border border-primary-700 rounded-md px-3 py-2 text-primary-100 placeholder-primary-500 focus:border-accent-500 focus:ring-accent-500" />
          <p class="text-xs text-primary-400 mt-1">
            Current: {formatBytes(editForm.quotaLimit)}
          </p>
        </div>
        
        <div>
          <label for="edit-password" class="block text-sm font-medium text-primary-300 mb-1">New Password (leave empty to keep current)</label>
          <input id="edit-password" type="password" bind:value={editForm.password} placeholder="••••••••" class="w-full bg-primary-900 border border-primary-700 rounded-md px-3 py-2 text-primary-100 placeholder-primary-500 focus:border-accent-500 focus:ring-accent-500" />
        </div>
      </div>
    </div>
    
      <div class="flex items-center justify-end gap-3 p-4 bg-primary-900/50 border-t border-primary-700 rounded-b-lg">
        <button on:click={() => showEditModal = false} class="px-4 py-2 cursor-pointer text-sm font-semibold bg-primary-700 text-primary-100 hover:bg-primary-600 rounded-md transition-colors">
          Cancel
        </button>
        <button on:click={saveUserChanges} class="flex cursor-pointer items-center justify-center gap-2 px-4 py-2 text-sm font-semibold bg-accent-600 text-primary-50 hover:bg-accent-700 rounded-md transition-colors">
          <Save class="w-4 h-4" />
          Save Changes
        </button>
      </div>
    </div>
  </div>
{/if}