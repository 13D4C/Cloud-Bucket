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

    const apiBaseUrl = 'http://localhost:8080';

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
            const response = await fetch(`${apiBaseUrl}/register`, {
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
            
            // Redirect to login page after a short delay to let user read the message
            setTimeout(() => {
                goto('/');
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
    <title>Register - MyCloud</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Poppins:wght@300;400;500;600&display=swap" rel="stylesheet">
</svelte:head>

<div class="register-container">
    <div class="register-card" in:fly={{ y: 50, duration: 600, delay: 200 }}>
        <div class="card-header">
            <h1>Create Account</h1>
            <p>Join us and start managing your files!</p>
        </div>

        <form on:submit|preventDefault={handleRegister}>
            <div class="input-group">
                <label for="username">Username</label>
                <input type="text" id="username" bind:value={username} required disabled={isLoading} />
            </div>
            <div class="input-group">
                <label for="email">Email Address</label>
                <input type="email" id="email" bind:value={email} required disabled={isLoading} />
            </div>
            <div class="input-group">
                <label for="phone">Phone Number</label>
                <input type="tel" id="phone" bind:value={phone} required disabled={isLoading} />
            </div>
            <div class="input-group">
                <label for="password">Password</label>
                <input type="password" id="password" bind:value={password} required disabled={isLoading} />
            </div>
            <button type="submit" class="primary-btn" disabled={isLoading}>
                {#if isLoading}
                    <span>Creating...</span>
                {:else}
                    <span>Create Account</span>
                {/if}
            </button>
        </form>

        <div class="card-footer">
            {#if message}
                <div class="message {messageType}">
                    {message}
                </div>
            {/if}
            <p>Already have an account? <a href="/" class="link-btn">Log in here</a></p>
        </div>
    </div>
</div><style>
    :root {
        --accent-color: #0d6efd;
        --accent-color-dark: #0b5ed7;
        --card-background: rgba(255, 255, 255, 0.85);
        --text-color-dark: #212529;
        --text-color-light: #6c757d;
        --border-color: rgba(0, 0, 0, 0.1);
        --shadow-color: rgba(31, 38, 135, 0.1);
    }

    .register-container {
        width: 100vw;
        height: 100vh;
        display: flex;
        justify-content: center;
        align-items: center;
        font-family: 'Poppins', sans-serif;
        background-image: url('https://images.unsplash.com/photo-1554147090-e1221a04a025?q=80&w=1748&auto=format&fit=crop');
        background-size: cover;
        background-position: center;
        overflow: hidden;
    }

    .register-card {
        max-width: 450px;
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
        gap: 1.25rem;
    }

    .card-header { text-align: center; margin-bottom: 0.5rem; }
    .card-header h1 { margin: 0; font-size: 2.25rem; font-weight: 600; color: var(--text-color-dark); }
    .card-header p { margin: 0.5rem 0 0; color: var(--text-color-light); }
    
    form { display: flex; flex-direction: column; gap: 1rem; }
    
    .input-group label { display: block; margin-bottom: 0.5rem; font-size: 0.875rem; font-weight: 500; color: var(--text-color-light); }
    .input-group input { width: 100%; padding: 0.9rem; border: 1px solid #ced4da; border-radius: 0.75rem; background: #fff; font-size: 1rem; box-sizing: border-box; transition: all 0.3s ease; }
    
    .input-group input:focus { 
        outline: none; 
        border-color: var(--accent-color); 
        box-shadow: 0 0 0 4px rgba(13, 110, 253, 0.15); 
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
        transition: background-color 0.3s ease; 
        display: flex; 
        justify-content: center; 
        align-items: center; 
    }
    
    .primary-btn:hover:not(:disabled) { 
        background: var(--accent-color-dark); 
    }
    .primary-btn:disabled { 
        background-color: #a0c7f8; 
        cursor: not-allowed; 
    }

    .card-footer { text-align: center; font-size: 0.875rem; margin-top: 0.5rem; }
    .card-footer p { margin: 0; color: var(--text-color-light); }
    
    .link-btn { 
        background: none; 
        border: none; 
        color: var(--accent-color); 
        font-weight: 500; 
        cursor: pointer; 
        text-decoration: none; 
    }
    .link-btn:hover { 
        text-decoration: underline; 
    }
    
    .message { padding: 0.75rem; border-radius: 0.5rem; margin-bottom: 1rem; font-weight: 500; }
    .message.error { background-color: rgba(220, 53, 69, 0.1); color: #b02a37; }
    .message.success { 
        background-color: rgba(13, 110, 253, 0.1); 
        color: #084298; 
    }
</style>