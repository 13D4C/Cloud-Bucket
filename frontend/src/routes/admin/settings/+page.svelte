<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi } from '$lib/api';
	import { 
		Settings, 
		Shield, 
		HardDrive, 
		Users, 
		Server,
		Key,
		Clock,
		AlertTriangle,
		Save,
		RefreshCw,
		Check,
		X,
		FileText,
		Database,
		Mail,
		Globe
	} from 'lucide-svelte';

	interface SystemSettings {
		siteName: string;
		siteDescription: string;
		maintenanceMode: boolean;
		allowRegistration: boolean;
		maxFileSize: number;
		allowedFileTypes: string[];
		requireEmailVerification: boolean;
		supportEmail: string;
	}

	interface StorageSettings {
		defaultUserQuota: number;
		maxUserQuota: number;
		autoCleanupEnabled: boolean;
		cleanupDays: number;
		storageWarningThreshold: number;
		compressionEnabled: boolean;
	}

	interface SecuritySettings {
		sessionTimeout: number;
		passwordMinLength: number;
		requireStrongPassword: boolean;
		maxLoginAttempts: number;
		lockoutDuration: number;
		twoFactorEnabled: boolean;
		autoBackupEnabled: boolean;
		backupRetentionDays: number;
	}

	let activeTab: 'system' | 'storage' | 'security' = 'system';
	let loading = true;
	let saving = false;
	let error = '';
	let successMessage = '';

	let systemSettings: SystemSettings = {
		siteName: 'IT Cloud Storage',
		siteDescription: 'Secure file storage and sharing platform',
		maintenanceMode: false,
		allowRegistration: true,
		maxFileSize: 100, // MB
		allowedFileTypes: ['pdf', 'doc', 'docx', 'txt', 'jpg', 'png', 'gif', 'zip'],
		requireEmailVerification: false,
		supportEmail: 'admin@itcloud.com'
	};

	let storageSettings: StorageSettings = {
		defaultUserQuota: 5000, // MB
		maxUserQuota: 50000, // MB
		autoCleanupEnabled: true,
		cleanupDays: 30,
		storageWarningThreshold: 80, // percentage
		compressionEnabled: true
	};

	let securitySettings: SecuritySettings = {
		sessionTimeout: 24, // hours
		passwordMinLength: 8,
		requireStrongPassword: true,
		maxLoginAttempts: 5,
		lockoutDuration: 15, // minutes
		twoFactorEnabled: false,
		autoBackupEnabled: true,
		backupRetentionDays: 30
	};

	onMount(async () => {
		await loadSettings();
	});

	async function loadSettings() {
		try {
			loading = true;
			error = '';
			
			const response = await fetchApi('/api/admin/settings');
			if (!response.ok) throw new Error('Failed to load settings');
			const data = await response.json();
			
			systemSettings = data.system || systemSettings;
			storageSettings = data.storage || storageSettings;
			securitySettings = data.security || securitySettings;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load settings';
		} finally {
			loading = false;
		}
	}

	async function saveSettings() {
		try {
			saving = true;
			error = '';
			successMessage = '';

			const settings = {
				system: systemSettings,
				storage: storageSettings,
				security: securitySettings
			};

			const response = await fetchApi('/api/admin/settings', {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(settings)
			});
			
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || 'Failed to save settings');
			}
			
			successMessage = 'Settings saved successfully!';
			setTimeout(() => successMessage = '', 3000);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save settings';
		} finally {
			saving = false;
		}
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
	}

	function addFileType() {
		const newType = prompt('Enter file extension (without dot):');
		if (newType && !systemSettings.allowedFileTypes.includes(newType.toLowerCase())) {
			systemSettings.allowedFileTypes = [...systemSettings.allowedFileTypes, newType.toLowerCase()];
		}
	}

	function removeFileType(type: string) {
		systemSettings.allowedFileTypes = systemSettings.allowedFileTypes.filter(t => t !== type);
	}
