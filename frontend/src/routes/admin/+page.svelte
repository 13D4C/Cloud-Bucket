<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchApi } from '$lib/api';
  import { 
    Users, 
    HardDrive, 
    Activity, 
    Shield,
    Search,
    Edit2,
    Trash2,
    MoreVertical,
    ChevronDown,
    UserPlus,
    Download,
    RefreshCw
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

  interface SystemStats {
    totalUsers: number;
    activeUsers: number;
    totalStorage: number;
    usedStorage: number;
  }

  let users: User[] = [];
  let filteredUsers: User[] = [];
  let stats: SystemStats = {
    totalUsers: 0,
    activeUsers: 0,
    totalStorage: 0,
    usedStorage: 0
  };
  
  let loading = true;
  let error = '';
  let searchQuery = '';
  let roleFilter = 'all';
  let statusFilter = 'all';
  let selectedUser: User | null = null;
  let showEditModal = false;
  let showDeleteConfirm = false;
  let actionMenuOpen: number | null = null;

  // Fetch data on mount
  onMount(async () => {
    await Promise.all([fetchUsers(), fetchStats()]);
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

  async function fetchStats() {
    try {
      const response = await fetchApi('/api/admin/stats');
      if (!response.ok) throw new Error('Failed to fetch stats');
      stats = await response.json();
    } catch (err) {
      console.error('Failed to load stats:', err);
    }
  }

  function filterUsers() {
    filteredUsers = users.filter(user => {
      const matchesSearch = searchQuery === '' || 
        user.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
        user.email.toLowerCase().includes(searchQuery.toLowerCase());
      
      const matchesRole = roleFilter === 'all' || user.role === roleFilter;
      const matchesStatus = statusFilter === 'all' || user.status === statusFilter;
      
      return matchesSearch && matchesRole && matchesStatus;
    });
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
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', { 
      year: 'numeric', 
      month: 'short', 
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function getQuotaPercentage(used: number, limit: number): number {
    if (limit === 0) return 0;
    return Math.min(100, (used / limit) * 100);
  }

  function getStatusBadgeClass(status: string): string {
    switch (status) {
      case 'active': return styles.badge.success;
      case 'disabled': return styles.badge.error;
      case 'suspended': return styles.badge.warning;
      default: return styles.badge.default;
    }
  }

  function getRoleBadgeClass(role: string): string {
    switch (role) {
      case 'Admin': return styles.badge.base +  styles.badge.error;
      case 'User': return styles.badge.base + styles.badge.info;
      default: return styles.badge.default;
    }
  }

  async function handleEditUser(user: User) {
    selectedUser = user;
    showEditModal = true;
    actionMenuOpen = null;
  }

  async function handleDeleteUser(user: User) {
    selectedUser = user;
    showDeleteConfirm = true;
    actionMenuOpen = null;
  }

  async function handleToggleStatus(user: User) {
    try {
      const newStatus = user.status === 'active' ? 'disabled' : 'active';
      const response = await fetchApi(`/api/admin/users/${user.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: newStatus })
      });
      
      if (!response.ok) throw new Error('Failed to update user status');
      await fetchUsers();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to update user';
    }
    actionMenuOpen = null;
  }

  async function exportUsers() {
    const csv = [
      ['Username', 'Email', 'Role', 'Status', 'Quota Used', 'Quota Limit', 'Created', 'Last Login'],
      ...filteredUsers.map(user => [
        user.username,
        user.email,
        user.role,
        user.status,
        formatBytes(user.quotaUsed),
        formatBytes(user.quotaLimit),
        formatDate(user.createdAt),
        formatDate(user.lastLogin)
      ])
    ].map(row => row.join(',')).join('\n');

    const blob = new Blob([csv], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `users-export-${new Date().toISOString().split('T')[0]}.csv`;
    a.click();
    window.URL.revokeObjectURL(url);
  }

  // Reactive statements
  $: searchQuery, roleFilter, statusFilter, filterUsers();
</script>

<div class="space-y-6">
  <!-- Stats Cards -->
  <div class={styles.grid.cols4}>
    <div class={cn(styles.card.base, styles.card.padding)}>
      <div class="flex items-center justify-between">
        <div>
          <p class={styles.text.small}>Total Users</p>
          <p class="text-2xl font-bold text-text-primary mt-1">{stats.totalUsers}</p>
        </div>
        <div class="p-3 bg-blue-500/10 rounded-lg">
          <Users class="w-6 h-6 text-blue-400" />
        </div>
      </div>
    </div>

    <div class={cn(styles.card.base, styles.card.padding)}>
      <div class="flex items-center justify-between">
        <div>
          <p class={styles.text.small}>Active Users</p>
          <p class="text-2xl font-bold text-text-primary mt-1">{stats.activeUsers}</p>
        </div>
        <div class="p-3 bg-green-500/10 rounded-lg">
          <Activity class="w-6 h-6 text-green-400" />
        </div>
      </div>
    </div>

    <div class={cn(styles.card.base, styles.card.padding)}>
      <div class="flex items-center justify-between">
        <div>
          <p class={styles.text.small}>Storage Used</p>
          <p class="text-2xl font-bold text-text-primary mt-1">
            {formatBytes(stats.usedStorage)}
          </p>
        </div>
        <div class="p-3 bg-purple-500/10 rounded-lg">
          <HardDrive class="w-6 h-6 text-purple-400" />
        </div>
      </div>
    </div>

    <div class={cn(styles.card.base, styles.card.padding)}>
      <div class="flex items-center justify-between">
        <div>
          <p class={styles.text.small}>Admin Users</p>
          <p class="text-2xl font-bold text-text-primary mt-1">
            {users.filter(u => u.role === 'Admin').length}
          </p>
        </div>
        <div class="p-3 bg-red-500/10 rounded-lg">
          <Shield class="w-6 h-6 text-red-400" />
        </div>
      </div>
    </div>
  </div>

  <!-- Users Table Section -->
  <div class={cn(styles.card.base, 'overflow-hidden')}>
    <!-- Table Header with Filters -->
    <div class="px-6 py-4 border-b border-border">
      <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <h3 class={styles.text.h3}>User Accounts</h3>
        
        <div class="flex items-center gap-3">
          <!-- Search -->
          <div class="relative">
            <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
            <input
              type="text"
              placeholder="Search users..."
              bind:value={searchQuery}
              class={cn(styles.form.input, 'pl-10 w-64')}
            />
          </div>

          <!-- Role Filter -->
          <select 
            bind:value={roleFilter}
            class={cn(styles.form.select, 'w-32')}
          >
            <option value="all">All Roles</option>
            <option value="Admin">Admin</option>
            <option value="User">User</option>
          </select>

          <!-- Status Filter -->
          <select 
            bind:value={statusFilter}
            class={cn(styles.form.select, 'w-32')}
          >
            <option value="all">All Status</option>
            <option value="active">Active</option>
            <option value="disabled">Disabled</option>
          </select>

          <!-- Actions -->
          <button
            on:click={fetchUsers}
            class={cn(styles.button.base, styles.button.ghost, styles.button.size.sm)}
            title="Refresh"
          >
            <RefreshCw class="w-4 h-4" />
          </button>

          <button
            on:click={exportUsers}
            class={cn(styles.button.base, styles.button.secondary, styles.button.size.sm)}
          >
            <Download class="w-4 h-4" />
            Export
          </button>

          <button
            class={cn(styles.button.base, styles.button.primary, styles.button.size.sm)}
          >
            <UserPlus class="w-4 h-4" />
            Add User
          </button>
        </div>
      </div>
    </div>

    <!-- Table -->
    <div class={styles.table.wrapper}>
      {#if loading}
        <div class="flex items-center justify-center py-12">
          <div class="text-text-muted">Loading users...</div>
        </div>
      {:else if error}
        <div class={cn(styles.alert.base, styles.alert.error, 'm-6')}>
          {error}
        </div>
      {:else if filteredUsers.length === 0}
        <div class="flex flex-col items-center justify-center py-12">
          <Users class="w-12 h-12 text-text-muted mb-3" />
          <p class="text-text-muted">No users found</p>
        </div>
      {:else}
        <table class={styles.table.base}>
          <thead class={styles.table.header}>
            <tr>
              <th class={styles.table.headerCell}>Username</th>
              <th class={styles.table.headerCell}>Email</th>
              <th class={styles.table.headerCell}>Role</th>
              <th class={styles.table.headerCell}>Storage Usage</th>
              <th class={styles.table.headerCell}>Actions</th>
            </tr>
          </thead>
          <tbody class={styles.table.body}>
            {#each filteredUsers as user (user.id)}
              <tr class={styles.table.row}>
                <td class={styles.table.cellPrimary}>
                  <div class="flex items-center gap-3">
                    <div class="w-8 h-8 bg-accent/20 rounded-full flex items-center justify-center">
                      <span class="text-xs font-medium text-accent">
                        {user.username.charAt(0).toUpperCase()}
                      </span>
                    </div>
                    <span>{user.username}</span>
                  </div>
                </td>
                <td class={styles.table.cell}>{user.email}</td>
                <td class={styles.table.cell}>
                  <span class={getRoleBadgeClass(user.role)}>
                    {user.role}
                  </span>
                </td>
                <td class={styles.table.cell}>
                  <div class="space-y-1">
                    <div class="flex items-center justify-between text-xs">
                      <span>{formatBytes(user.quotaUsed)}</span>
                      <span class="text-text-muted">/ {formatBytes(user.quotaLimit)}</span>
                    </div>
                    <div class="w-32 h-1.5 bg-bg-tertiary rounded-full overflow-hidden">
                      <div 
                        class="h-full bg-accent transition-all duration-300"
                        style="width: {getQuotaPercentage(user.quotaUsed, user.quotaLimit)}%"
                      ></div>
                    </div>
                  </div>
                </td>
                <td class={styles.table.cell}>
                  <div class="relative">
                    <button
                      on:click={() => actionMenuOpen = actionMenuOpen === user.id ? null : user.id}
                      class={cn(styles.button.base, styles.button.ghost, styles.button.size.sm)}
                    >
                      <MoreVertical class="w-4 h-4" />
                    </button>
                    
                    {#if actionMenuOpen === user.id}
                      <div class="absolute right-0 mt-2 w-48 bg-bg-secondary border border-border rounded-lg shadow-lg z-10">
                        <button
                          on:click={() => handleEditUser(user)}
                          class="w-full px-4 py-2 text-left text-sm text-text-secondary hover:bg-bg-tertiary hover:text-text-primary transition-colors"
                        >
                          <Edit2 class="inline w-4 h-4 mr-2" />
                          Edit User
                        </button>
                        <button
                          on:click={() => handleDeleteUser(user)}
                          class="w-full px-4 py-2 text-left text-sm text-error hover:bg-bg-tertiary transition-colors"
                        >
                          <Trash2 class="inline w-4 h-4 mr-2" />
                          Delete User
                        </button>
                      </div>
                    {/if}
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  </div>
</div>

<!-- Click outside to close action menu -->
{#if actionMenuOpen !== null}
  <div 
    class="fixed inset-0 z-0" 
    on:click={() => actionMenuOpen = null}
    role="button"
    tabindex="0"
    aria-label="Close actions menu"
    on:keydown={(e) => {
      if (e.key === 'Escape' || e.key === 'Enter' || e.key === ' ') {
        actionMenuOpen = null;
      }
    }}
  ></div>
{/if}