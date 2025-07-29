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
</div>

<style>
    :root {
        --card-background: rgba(31, 41, 55, 0.7);
        --accent-color: #ef4444;       
        --accent-color-dark: #dc2626;  
        --text-color-primary: #f9fafb;   
        --text-color-secondary: #9ca3af; 
        --border-color: rgba(239, 68, 68, 0.2); 
        --shadow-color: rgba(0, 0, 0, 0.3);
        --input-bg-color: #111827;
        --input-border-color: #4b5563;
        --overlay-color: rgba(17, 24, 39, 0.7);
        --success-bg: rgba(34, 197, 94, 0.1);
        --success-text: #4ade80;
        --error-bg: rgba(239, 68, 68, 0.1);
        --error-text: #f87171;
    }

    .register-container {
        width: 100vw;
        height: 100vh;
        display: flex;
        justify-content: center;
        align-items: center;
        font-family: 'Poppins', sans-serif;
        overflow: hidden;
        position: relative;
        background-image: url('https://i.pinimg.com/originals/c2/2b/17/c22b1785a23277965498f76881cdcb85.gif');
        background-size: cover;
        background-position: center;
    }

    .register-container::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background-color: var(--overlay-color);
        z-index: 1;
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
        position: relative;
        z-index: 2;
    }

    .card-header { text-align: center; margin-bottom: 0.5rem; }
    .card-header h1 { margin: 0; font-size: 2.25rem; font-weight: 600; color: var(--text-color-primary); }
    .card-header p { margin: 0.5rem 0 0; color: var(--text-color-secondary); }
    
    form { display: flex; flex-direction: column; gap: 1rem; }
    
    .input-group label { display: block; margin-bottom: 0.5rem; font-size: 0.875rem; font-weight: 500; color: var(--text-color-secondary); }
    .input-group input { 
        width: 100%; 
        padding: 0.9rem; 
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
        display: flex; 
        justify-content: center; 
        align-items: center;
        box-shadow: 0 4px 15px rgba(239, 68, 68, 0.2);
    }
    .primary-btn:hover:not(:disabled) { 
        background: var(--accent-color-dark); 
        transform: translateY(-2px);
    }
    .primary-btn:disabled { 
        background-color: #4b5563; /* Darker disabled color */
        cursor: not-allowed; 
    }

    .card-footer { text-align: center; font-size: 0.875rem; margin-top: 0.5rem; }
    .card-footer p { margin: 0; color: var(--text-color-secondary); }
    
    .link-btn { 
        background: none; 
        border: none; 
        color: var(--accent-color); 
        font-weight: 500; 
        cursor: pointer; 
        text-decoration: none; 
        transition: color 0.2s ease;
    }
    .link-btn:hover { 
        text-decoration: underline; 
        color: var(--accent-color-dark);
    }
    
    .message { padding: 0.75rem; border-radius: 0.5rem; margin-bottom: 1rem; font-weight: 500; border: 1px solid transparent; }
    .message.error { 
        background-color: var(--error-bg); 
        color: var(--error-text);
        border-color: rgba(239, 68, 68, 0.3);
    }
    .message.success { 
        background-color: var(--success-bg); 
        color: var(--success-text);
        border-color: rgba(34, 197, 94, 0.3);
    }
</style>