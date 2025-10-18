<script lang="ts">
	import { goto } from '$app/navigation';
	import { fly, fade } from 'svelte/transition';
    import { page } from '$app/stores';
    import { jwtToken } from '$lib/stores/auth';

	let username = '';
	let password = '';
	let message = '';
    const apiBaseUrl = '';
	// const apiBaseUrl = 'http://localhost:8080';

    let isLeaving = false;

	async function handleRegister() {
		message = 'Registering...';
		try {
			const response = await fetch(`${apiBaseUrl}/register`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) });
			const data = await response.json();
			if (!response.ok) throw new Error(data.error || 'Something went wrong');
			message = data.message + ' Please log in.';
		} catch (error: any) { message = `Registration failed: ${error.message}`; }
	}

	async function handleLogin() {
		message = '';
		try {
			const response = await fetch(`${apiBaseUrl}/login`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username, password }) });
			const data = await response.json();
			if (!response.ok) throw new Error(data.error || 'Something went wrong');
			
			jwtToken.set(data.token);

            const payload = JSON.parse(atob(data.token.split('.')[1]));
            const userRole = payload.role || 'User';

            isLeaving = true; 
            if (userRole === 'Admin') {
                setTimeout(() => { goto('/admin'); }, 400);
            } else {
                setTimeout(() => { goto('/files'); }, 400);
            }

		} catch (error: any) {
			message = `Login failed: ${error.message}`;
		}
	}
</script>

<svelte:head>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous">
    <link href="https://fonts.googleapis.com/css2?family=Poppins:wght@300;400;500;600&display=swap" rel="stylesheet">
</svelte:head>

<!-- {#key} block is used to re-trigger transitions on navigation -->
{#key $page.url.pathname}
    <div class="w-screen h-screen flex justify-center items-center font-sans overflow-hidden relative bg-cover bg-center"
         style="background-image: url('https://i.pinimg.com/originals/c2/2b/17/c22b1785a23277965498f76881cdcb85.gif');">
        
        <!-- Dark overlay -->
        <div class="absolute inset-0 bg-primary-900/70 z-10"></div>
        
        {#if !isLeaving}
            <div 
                class="max-w-md w-full p-10 bg-primary-800/70 rounded-3xl border border-accent-500/20 shadow-2xl backdrop-blur-xl flex flex-col gap-6 relative z-20" 
                in:fly={{ y: 50, duration: 600, delay: 200 }} 
                out:fly={{ y: 50, duration: 300 }}
            >
                <div class="text-center">
                    <h1 class="text-4xl font-semibold text-primary-50 m-0">Welcome Back!</h1>
                    <p class="text-primary-400 mt-2 mb-0">Enter your credentials to access your files.</p>
                </div>

                <form on:submit|preventDefault={handleLogin} class="flex flex-col gap-4">
                    <div class="flex flex-col">
                        <label for="username" class="block mb-2 text-sm text-primary-300 font-medium">Username</label>
                        <input 
                            type="text" 
                            id="username" 
                            bind:value={username} 
                            required 
                            class="w-full p-4 border border-primary-600 rounded-xl bg-primary-900 text-primary-50 text-base box-border transition-all duration-300 focus:outline-none focus:border-accent-500 focus:shadow-lg focus:shadow-accent-500/15"
                        />
                    </div>
                    <div class="flex flex-col">
                        <label for="password" class="block mb-2 text-sm text-primary-300 font-medium">Password</label>
                        <input 
                            type="password" 
                            id="password" 
                            bind:value={password} 
                            required 
                            class="w-full p-4 border border-primary-600 rounded-xl bg-primary-900 text-primary-50 text-base box-border transition-all duration-300 focus:outline-none focus:border-accent-500 focus:shadow-lg focus:shadow-accent-500/15"
                        />
                    </div>
                    <button 
                        type="submit" 
                        class="mt-2 p-4 border-none rounded-xl bg-accent-500 text-white text-base font-medium cursor-pointer transition-all duration-200 shadow-lg shadow-accent-500/20 hover:bg-accent-600 hover:-translate-y-0.5 hover:shadow-xl hover:shadow-accent-500/30 active:translate-y-0.5 active:shadow-md active:shadow-accent-500/20"
                    >
                        Log In
                    </button>
                </form>

                <div class="text-center text-sm">
                    {#if message}
                        <div class="mb-4" transition:fade>
                            <p class="p-3 rounded-lg bg-red-900/10 text-red-400 border border-red-800/20">
                                {message}
                            </p>
                        </div>
                    {/if}
                    <p class="text-primary-400 m-0">
                        Don't have an account? 
                        <a 
                            href="/register" 
                            class="bg-none border-none text-accent-500 font-medium cursor-pointer p-0  no-underline transition-colors duration-200 hover:underline hover:text-accent-600"
                        >
                            Register here
                        </a>
                    </p>
                </div>
            </div>
        {/if}
    </div>
{/key}

