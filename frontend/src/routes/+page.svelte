<script lang="ts">
	import { goto } from '$app/navigation';
	import { fly, fade } from 'svelte/transition';
    import { page } from '$app/stores';
    import { jwtToken } from '$lib/stores/auth';

	let username = '';
	let password = '';
	let message = '';
	const apiBaseUrl = 'http://localhost:8080';

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

            isLeaving = true; 
            setTimeout(() => { goto('/files'); }, 400);

		} catch (error: any) {
			message = `Login failed: ${error.message}`;
		}
	}
</script>

<svelte:head>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Poppins:wght@300;400;500;600&display=swap" rel="stylesheet">
</svelte:head>

<!-- {#key} block is used to re-trigger transitions on navigation -->
{#key $page.url.pathname}
    <div class="login-container">
        {#if !isLeaving}
            <div 
                class="login-card" 
                in:fly={{ y: 50, duration: 600, delay: 200 }} 
                out:fly={{ y: 50, duration: 300 }}
            >
                <div class="card-header">
                    <h1>Welcome Back!</h1>
                    <p>Enter your credentials to access your files.</p>
                </div>

                <form on:submit|preventDefault={handleLogin}>
                    <div class="input-group">
                        <label for="username">Username</label>
                        <input type="text" id="username" bind:value={username} required />
                    </div>
                    <div class="input-group">
                        <label for="password">Password</label>
                        <input type="password" id="password" bind:value={password} required />
                    </div>
                    <button type="submit" class="primary-btn">Log In</button>
                </form>

                <div class="card-footer">
                    {#if message}
                        <div class="message-wrapper" transition:fade>
                            <p class="message">{message}</p>
                        </div>
                    {/if}
                    <p>Don't have an account? <a href="/register" class="link-btn">Register here</a></p>
                </div>
            </div>
        {/if}
    </div>
{/key}


<style>
    :root {
        /* Keep the color variables for the card and elements */
        --card-background: rgba(31, 41, 55, 0.7);
        --accent-color: #ef4444;
        --accent-color-dark: #dc2626;
        --text-color-primary: #f9fafb;
        --text-color-secondary: #9ca3af;
        --border-color: rgba(239, 68, 68, 0.2);
        --shadow-color: rgba(0, 0, 0, 0.3);
        --input-bg-color: #111827;
        --input-border-color: #4b5563;
        --overlay-color: rgba(17, 24, 39, 0.7); /* Dark overlay with 70% opacity */
    }

    .login-container {
        width: 100vw;
        height: 100vh;
        display: flex;
        justify-content: center;
        align-items: center;
        font-family: 'Poppins', sans-serif;
        overflow: hidden;
        position: relative; /* Needed for the overlay */

        /* --- THIS IS THE GIF BACKGROUND --- */
        background-image: url('https://i.pinimg.com/originals/c2/2b/17/c22b1785a23277965498f76881cdcb85.gif');
        background-size: cover;
        background-position: center;
    }

    /* This pseudo-element creates the dark overlay */
    .login-container::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background-color: var(--overlay-color);
        z-index: 1; /* Place overlay above the background */
    }
    
    .login-card {
        max-width: 420px;
        width: 100%;
        padding: 2.5rem 3rem;
        background: var(--card-background);
        border-radius: 1.5rem;
        border: 1px solid var(--border-color);
        box-shadow: 0 8px 32px 0 var(--shadow-color);
        backdrop-filter: blur(12px);
        -webkit-backdrop-filter: blur(12px);
        display: flex;
        flex-direction: column;
        gap: 1.5rem;

        /* --- IMPORTANT: Place the card on top of the overlay --- */
        position: relative;
        z-index: 2;
    }

    .card-header { text-align: center; }
    .card-header h1 { margin: 0; font-size: 2.25rem; font-weight: 600; color: var(--text-color-primary); }
    .card-header p { margin: 0.5rem 0 0; color: var(--text-color-secondary); }

    form { display: flex; flex-direction: column; gap: 1rem; }
    
    .input-group label { 
        display: block; 
        margin-bottom: 0.5rem; 
        font-size: 0.875rem; 
        color: var(--text-color-secondary); 
        font-weight: 500; 
    }
    .input-group input { 
        width: 100%; 
        padding: 1rem; 
        border: 1px solid var(--input-border-color); 
        border-radius: 0.75rem; 
        background: var(--input-bg-color); 
        font-size: 1rem; 
        color: var(--text-color-primary); 
        box-sizing: border-box; 
        transition: all 0.3s ease; 
    }
    .input-group input:focus { 
        outline: none; 
        border-color: var(--accent-color); 
        box-shadow: 0 0 0 4px rgba(239, 68, 68, 0.15); 
    }

    .primary-btn { 
        margin-top: 0.5rem; 
        padding: 1rem; 
        border: none; 
        border-radius: 0.75rem; 
        background: var(--accent-color); 
        color: white; 
        font-size: 1rem; 
        font-weight: 500; 
        cursor: pointer; 
        transition: all 0.2s ease; 
        box-shadow: 0 4px 15px rgba(239, 68, 68, 0.2); 
    }
    .primary-btn:hover { 
        background: var(--accent-color-dark); 
        transform: translateY(-2px); 
        box-shadow: 0 6px 20px rgba(239, 68, 68, 0.3); 
    }
    .primary-btn:active { 
        transform: translateY(1px); 
        box-shadow: 0 2px 10px rgba(239, 68, 68, 0.2); 
    }

    .card-footer { text-align: center; font-size: 0.875rem; }
    .card-footer p { margin: 0; color: var(--text-color-secondary); }

    .link-btn { 
        background: none; 
        border: none; 
        color: var(--accent-color); 
        font-weight: 500; 
        cursor: pointer; 
        padding: 0; 
        font-size: inherit; 
        text-decoration: none; 
        transition: color 0.2s ease;
    }
    .link-btn:hover { text-decoration: underline; color: var(--accent-color-dark); }

    .message-wrapper { margin-bottom: 1rem; }
    .message { 
        padding: 0.75rem; 
        border-radius: 0.5rem; 
        background-color: rgba(220, 53, 69, 0.1); 
        color: #f87171;
        border: 1px solid rgba(220, 53, 69, 0.2); 
    }
</style>