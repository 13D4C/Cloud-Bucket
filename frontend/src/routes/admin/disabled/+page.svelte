<!-- src/routes/admin/disabled/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { fetchApi } from '$lib/api';
  import { 
    Search, 
    RefreshCw,
    UserCheck,
    Trash2,
    AlertTriangle,
    Calendar,
    HardDrive,
    Mail,
    Phone,
    User,
    Clock
  } from 'lucide-svelte';
  import { styles, cn } from '$lib/components/ui/styles';

  interface DisabledUser {
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

  let disabledUsers: DisabledUser[] = [];
  let filteredUsers: DisabledUser[] = [];
  let searchQuery = '';
  let loading = true;
  let error = '';
  let refreshing = false;

  onMount(() => {
    fetchDisabledUsers();
  });

  async function fetchDisabledUsers() {
    try {
      loading = true;
      const response = await fetchApi('/api/admin/users');
      if (!response.ok) throw new Error('Failed to fetch users');
      
      const allUsers = await response.json();
      // Filter for disabled users only
      disabledUsers = allUsers.filter((user: DisabledUser) => user.status === 'Disabled');
      filterUsers();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load disabled users';
    } finally {
      loading = false;
    }
  }

  async function refreshUsers() {
    refreshing = true;
    await fetchDisabledUsers();
    refreshing = false;
  }

  function filterUsers() {
    if (searchQuery === '') {
      filteredUsers = disabledUsers;
    } else {
      const query = searchQuery.toLowerCase();
      filteredUsers = disabledUsers.filter(user => 
        user.username.toLowerCase().includes(query) ||
        user.email.toLowerCase().includes(query) ||
        user.phone.includes(query)
      );
    }
  }

  async function enableUser(user: DisabledUser) {
    if (!confirm(`Are you sure you want to enable user "${user.username}"? They will regain access to their account.`)) {
      return;
    }

    try {
      const response = await fetchApi(`/api/admin/users/${user.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: 'Active' })
      });

      if (!response.ok) throw new Error('Failed to enable user');
      
      await fetchDisabledUsers();
    } catch (err) {
      alert('Failed to enable user: ' + (err instanceof Error ? err.message : 'Unknown error'));
    }
  }

  async function deleteUser(user: DisabledUser) {
    if (!confirm(`Are you sure you want to permanently delete user "${user.username}"? This action cannot be undone and will delete all their files.`)) {
      return;
    }

    try {
      const response = await fetchApi(`/api/admin/users/${user.id}`, {
        method: 'DELETE'
      });

      if (!response.ok) throw new Error('Failed to delete user');
      await fetchDisabledUsers();
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
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function getUsagePercentage(used: number, limit: number): number {
    return limit > 0 ? (used / limit) * 100 : 0;
  }

  $: searchQuery, filterUsers();
</script>

<svelte:head>
  <title>Disabled Users - Admin Panel</title>
</svelte:head>

<div class="p-6 space-y-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-3xl font-bold text-primary-100 flex items-center gap-3">
        <AlertTriangle class="w-8 h-8 text-yellow-500" />
        Disabled Users
      </h1>
      <p class="text-primary-400">Manage users who have been disabled from accessing the system</p>
    </div>
    
    <button 
      on:click={refreshUsers}
      disabled={refreshing}
      class="flex items-center justify-center gap-2 px-4 py-2 text-sm font-semibold bg-primary-700 text-primary-100 hover:bg-primary-600 rounded-md transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-primary-900 focus-visible:ring-accent-500 disabled:opacity-50"
    >
      <RefreshCw class="w-4 h-4" />
      Refresh
    </button>
  </div>

  <!-- Search Bar -->
  <div class="bg-primary-800 p-4 rounded-lg border border-primary-700">
    <div class="flex items-center justify-between gap-4">
      <div class="relative flex-1">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-primary-400" />
        <input
          type="text"
          placeholder="Search disabled users by username, email, or phone..."
          bind:value={searchQuery}
          class="w-full bg-primary-900 border border-primary-700 rounded-md pl-10 pr-3 py-2 text-primary-100 placeholder-primary-500 focus:border-accent-500 focus:ring-0"
        />
      </div>
      
      <div class="text-sm text-primary-400">
        {filteredUsers.length} of {disabledUsers.length} disabled users
      </div>
    </div>
  </div>

  <!-- Content Area -->
  {#if loading}
    <div class="flex items-center justify-center py-12">
      <div class="flex items-center gap-3 text-primary-400">
        <RefreshCw class="w-5 h-5 animate-spin" />
        Loading disabled users...
      </div>
    </div>
  {:else if error}
    <div class="bg-red-900/20 border border-red-700 rounded-lg p-4">
      <div class="flex items-center gap-2 text-red-400">
        <AlertTriangle class="w-5 h-5" />
        <span class="font-medium">Error loading disabled users</span>
      </div>
      <p class="text-red-300 mt-1">{error}</p>
      <button 
        on:click={fetchDisabledUsers}
        class="mt-3 px-3 py-1 bg-red-800 hover:bg-red-700 rounded text-sm text-red-100 transition-colors"
      >
        Try Again
      </button>
    </div>
  {:else if filteredUsers.length === 0}
    <div class="text-center py-12">
      {#if disabledUsers.length === 0}
        <div class="flex flex-col items-center gap-4">
          <div class="p-4 bg-green-900/20 rounded-full">
            <UserCheck class="w-12 h-12 text-green-400" />
          </div>
          <div>
            <h3 class="text-xl font-semibold text-primary-100 mb-2">No Disabled Users</h3>
            <p class="text-primary-400">All users are currently active and have access to the system.</p>
          </div>
        </div>
      {:else}
        <div class="flex flex-col items-center gap-4">
          <div class="p-4 bg-primary-700 rounded-full">
            <Search class="w-12 h-12 text-primary-400" />
          </div>
          <div>
            <h3 class="text-xl font-semibold text-primary-100 mb-2">No Results Found</h3>
            <p class="text-primary-400">No disabled users match your search criteria.</p>
            <button 
              on:click={() => searchQuery = ''} 
              class="mt-2 text-accent-400 hover:text-accent-300 text-sm underline"
            >
              Clear search
            </button>
          </div>
        </div>
      {/if}
    </div>
  {:else}
    <!-- Users Table -->
    <div class="bg-primary-800 rounded-lg border border-primary-700 overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-primary-700">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-primary-300 uppercase tracking-wider">
                User
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-primary-300 uppercase tracking-wider">
                Contact
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-primary-300 uppercase tracking-wider">
                Storage Usage
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-primary-300 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-primary-700">
            {#each filteredUsers as user (user.id)}
              <tr class="hover:bg-primary-750 transition-colors">
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="flex items-center gap-3">
                    <div class="flex-shrink-0">
                      <div class="w-10 h-10 bg-red-900/30 rounded-full flex items-center justify-center">
                        <User class="w-5 h-5 text-red-400" />
                      </div>
                    </div>
                    <div>
                      <div class="text-sm font-medium text-primary-100">{user.username}</div>
                      <div class="text-xs text-red-400 flex items-center gap-1">
                        <AlertTriangle class="w-3 h-3" />
                        Disabled Account
                      </div>
                    </div>
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="space-y-1">
                    <div class="text-sm text-primary-200 flex items-center gap-1">
                      <Mail class="w-3 h-3 text-primary-400" />
                      {user.email}
                    </div>
                    <div class="text-sm text-primary-300 flex items-center gap-1">
                      <Phone class="w-3 h-3 text-primary-400" />
                      {user.phone}
                    </div>
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="space-y-2">
                    <div class="flex items-center gap-2">
                      <HardDrive class="w-4 h-4 text-primary-400" />
                      <span class="text-sm text-primary-200">
                        {formatBytes(user.quotaUsed)} / {formatBytes(user.quotaLimit)}
                      </span>
                    </div>
                    <div class="w-24 bg-primary-700 rounded-full h-2">
                      <div 
                        class="bg-accent-500 h-2 rounded-full transition-all duration-300"
                        style="width: {Math.min(getUsagePercentage(user.quotaUsed, user.quotaLimit), 100)}%"
                      ></div>
                    </div>
                    <div class="text-xs text-primary-400">
                      {getUsagePercentage(user.quotaUsed, user.quotaLimit).toFixed(1)}% used
                    </div>
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <div class="flex items-center gap-2">
                    <button
                    on:click={() => enableUser(user)}
                    class="cursor-pointer flex items-center gap-1 px-3 py-1 text-sm font-medium bg-secondary-300 text-primary-50 rounded"
                    >
                    <UserCheck class="w-3 h-3" />
                    Enable
                    </button>

                    <button
                    on:click={() => deleteUser(user)}
                    class="cursor-pointer flex items-center gap-1 px-3 py-1 text-sm font-medium bg-accent-600 text-primary-100 rounded"
                    >
                    <Trash2 class="w-3 h-3" />
                    Delete
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    <!-- Summary Stats -->
    {#if filteredUsers.length > 0}
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="bg-primary-800 p-4 rounded-lg border border-primary-700">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-red-900/30 rounded-lg">
              <AlertTriangle class="w-6 h-6 text-red-400" />
            </div>
            <div>
              <div class="text-2xl font-bold text-primary-100">{disabledUsers.length}</div>
              <div class="text-sm text-primary-400">Disabled Users</div>
            </div>
          </div>
        </div>
        
        <div class="bg-primary-800 p-4 rounded-lg border border-primary-700">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-accent-900/30 rounded-lg">
              <HardDrive class="w-6 h-6 text-accent-400" />
            </div>
            <div>
              <div class="text-2xl font-bold text-primary-100">
                {formatBytes(disabledUsers.reduce((acc, user) => acc + user.quotaUsed, 0))}
              </div>
              <div class="text-sm text-primary-400">Storage Used</div>
            </div>
          </div>
        </div>
      </div>
    {/if}
  {/if}
</div>