</script>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<Settings class="w-8 h-8 text-accent-400" />
			<div>
				<h1 class="text-3xl font-bold text-primary-50">System Settings</h1>
				<p class="text-sm text-primary-400">Configure system-wide settings and preferences</p>
			</div>
		</div>
		
		<div class="flex items-center gap-3">
			<button
				on:click={loadSettings}
				class="inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 hover:bg-primary-700 text-primary-300 hover:text-primary-50 px-3 py-1.5 text-sm gap-1.5"
				disabled={loading || saving}
			>
				<RefreshCw class="w-4 h-4" />
				Refresh
			</button>
			
			<button
				on:click={saveSettings}
				class="inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 bg-accent-500 hover:bg-accent-600 text-white focus:ring-accent-500 px-4 py-2 text-base gap-2"
				disabled={saving}
			>
				{#if saving}
					<RefreshCw class="w-4 h-4 animate-spin" />
					Saving...
				{:else}
					<Save class="w-4 h-4" />
					Save Changes
				{/if}
			</button>
		</div>
	</div>

	<!-- Success/Error Messages -->
	{#if successMessage}
		<div class="flex items-center gap-3 p-4 bg-green-500/10 text-green-300 border border-green-500 rounded-lg">
			<Check class="w-4 h-4" />
			{successMessage}
		</div>
	{/if}

	{#if error}
		<div class="flex items-center gap-3 p-4 bg-red-500/10 text-red-300 border border-red-500 rounded-lg">
			<AlertTriangle class="w-4 h-4" />
			{error}
		</div>
	{/if}

	<!-- Tabs -->
	<div class="border-b border-primary-700">
		<nav class="flex space-x-4">
			<button
				class="py-2 px-1 border-b-2 font-medium cursor-pointer text-sm transition-colors {activeTab === 'system' ? 'border-accent-500 text-accent-400' : 'border-transparent text-primary-400 hover:text-primary-300'}"
				on:click={() => activeTab = 'system'}
			>
				<div class="flex items-center gap-2">
					<Globe class="w-4 h-4" />
					System
				</div>
			</button>
			<button
				class="py-2 px-1 border-b-2 font-medium cursor-pointer  text-sm transition-colors {activeTab === 'storage' ? 'border-accent-500 text-accent-400' : 'border-transparent text-primary-400 hover:text-primary-300'}"
				on:click={() => activeTab = 'storage'}
			>
				<div class="flex items-center gap-2">
					<HardDrive class="w-4 h-4" />
					Storage
				</div>
			</button>
			<button
				class="py-2 px-1 border-b-2 font-medium cursor-pointer  text-sm transition-colors {activeTab === 'security' ? 'border-accent-500 text-accent-400' : 'border-transparent text-primary-400 hover:text-primary-300'}"
				on:click={() => activeTab = 'security'}
			>
				<div class="flex items-center gap-2">
					<Shield class="w-4 h-4" />
					Security
				</div>
			</button>
		</nav>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="text-primary-400">Loading settings...</div>
		</div>
	{/if}

	<!-- System Settings Tab -->
	{#if !loading && activeTab === 'system'}
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<!-- General Settings -->
			<div class="bg-primary-800 border border-primary-700 rounded-xl p-6">
				<h3 class="text-xl font-semibold text-primary-50 mb-4">General Settings</h3>
				
				<div class="space-y-4">
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Site Name</label>
						<input
							type="text"
							bind:value={systemSettings.siteName}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
							placeholder="Enter site name"
						/>
					</div>
					
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Site Description</label>
						<textarea
							bind:value={systemSettings.siteDescription}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none h-20 resize-none"
							placeholder="Enter site description"
						></textarea>
					</div>
					
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Support Email</label>
						<input
							type="email"
							bind:value={systemSettings.supportEmail}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
							placeholder="support@example.com"
						/>
					</div>
				</div>
			</div>

			<!-- System Controls -->
			<div class="bg-primary-800 border border-primary-700 rounded-xl p-6">
				<h3 class="text-xl font-semibold text-primary-50 mb-4">System Controls</h3>
				
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<div>
							<label class="block text-sm font-medium text-primary-300 mb-1">Maintenance Mode</label>
							<p class="text-sm text-primary-400">Temporarily disable site access</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input
								type="checkbox"
								bind:checked={systemSettings.maintenanceMode}
								class="sr-only peer"
							/>
							<div class="w-11 h-6 bg-primary-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-500"></div>
						</label>
					</div>
					
					<div class="flex items-center justify-between">
						<div>
							<label class="block text-sm font-medium text-primary-300 mb-1">Allow Registration</label>
							<p class="text-sm text-primary-400">Allow new users to register</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input
								type="checkbox"
								bind:checked={systemSettings.allowRegistration}
								class="sr-only peer"
							/>
							<div class="w-11 h-6 bg-primary-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-500"></div>
						</label>
					</div>
					
					<div class="flex items-center justify-between">
						<div>
							<label class="block text-sm font-medium text-primary-300 mb-1">Email Verification</label>
							<p class="text-sm text-primary-400">Require email verification for new accounts</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input
								type="checkbox"
								bind:checked={systemSettings.requireEmailVerification}
								class="sr-only peer"
							/>
							<div class="w-11 h-6 bg-primary-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-500"></div>
						</label>
					</div>
				</div>
			</div>

			<!-- File Upload Settings -->
			<div class="bg-primary-800 border border-primary-700 rounded-xl p-6 lg:col-span-2">
				<h3 class="text-xl font-semibold text-primary-50 mb-4">File Upload Settings</h3>
				
				<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Maximum File Size (MB)</label>
						<input
							type="number"
							bind:value={systemSettings.maxFileSize}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
							min="1"
							max="1000"
						/>
					</div>
					
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Allowed File Types</label>
						<div class="flex flex-wrap gap-2 mb-2">
							{#each systemSettings.allowedFileTypes as type}
								<span class="inline-flex items-center gap-1 px-2 py-1 bg-primary-700 text-primary-200 text-xs rounded">
									.{type}
									<button
										on:click={() => removeFileType(type)}
										class="text-red-400 hover:text-red-300"
									>
										<X class="w-3 h-3" />
									</button>
								</span>
							{/each}
						</div>
						<button
							on:click={addFileType}
							class="inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 bg-primary-700 hover:bg-primary-600 text-primary-50 border border-primary-600 focus:ring-primary-600 px-3 py-1.5 text-sm gap-1.5"
						>
							Add Type
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- Storage Settings Tab -->
	{#if !loading && activeTab === 'storage'}
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<!-- Quota Settings -->
			<div class="bg-primary-800 border border-primary-700 rounded-xl p-6">
				<h3 class="text-xl font-semibold text-primary-50 mb-4">User Quota Settings</h3>
				
				<div class="space-y-4">
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Default User Quota (MB)</label>
						<input
							type="number"
							bind:value={storageSettings.defaultUserQuota}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
							min="100"
							step="100"
						/>
						<p class="text-sm text-primary-400 mt-1">Default storage quota for new users</p>
					</div>
					
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Maximum User Quota (MB)</label>
						<input
							type="number"
							bind:value={storageSettings.maxUserQuota}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
							min="1000"
							step="1000"
						/>
						<p class="text-sm text-primary-400 mt-1">Maximum quota that can be assigned to a user</p>
					</div>
					
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Storage Warning Threshold (%)</label>
						<input
							type="number"
							bind:value={storageSettings.storageWarningThreshold}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
							min="50"
							max="95"
						/>
						<p class="text-sm text-primary-400 mt-1">Warn users when they reach this percentage of their quota</p>
					</div>
				</div>
			</div>

			<!-- Cleanup Settings -->
			<div class="bg-primary-800 border border-primary-700 rounded-xl p-6">
				<h3 class="text-xl font-semibold text-primary-50 mb-4">Cleanup & Optimization</h3>
				
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<div>
							<label class="block text-sm font-medium text-primary-300 mb-1">Auto Cleanup</label>
							<p class="text-sm text-primary-400">Automatically clean up deleted files</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input
								type="checkbox"
								bind:checked={storageSettings.autoCleanupEnabled}
								class="sr-only peer"
							/>
							<div class="w-11 h-6 bg-primary-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-500"></div>
						</label>
					</div>
					
					{#if storageSettings.autoCleanupEnabled}
						<div>
							<label class="block text-sm font-medium text-primary-300 mb-2">Cleanup After (Days)</label>
							<input
								type="number"
								bind:value={storageSettings.cleanupDays}
								class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
								min="1"
								max="365"
							/>
							<p class="text-sm text-primary-400 mt-1">Delete files from trash after this many days</p>
						</div>
					{/if}
					
					<div class="flex items-center justify-between">
						<div>
							<label class="block text-sm font-medium text-primary-300 mb-1">File Compression</label>
							<p class="text-sm text-primary-400">Enable automatic file compression</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input
								type="checkbox"
								bind:checked={storageSettings.compressionEnabled}
								class="sr-only peer"
							/>
							<div class="w-11 h-6 bg-primary-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-500"></div>
						</label>
					</div>
				</div>
			</div>

			<!-- Storage Statistics -->
			<div class="bg-primary-800 border border-primary-700 rounded-xl p-6 lg:col-span-2">
				<h3 class="text-xl font-semibold text-primary-50 mb-4">Storage Overview</h3>
				
				<div class="grid grid-cols-1 md:grid-cols-4 gap-4">
					<div class="text-center p-4 bg-primary-900/30 rounded-lg">
						<Database class="w-8 h-8 text-blue-400 mx-auto mb-2" />
						<p class="text-2xl font-bold text-primary-100">
							{formatBytes(storageSettings.defaultUserQuota * 1024 * 1024)}
						</p>
						<p class="text-sm text-primary-400">Default Quota</p>
					</div>
					
					<div class="text-center p-4 bg-primary-900/30 rounded-lg">
						<HardDrive class="w-8 h-8 text-green-400 mx-auto mb-2" />
						<p class="text-2xl font-bold text-primary-100">
							{formatBytes(storageSettings.maxUserQuota * 1024 * 1024)}
						</p>
						<p class="text-sm text-primary-400">Maximum Quota</p>
					</div>
					
					<div class="text-center p-4 bg-primary-900/30 rounded-lg">
						<Clock class="w-8 h-8 text-yellow-400 mx-auto mb-2" />
						<p class="text-2xl font-bold text-primary-100">{storageSettings.cleanupDays}</p>
						<p class="text-sm text-primary-400">Cleanup Days</p>
					</div>
					
					<div class="text-center p-4 bg-primary-900/30 rounded-lg">
						<AlertTriangle class="w-8 h-8 text-red-400 mx-auto mb-2" />
						<p class="text-2xl font-bold text-primary-100">{storageSettings.storageWarningThreshold}%</p>
						<p class="text-sm text-primary-400">Warning Threshold</p>
					</div>
				</div>
			</div>
		</div>
	{/if}

	<!-- Security Settings Tab -->
	{#if !loading && activeTab === 'security'}
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<!-- Authentication Settings -->
			<div class="bg-primary-800 border border-primary-700 rounded-xl p-6">
				<h3 class="text-xl font-semibold text-primary-50 mb-4">Authentication</h3>
				
				<div class="space-y-4">
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Session Timeout (Hours)</label>
						<input
							type="number"
							bind:value={securitySettings.sessionTimeout}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
							min="1"
							max="168"
						/>
						<p class="text-sm text-primary-400 mt-1">Automatically log out users after this time</p>
					</div>
					
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Max Login Attempts</label>
						<input
							type="number"
							bind:value={securitySettings.maxLoginAttempts}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
							min="3"
							max="10"
						/>
					</div>
					
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Lockout Duration (Minutes)</label>
						<input
							type="number"
							bind:value={securitySettings.lockoutDuration}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
							min="5"
							max="60"
						/>
					</div>
					
					<div class="flex items-center justify-between">
						<div>
							<label class="block text-sm font-medium text-primary-300 mb-1">Two-Factor Authentication</label>
							<p class="text-sm text-primary-400">Enable 2FA for enhanced security</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input
								type="checkbox"
								bind:checked={securitySettings.twoFactorEnabled}
								class="sr-only peer"
							/>
							<div class="w-11 h-6 bg-primary-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-500"></div>
						</label>
					</div>
				</div>
			</div>

			<!-- Password Policy -->
			<div class="bg-primary-800 border border-primary-700 rounded-xl p-6">
				<h3 class="text-xl font-semibold text-primary-50 mb-4">Password Policy</h3>
				
				<div class="space-y-4">
					<div>
						<label class="block text-sm font-medium text-primary-300 mb-2">Minimum Password Length</label>
						<input
							type="number"
							bind:value={securitySettings.passwordMinLength}
							class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
							min="6"
							max="32"
						/>
					</div>
					
					<div class="flex items-center justify-between">
						<div>
							<label class="block text-sm font-medium text-primary-300 mb-1">Require Strong Password</label>
							<p class="text-sm text-primary-400">Must contain uppercase, lowercase, numbers, and symbols</p>
						</div>
						<label class="relative inline-flex items-center cursor-pointer">
							<input
								type="checkbox"
								bind:checked={securitySettings.requireStrongPassword}
								class="sr-only peer"
							/>
							<div class="w-11 h-6 bg-primary-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-500"></div>
						</label>
					</div>
				</div>
			</div>

			<!-- Backup Settings -->
			<div class="bg-primary-800 border border-primary-700 rounded-xl p-6 lg:col-span-2">
				<h3 class="text-xl font-semibold text-primary-50 mb-4">Backup & Recovery</h3>
				
				<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
					<div class="space-y-4">
						<div class="flex items-center justify-between">
							<div>
								<label class="block text-sm font-medium text-primary-300 mb-1">Auto Backup</label>
								<p class="text-sm text-primary-400">Automatically backup system data</p>
							</div>
							<label class="relative inline-flex items-center cursor-pointer">
								<input
									type="checkbox"
									bind:checked={securitySettings.autoBackupEnabled}
									class="sr-only peer"
								/>
								<div class="w-11 h-6 bg-primary-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent-500"></div>
							</label>
						</div>
						
						{#if securitySettings.autoBackupEnabled}
							<div>
								<label class="block text-sm font-medium text-primary-300 mb-2">Backup Retention (Days)</label>
								<input
									type="number"
									bind:value={securitySettings.backupRetentionDays}
									class="w-full px-3 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:border-accent-500 focus:outline-none"
									min="7"
									max="365"
								/>
							</div>
						{/if}
					</div>
					
					<div class="space-y-3">
						<button class="inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 bg-primary-600 hover:bg-primary-700 text-primary-50 border border-primary-500 hover:border-primary-400 w-full px-4 py-2 gap-2">
							<Database class="w-4 h-4" />
							Create Manual Backup
						</button>
						
						<button class="inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 hover:bg-primary-700 text-primary-300 hover:text-primary-50 w-full px-4 py-2 gap-2">
							<FileText class="w-4 h-4" />
							View Backup History
						</button>
						
						<button class="inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 hover:bg-primary-700 text-primary-300 hover:text-primary-50 w-full px-4 py-2 gap-2">
							<RefreshCw class="w-4 h-4" />
							Restore from Backup
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>