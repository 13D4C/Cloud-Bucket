<script lang="ts">
    import { goto } from '$app/navigation';
    import { fly } from 'svelte/transition';

    let username = '';
    let password = '';
    let email = '';
    let phone = '';
    let message = '';
    let messageType: 'error' | 'success' = 'error';
    let isLoading = false;

    // แก้ไข URL นี้ถ้า API ของคุณไม่ได้อยู่ที่ localhost:8080
    // const apiBaseUrl = 'http://localhost:8080';
    const apiBaseUrl = '';

    async function handleRegister() {
        message = '';
        isLoading = true;

        if (!username || !password || !email || !phone) {
            messageType = 'error';
            message = 'Please fill out all fields.';
            isLoading = false;
            return;
        }

        try {
            const response = await fetch(`${apiBaseUrl}/auth/register`, { // Endpoint ของคุณอาจจะเป็น /register หรือ /auth/register
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password, email, phone })
            });
            const data = await response.json();
            if (!response.ok) {
                throw new Error(data.error || 'Registration failed');
            }
            messageType = 'success';
            message = data.message;
            
            setTimeout(() => {
                goto('/'); // Redirect to login page
            }, 2000);

        } catch (error: any) {
            messageType = 'error';
            message = error.message;
        } finally {
            isLoading = false;
        }
    }
</script>

<svelte:head>
    <title>Register - IT-Cloud</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous">
    <link href="https://fonts.googleapis.com/css2?family=Poppins:wght@300;400;500;600&display=swap" rel="stylesheet">
</svelte:head>

<div class="w-screen h-screen flex justify-center items-center font-sans overflow-hidden relative bg-cover bg-center"
     style="background-image: url('https://i.pinimg.com/originals/c2/2b/17/c22b1785a23277965498f76881cdcb85.gif');">
    
    <!-- Dark overlay -->
    <div class="absolute inset-0 bg-primary-900/70 z-10"></div>
    
    <div class="max-w-sm w-full p-10 bg-primary-800/70 rounded-3xl border border-accent-500/20 shadow-2xl backdrop-blur-xl flex flex-col gap-5 relative z-20" 
         in:fly={{ y: 50, duration: 600, delay: 200 }}>
        
        <div class="text-center mb-2">
            <h1 class="text-4xl font-semibold text-primary-50 m-0">Create Account</h1>
            <p class="text-primary-400 mt-2 mb-0">Join us and start managing your files!</p>
        </div>

        <form on:submit|preventDefault={handleRegister} class="flex flex-col gap-4">
            <div class="flex flex-col">
                <label for="username" class="block mb-2 text-sm font-medium text-primary-300">Username</label>
                <input 
                    type="text" 
                    id="username" 
                    bind:value={username} 
                    required 
                    disabled={isLoading} 
                    class="w-full p-3.5 border border-primary-600 rounded-xl bg-primary-900 text-primary-50 text-base box-border transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:border-accent-500 focus:shadow-lg focus:shadow-accent-500/15"
                />
            </div>
            <div class="flex flex-col">
                <label for="email" class="block mb-2 text-sm font-medium text-primary-300">Email Address</label>
                <input 
                    type="email" 
                    id="email" 
                    bind:value={email} 
                    required 
                    disabled={isLoading} 
                    class="w-full p-3.5 border border-primary-600 rounded-xl bg-primary-900 text-primary-50 text-base box-border transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:border-accent-500 focus:shadow-lg focus:shadow-accent-500/15"
                />
            </div>
            <div class="flex flex-col">
                <label for="phone" class="block mb-2 text-sm font-medium text-primary-300">Phone Number</label>
                <input 
                    type="tel" 
                    id="phone" 
                    bind:value={phone} 
                    required 
                    disabled={isLoading} 
                    class="w-full p-3.5 border border-primary-600 rounded-xl bg-primary-900 text-primary-50 text-base box-border transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:border-accent-500 focus:shadow-lg focus:shadow-accent-500/15"
                />
            </div>
            <div class="flex flex-col">
                <label for="password" class="block mb-2 text-sm font-medium text-primary-300">Password</label>
                <input 
                    type="password" 
                    id="password" 
                    bind:value={password} 
                    required 
                    disabled={isLoading} 
                    class="w-full p-3.5 border border-primary-600 rounded-xl bg-primary-900 text-primary-50 text-base box-border transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:border-accent-500 focus:shadow-lg focus:shadow-accent-500/15"
                />
            </div>
            <button 
                type="submit" 
                disabled={isLoading}
                class="mt-2 p-4 border-none rounded-xl bg-accent-500 text-white text-base font-medium cursor-pointer transition-all duration-200 flex justify-center items-center shadow-lg shadow-accent-500/20 disabled:bg-primary-600 disabled:cursor-not-allowed hover:bg-accent-600 hover:-translate-y-0.5"
            >
                {#if isLoading}
                    <span>Creating...</span>
                {:else}
                    <span>Create Account</span>
                {/if}
            </button>
        </form>

        <div class="text-center text-sm mt-2">
            {#if message}
                <div class="p-3 rounded-lg mb-4 font-medium border border-transparent {messageType === 'error' ? 'bg-red-900/10 text-red-400 border-red-800/30' : 'bg-green-900/10 text-green-400 border-green-800/30'}">
                    {message}
                </div>
            {/if}
            <p class="text-primary-400 m-0">
                Already have an account? 
                <a 
                    href="/" 
                    class="bg-none border-none text-accent-500 font-medium cursor-pointer no-underline transition-colors duration-200 hover:underline hover:text-accent-600"
                >
                    Log in here
                </a>
            </p>
        </div>
    </div>
</div>
