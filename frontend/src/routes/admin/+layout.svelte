  <script lang="ts">
    import { goto } from '$app/navigation';
    import { page } from '$app/stores';
    import { onMount } from 'svelte';
    import { 
      Shield, 
      Users, 
      UserX, 
      Home, 
      LogOut, 
      Settings,
      BarChart3,
      Cloud
    } from 'lucide-svelte';
    import { styles, cn } from '$lib/components/ui/styles';
    import { jwtToken } from '$lib/stores/auth';
    import { get } from 'svelte/store';

    let userRole: string = '';
    let username: string = '';

    onMount(async () => {
      // Decode JWT to get user info
      const token = get(jwtToken);
      if (token) {
        try {
          const payload = JSON.parse(atob(token.split('.')[1]));
          username = payload.sub || '';
          userRole = payload.role || 'User';
          // Redirect non-admin users
          if (userRole !== 'Admin') {
            await goto('/files');
          }
        } catch (error) {
          console.error('Failed to decode token:', error);
          await goto('/');
        }
      }
    });

    function handleLogout() {
      localStorage.removeItem('jwt_token');
      jwtToken.set(null);
      goto('/');
    }

    // Navigation items
    const navItems = [
      { 
        label: 'Dashboard', 
        href: '/admin', 
        icon: BarChart3,
        exact: true 
      },
      { 
        label: 'Accounts', 
        href: '/admin/accounts', 
        icon: Users
      },
      { 
        label: 'Disabled Users', 
        href: '/admin/disabled', 
        icon: UserX 
      },
      { 
        label: 'System Settings', 
        href: '/admin/settings', 
        icon: Settings 
      }
    ];

    $: currentPath = $page.url.pathname;
    
    $: isActive = (item: typeof navItems[0]): boolean => {
      if (item.exact) {
        return currentPath === item.href;
      }
      if (item.href === '/admin') {
        return currentPath === '/admin';
      }
      return currentPath.startsWith(item.href);
    };
  </script>

  <div class="flex h-screen bg-primary-900">
    <!-- Sidebar -->
    <aside class="w-64 bg-primary-800 border-r border-primary-700 flex flex-col">
      <!-- Logo -->
      <div class="p-6 border-b border-primary-700">
        <div class="flex items-center gap-3">
          <div class="p-2 bg-accent-500/10 rounded-lg">
            <Cloud class="w-6 h-6 text-accent-500" />
          </div>
          <div>
            <h1 class="text-lg font-bold text-primary-50">IT-Cloud Admin</h1>
            <p class="text-xs text-primary-400">Control Panel</p>
          </div>
        </div>
      </div>

      <!-- User Info -->
      <div class="px-6 py-4 border-b border-primary-700">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 bg-accent-500/20 rounded-full flex items-center justify-center">
            <Shield class="w-5 h-5 text-accent-500" />
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-primary-50 truncate">{username}</p>
            <p class="text-xs text-primary-400">Administrator</p>
          </div>
        </div>
      </div>

      <!-- Navigation -->
      <nav class="flex-1 p-4 space-y-1 overflow-y-auto">
        {#each navItems as item}
          <a 
            href={item.href}
            class={cn(
              styles.nav.item,
              isActive(item) && styles.nav.itemActive
            )}
          >
            <svelte:component this={item.icon} class={styles.nav.icon} />
            <span class="font-medium">{item.label}</span>
          </a>
        {/each}
      </nav>

      <!-- Footer Actions -->
      <div class="p-4 border-t border-primary-700 space-y-1">
        <a 
          href="/files"
          class={styles.nav.item}
        >
          <Home class={styles.nav.icon} />
          <span class="font-medium">Back to Files</span>
        </a>
        <button 
          on:click={handleLogout}
          class={cn(styles.nav.item, 'w-full text-left')}
        >
          <LogOut class={styles.nav.icon} />
          <span class="font-medium">Logout</span>
        </button>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 flex flex-col overflow-hidden">
      <!-- Header -->
      <header class="bg-primary-800 border-b border-primary-700 px-6 py-4">
        <div class="flex items-center justify-between">
          <div>
            <h2 class={styles.text.h2}>
              {#if currentPath === '/admin'}
                Dashboard
              {:else if currentPath === '/admin/accounts'}
                User Accounts
              {:else if currentPath === '/admin/disabled'}
                Disabled Users
              {:else if currentPath === '/admin/settings'}
                System Settings
              {:else}
                Admin Panel
              {/if}
            </h2>
            <p class={styles.text.muted}>
              Manage your cloud storage system
            </p>
          </div>
          <div class="text-sm text-primary-400">
            {new Date().toLocaleDateString('en-US', { 
              weekday: 'long', 
              year: 'numeric', 
              month: 'long', 
              day: 'numeric' 
            })}
          </div>
        </div>
      </header>

      <!-- Page Content -->
      <div class="flex-1 overflow-y-auto p-6">
        <slot />
      </div>
    </main>
  </div>

  <style>
    /* Custom scrollbar styles */
    :global(*) {
      scrollbar-width: thin;
      scrollbar-color: #374151 #1f2937;
    }

    :global(*::-webkit-scrollbar) {
      width: 8px;
      height: 8px;
    }

    :global(*::-webkit-scrollbar-track) {
      background: #1f2937;
    }

    :global(*::-webkit-scrollbar-thumb) {
      background-color: #374151;
      border-radius: 4px;
    }

    :global(*::-webkit-scrollbar-thumb:hover) {
      background-color: #4b5563;
    }
  </style>