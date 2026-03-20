<script lang="ts">
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { client } from '~/lib/api';
	import { getErrorMessage } from '~/lib/errors';
	import type { Script, Character } from '~/lib/gen/clockkeeper/v1/clockkeeper_pb';
	import { Team } from '~/lib/gen/clockkeeper/v1/clockkeeper_pb';
	import TeamSection from '~/lib/components/TeamSection.svelte';
	import CharacterPickerModal from '~/lib/components/CharacterPickerModal.svelte';

	let script = $state<Script | undefined>();
	let name = $state('');
	let loading = $state(true);
	let error = $state('');
	let showAddCharacter = $state(false);
	let allCharacters = $state.raw<Character[]>([]);
	let lastSavedName = '';
	let lastSavedIds: string[] = [];

	const teamOrder = [Team.TOWNSFOLK, Team.OUTSIDER, Team.MINION, Team.DEMON, Team.TRAVELLER, Team.FABLED, Team.LORIC];

	const charactersByTeam = $derived.by(() => {
		if (!script?.characters) return {};
		const grouped: Record<number, Character[]> = {};
		for (const char of script.characters) {
			if (!grouped[char.team]) grouped[char.team] = [];
			grouped[char.team].push(char);
		}
		return grouped;
	});

	const scriptIdSet = $derived(new Set(script?.characterIds ?? []));

	// Auto-save with debounce.
	let saveTimer: ReturnType<typeof setTimeout> | undefined;
	const characterIds = $derived(script?.characterIds ?? []);

	$effect(() => {
		const _name = name;
		const _ids = characterIds;

		if (script?.isSystem) return;
		if (_name === lastSavedName && JSON.stringify(_ids) === JSON.stringify(lastSavedIds)) return;

		clearTimeout(saveTimer);
		saveTimer = setTimeout(() => {
			untrack(() => autoSave(_name, _ids));
		}, 800);

		return () => clearTimeout(saveTimer);
	});

	async function autoSave(currentName: string, currentIds: string[]) {
		if (!script) return;
		try {
			await client.updateScript({
				id: script.id,
				name: currentName,
				characterIds: currentIds
			});
			lastSavedName = currentName;
			lastSavedIds = currentIds;
		} catch (err) {
			error = getErrorMessage(err, 'Failed to save');
		}
	}

	async function loadScript(id: bigint) {
		loading = true;
		error = '';
		try {
			const resp = await client.getScript({ id });
			script = resp.script;
			name = script?.name ?? '';
			lastSavedName = name;
			lastSavedIds = script?.characterIds ?? [];
		} catch (err) {
			error = getErrorMessage(err, 'Failed to load script');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		const id = page.params.id;
		untrack(() => loadScript(BigInt(id)));
	});

	async function openAddCharacter() {
		showAddCharacter = true;
		if (allCharacters.length === 0) {
			try {
				const resp = await client.listCharacters({});
				allCharacters = resp.characters;
			} catch (err) {
				error = getErrorMessage(err, 'Failed to load characters');
			}
		}
	}

	function removeCharacter(charId: string) {
		if (!script) return;
		script = {
			...script,
			characterIds: script.characterIds.filter((id) => id !== charId),
			characters: script.characters.filter((c) => c.id !== charId)
		} as Script;
	}

	function addCharacter(char: Character) {
		if (!script) return;
		script = {
			...script,
			characterIds: [...script.characterIds, char.id],
			characters: [...script.characters, char]
		} as Script;
	}

	async function createFromEdition(editionId: string) {
		try {
			const resp = await client.createScriptFromEdition({ editionId, name: '' });
			if (resp.script) {
				goto(`/scripts/${resp.script.id}`);
			}
		} catch (err) {
			error = getErrorMessage(err, 'Failed to duplicate script');
		}
	}

	async function deleteScript() {
		if (!script) return;
		try {
			await client.deleteScript({ id: script.id });
			goto('/scripts');
		} catch (err) {
			error = getErrorMessage(err, 'Failed to delete script');
		}
	}
</script>

{#if loading}
	<p class="text-secondary">Loading...</p>
{:else if error && !script}
	<div class="rounded-lg bg-error-bg px-4 py-2 text-sm text-error-text">{error}</div>
{:else if script}
	<div class="space-y-6">
		<!-- Top bar -->
		<div class="flex items-center justify-between gap-4">
			<div class="flex min-w-0 flex-1 items-center gap-3">
				<a href="/scripts" aria-label="Back to scripts" class="text-secondary transition-colors hover:text-medium">
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</a>
				{#if script.isSystem}
					<h2 class="min-w-0 flex-1 text-lg font-medium text-primary">{script.name}</h2>
				{:else}
					<input
						bind:value={name}
						class="min-w-0 flex-1 rounded-lg border border-border bg-surface-alt px-3 py-2 text-lg font-medium text-primary focus:border-indigo-400 focus:outline-none"
					/>
				{/if}
			</div>
			<div class="flex items-center gap-2">
				{#if !script.isSystem}
					<button
						onclick={openAddCharacter}
						class="btn-primary rounded-lg bg-indigo-500 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-400"
					>
						Add Characters
					</button>
				{/if}
				<a
					href="/games/new?script={script.id}"
					class="btn-secondary rounded-lg border border-border px-3 py-2 text-sm text-medium transition-colors hover:bg-hover"
				>
					Start Game
				</a>
				{#if script.isSystem}
					<button
						onclick={() => createFromEdition(script.edition)}
						class="btn-secondary rounded-lg border border-border px-3 py-2 text-sm text-medium transition-colors hover:bg-hover"
					>
						Duplicate
					</button>
				{:else}
					<button
						onclick={deleteScript}
						aria-label="Delete script"
						class="rounded p-2 text-muted transition-colors hover:bg-hover hover:text-red-500"
					>
						<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
						</svg>
					</button>
				{/if}
			</div>
		</div>

		{#if error}
			<div class="rounded-lg bg-error-bg px-4 py-2 text-sm text-error-text">{error}</div>
		{/if}

		<!-- Character list grouped by team -->
		<div class="space-y-6">
			{#each teamOrder as team}
				{@const chars = charactersByTeam[team]}
				{#if chars && chars.length > 0}
					<TeamSection {team} characters={chars} removable={!script.isSystem} onremove={removeCharacter} />
				{/if}
			{/each}
		</div>

		{#if script.characterIds.length === 0}
			<div class="card-slate rounded-lg border border-dashed border-border-strong p-8 text-center">
				<p class="text-secondary">No characters yet. Click "Add Characters" to get started.</p>
			</div>
		{/if}
	</div>

	{#if showAddCharacter}
		<CharacterPickerModal
			title="Add Characters"
			characters={allCharacters}
			selectedIds={scriptIdSet}
			onselect={addCharacter}
			ondeselect={removeCharacter}
			onclose={() => (showAddCharacter = false)}
		/>
	{/if}
{/if}
