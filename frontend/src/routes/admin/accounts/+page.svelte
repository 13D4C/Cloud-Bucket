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
      <h1 class={styles.text.h1}>User Accounts</h1>
      <p class={styles.text.muted}>Manage individual user accounts and permissions</p>
    </div>
    
    <button 
      on:click={() => showCreateModal = true}
      class={cn(styles.button.base, styles.button.primary, styles.button.size.md)}
    >
      <Plus class="w-5 h-5" />
      Create User
    </button>
  </div>

  <!-- Search Bar -->
  <div class={cn(styles.card.base, styles.card.padding)}>
    <div class="relative">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-text-muted" />
      <input
        type="text"
        placeholder="Search by username, email, or phone..."
        bind:value={searchQuery}
        class={cn(styles.form.input, 'pl-10')}
      />
    </div>
  </div>

  <!-- Users Grid -->
  {#if loading}
    <div class="flex items-center justify-center py-12">
      <p class="text-text-muted">Loading users...</p>
    </div>
  {:else if error}
    <div class={cn(styles.alert.base, styles.alert.error)}>
      {error}
    </div>
  {:else if filteredUsers.length === 0}
    <div class="flex flex-col items-center justify-center py-12">
      <User class="w-12 h-12 text-text-muted mb-3" />
      <p class="text-text-muted">No users found</p>
    </div>
  {:else}
    <div class={styles.grid.cols3}>
      {#each filteredUsers as user (user.id)}
        <div class={cn(styles.card.base, styles.card.padding, styles.card.hover)}>
          <!-- User Header -->
          <div class="flex items-start justify-between mb-4">
            <div class="flex items-center gap-3">
              <div class="w-12 h-12 bg-accent/20 rounded-full flex items-center justify-center">
                <span class="text-lg font-bold text-accent">
                  {user.username.charAt(0).toUpperCase()}
                </span>
              </div>
              <div>
                <h3 class="font-semibold text-text-primary">{user.username}</h3>
                <span class={cn(
                  styles.badge.base,
                  user.role === 'Admin' ? styles.badge.error : styles.badge.info
                )}>
                  {user.role}
                </span>
              </div>
            </div>
            <span class={cn(
              'text-xs px-2 py-1 rounded-full',
              user.status === 'active' ? 'bg-green-900/30 text-green-400' : 'bg-red-900/30 text-red-400'
            )}>
              {user.status}
            </span>
          </div>

          <!-- User Details -->
          <div class="space-y-3 mb-4">
            <div class="flex items-center gap-2 text-sm">
              <Mail class="w-4 h-4 text-text-muted" />
              <span class="text-text-secondary truncate">{user.email}</span>
            </div>
            <div class="flex items-center gap-2 text-sm">
              <Phone class="w-4 h-4 text-text-muted" />
              <span class="text-text-secondary">{user.phone}</span>
            </div>
            <div class="flex items-center gap-2 text-sm">
              <HardDrive class="w-4 h-4 text-text-muted" />
              <span class="text-text-secondary">
                {formatBytes(user.quotaUsed)} / {formatBytes(user.quotaLimit)}
              </span>
            </div>
          </div>

          <!-- Storage Progress Bar -->
          <div class="mb-4">
            <div class="w-full h-2 bg-bg-primary rounded-full overflow-hidden">
              <div 
                class="h-full bg-accent transition-all duration-300"
                style="width: {Math.min(100, (user.quotaUsed / user.quotaLimit) * 100)}%"
              ></div>
            </div>
          </div>

          <!-- Dates -->
          <div class="text-xs text-text-muted mb-4">
            <p>Created: {formatDate(user.createdAt)}</p>
            <p>Last login: {formatDate(user.lastLogin)}</p>
          </div>

          <!-- Actions -->
          <div class="flex gap-2">
            <button
              on:click={() => openEditModal(user)}
              class={cn(styles.button.base, styles.button.secondary, styles.button.size.sm, 'flex-1')}
            >
              <Edit2 class="w-4 h-4" />
              Edit
            </button>
            <button
              on:click={() => deleteUser(user)}
              class={cn(styles.button.base, styles.button.danger, styles.button.size.sm, 'flex-1')}
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
    class={styles.modal.backdrop}
    on:click={() => showEditModal = false}
    tabindex="0"
    role="button"
    aria-label="Close edit modal"
    on:keydown={(e) => {
      if (e.key === 'Escape' || e.key === 'Enter' || e.key === ' ') {
        showEditModal = false;
      }
    }}
  >
    <div class={cn(styles.modal.content, 'max-w-2xl')} on:pointerdown|stopPropagation>
      <div class={styles.modal.header}>
        <h2 class={styles.text.h2}>Edit User: {editingUser.username}</h2>
        <button 
          on:click={() => showEditModal = false}
          class={cn(styles.button.base, styles.button.ghost, styles.button.size.sm)}
        >
          <X class="w-5 h-5" />
        </button>
      </div>
      
      <div class={styles.modal.body}>
        <div class="space-y-4">
          <div>
            <label for="edit-email" class={styles.form.label}>Email</label>
            <input 
              id="edit-email"
              type="email" 
              bind:value={editForm.email}
              class={styles.form.input}
            />
          </div>
          
          <div>
            <label for="edit-phone" class={styles.form.label}>Phone</label>
            <input 
              id="edit-phone"
              type="tel" 
              bind:value={editForm.phone}
              class={styles.form.input}
            />
          </div>
          
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="edit-role" class={styles.form.label}>Role</label>
              <select id="edit-role" bind:value={editForm.role} class={styles.form.select}>
                <option value="User">User</option>
                <option value="Admin">Admin</option>
              </select>
            </div>
            
            <div>
              <label for="edit-status" class={styles.form.label}>Status</label>
              <select id="edit-status" bind:value={editForm.status} class={styles.form.select}>
                <option value="active">Active</option>
                <option value="disabled">Disabled</option>
                <option value="suspended">Suspended</option>
              </select>
            </div>
          </div>
          
          <div>
            <label for="edit-quota" class={styles.form.label}>Storage Quota (bytes)</label>
            <input 
              id="edit-quota"
              type="number" 
              bind:value={editForm.quotaLimit}
              class={styles.form.input}
            />
            <p class={styles.form.help}>
              Current: {formatBytes(editForm.quotaLimit)}
            </p>
          </div>
          
          <div>
            <label for="edit-password" class={styles.form.label}>New Password (leave empty to keep current)</label>
            <input 
              id="edit-password"
              type="password" 
              bind:value={editForm.password}
              placeholder="Enter new password..."
              class={styles.form.input}
            />
          </div>
        </div>
      </div>
      
      <div class={styles.modal.footer}>
        <button 
          on:click={() => showEditModal = false}
          class={cn(styles.button.base, styles.button.secondary, styles.button.size.md)}
        >
          Cancel
        </button>
        <button 
          on:click={saveUserChanges}
          class={cn(styles.button.base, styles.button.primary, styles.button.size.md)}
        >
          <Save class="w-4 h-4" />
          Save Changes
        </button>
      </div>
    </div>
  </div>
{/if}