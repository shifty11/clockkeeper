<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { isAuthenticated, initAuth, getToken, logout } from '~/lib/auth';
	import { initTheme } from '~/lib/theme';
	import ThemeSwitcher from '~/lib/components/ThemeSwitcher.svelte';
	import '~/app.css';

	let { children }: { children: Snippet } = $props();
	let authenticated = $state(false);

	isAuthenticated.subscribe((v) => (authenticated = v));

	onMount(() => {
		initAuth();
		initTheme();
		if (!getToken() && !page.url.pathname.startsWith('/login')) {
			goto('/login');
		}
	});

</script>

{#if page.url.pathname.startsWith('/login')}
	{@render children()}
{:else if authenticated}
	<div class="min-h-dvh text-primary">
		<nav class="border-b border-border bg-surface">
			<div class="mx-auto flex max-w-screen-xl items-center justify-between px-4 py-3">
				<div class="flex items-center gap-6">
					<a href="/" class="text-xl font-bold text-indigo-600 dark:text-indigo-400">Clock Keeper</a>
					<a
						href="/almanac"
						class="text-sm font-medium transition-colors {page.url.pathname.startsWith('/almanac') ? 'text-primary' : 'text-secondary hover:text-primary'}"
					>
						Almanac
					</a>
				</div>
				<div class="flex items-center gap-2">
					<ThemeSwitcher />
					<button
						onclick={logout}
						class="rounded-lg px-3 py-1.5 text-sm text-secondary transition-colors hover:bg-hover hover:text-primary"
					>
						Logout
					</button>
				</div>
			</div>
		</nav>
		<main class="mx-auto max-w-screen-xl p-4">
			{@render children()}
		</main>
	</div>
{/if}
