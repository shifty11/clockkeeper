<script lang="ts">
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { client } from '~/lib/api';
	import { getErrorMessage } from '~/lib/errors';
	import type { Game, Character, Script } from '~/lib/gen/clockkeeper/v1/clockkeeper_pb';
	import { Team } from '~/lib/gen/clockkeeper/v1/clockkeeper_pb';
	import CharacterCard from '~/lib/components/CharacterCard.svelte';
	import CharacterPickerModal from '~/lib/components/CharacterPickerModal.svelte';
	import DistributionBar from '~/lib/components/DistributionBar.svelte';
	import ReminderToken from '~/lib/components/ReminderToken.svelte';
	import SetupSidebar from '~/lib/components/SetupSidebar.svelte';
	import NightOrder from '~/lib/components/NightOrder.svelte';
	type GameTab = 'setup' | 'nightorder' | 'grimoire';
	const tabs: { id: GameTab; label: string }[] = [
		{ id: 'setup', label: 'Setup' },
		{ id: 'nightorder', label: 'Night Order' },
		{ id: 'grimoire', label: 'Grimoire' },
	];
	const validTabs = new Set<GameTab>(['setup', 'nightorder', 'grimoire']);
	const initialTab = page.url.searchParams.get('tab') as GameTab | null;
	let activeTab = $state<GameTab>(initialTab && validTabs.has(initialTab) ? initialTab : 'setup');

	function setTab(tab: GameTab) {
		activeTab = tab;
		const url = new URL(window.location.href);
		url.searchParams.set('tab', tab);
		goto(url.toString(), { replaceState: true, noScroll: true });
	}

	let game = $state<Game | undefined>();
	let script = $state<Script | undefined>();
	let loading = $state(true);
	let error = $state('');
	let randomizing = $state(false);

	// Picker state.
	let showCharacterPicker = $state(false);
	let pickerTeam = $state<Team | undefined>();
	let allCharacters = $state<Character[]>([]);

	const teamOrder = [Team.TOWNSFOLK, Team.OUTSIDER, Team.MINION, Team.DEMON] as const;

	const teamLabels: Record<number, string> = {
		[Team.TOWNSFOLK]: 'Townsfolk',
		[Team.OUTSIDER]: 'Outsiders',
		[Team.MINION]: 'Minions',
		[Team.DEMON]: 'Demons',
		[Team.TRAVELLER]: 'Travellers',
		[Team.FABLED]: 'Fabled',
		[Team.LORIC]: 'Lorics'
	};

	const teamHeaderColors: Record<number, string> = {
		[Team.TOWNSFOLK]: 'text-blue-600 dark:text-blue-400',
		[Team.OUTSIDER]: 'text-cyan-600 dark:text-cyan-400',
		[Team.MINION]: 'text-orange-600 dark:text-orange-400',
		[Team.DEMON]: 'text-red-600 dark:text-red-400',
		[Team.TRAVELLER]: 'bg-gradient-to-r from-blue-500 to-red-500 bg-clip-text text-transparent',
		[Team.FABLED]: 'text-yellow-500 dark:text-yellow-400',
		[Team.LORIC]: 'text-green-600 dark:text-green-400'
	};

	const addCardColors: Record<number, string> = {
		[Team.TOWNSFOLK]: 'border-blue-300 text-blue-400 hover:bg-blue-50 dark:border-blue-700 dark:text-blue-500 dark:hover:bg-blue-950/30',
		[Team.OUTSIDER]: 'border-cyan-300 text-cyan-400 hover:bg-cyan-50 dark:border-cyan-700 dark:text-cyan-500 dark:hover:bg-cyan-950/30',
		[Team.MINION]: 'border-orange-300 text-orange-400 hover:bg-orange-50 dark:border-orange-700 dark:text-orange-500 dark:hover:bg-orange-950/30',
		[Team.DEMON]: 'border-red-300 text-red-400 hover:bg-red-50 dark:border-red-700 dark:text-red-500 dark:hover:bg-red-950/30',
		[Team.TRAVELLER]: 'card-traveller-add text-purple-400 dark:text-purple-500',
		[Team.FABLED]: 'border-yellow-300 text-yellow-500 hover:bg-yellow-50 dark:border-yellow-700 dark:text-yellow-500 dark:hover:bg-yellow-950/30',
		[Team.LORIC]: 'border-green-300 text-green-400 hover:bg-green-50 dark:border-green-700 dark:text-green-500 dark:hover:bg-green-950/30'
	};

	// Characters grouped by team — includes both script and extra characters.
	const charactersByTeam = $derived.by(() => {
		const grouped: Record<number, Character[]> = {};
		const skip = new Set([Team.TRAVELLER, Team.FABLED, Team.LORIC]);
		for (const char of script?.characters ?? []) {
			if (skip.has(char.team)) continue;
			if (!grouped[char.team]) grouped[char.team] = [];
			grouped[char.team].push(char);
		}
		for (const char of game?.extraCharacterDetails ?? []) {
			if (skip.has(char.team)) continue;
			if (!grouped[char.team]) grouped[char.team] = [];
			grouped[char.team].push(char);
		}
		return grouped;
	});

	// Selected = script roles + extra characters (both show as "selected" in the grid).
	const selectedRoleIdSet = $derived(
		new Set([...(game?.selectedRoleIds ?? []), ...(game?.extraCharacterIds ?? [])])
	);

	// Track which IDs belong to the script vs extra (for toggle behavior).
	const scriptCharIdSet = $derived(new Set(script?.characters?.map((c) => c.id) ?? []));
	const extraCharIdSet = $derived(new Set(game?.extraCharacterIds ?? []));

	const selectedTravellerIdSet = $derived(
		new Set(game?.selectedTravellerIds ?? [])
	);

	const fabledCharacters = $derived(
		(game?.extraCharacterDetails ?? []).filter((c) => c.team === Team.FABLED)
	);
	const loricCharacters = $derived(
		(game?.extraCharacterDetails ?? []).filter((c) => c.team === Team.LORIC)
	);

	const optionalTeams = $derived([
		{ team: Team.TRAVELLER, label: 'Travellers', singular: 'Traveller', chars: game?.selectedTravellerCharacters ?? [], remove: removeTraveller },
		{ team: Team.FABLED, label: 'Fabled', singular: 'Fabled', chars: fabledCharacters, remove: removeExtraChar },
		{ team: Team.LORIC, label: 'Lorics', singular: 'Loric', chars: loricCharacters, remove: removeExtraChar },
	]);
	const emptyOptionals = $derived(optionalTeams.filter((o) => o.chars.length === 0));

	// Combined selectedIds for the character picker modal.
	const pickerSelectedIds = $derived(
		new Set([...(game?.selectedRoleIds ?? []), ...(game?.extraCharacterIds ?? []), ...(script?.characterIds ?? []), ...(game?.selectedTravellerIds ?? [])])
	);

	const currentDist = $derived.by(() => {
		if (!game) return { townsfolk: 0, outsiders: 0, minions: 0, demons: 0 };
		const d = { townsfolk: 0, outsiders: 0, minions: 0, demons: 0 };
		// Count from all characters (script + extra) that are selected.
		for (const [, chars] of Object.entries(charactersByTeam)) {
			for (const c of chars) {
				if (!selectedRoleIdSet.has(c.id)) continue;
				if (c.team === Team.TOWNSFOLK) d.townsfolk++;
				else if (c.team === Team.OUTSIDER) d.outsiders++;
				else if (c.team === Team.MINION) d.minions++;
				else if (c.team === Team.DEMON) d.demons++;
			}
		}
		return d;
	});

	const characterById = $derived.by(() => {
		const map = new Map<string, Character>();
		for (const char of script?.characters ?? []) {
			map.set(char.id, char);
		}
		for (const char of game?.selectedTravellerCharacters ?? []) {
			map.set(char.id, char);
		}
		for (const char of game?.extraCharacterDetails ?? []) {
			map.set(char.id, char);
		}
		return map;
	});

	async function loadGame(gameId: bigint) {
		loading = true;
		error = '';
		game = undefined;
		script = undefined;
		try {
			const resp = await client.getGame({ id: gameId });
			game = resp.game;
			if (game) {
				const scriptResp = await client.getScript({ id: game.scriptId });
				script = scriptResp.script;
			}
		} catch (err) {
			error = getErrorMessage(err, 'Failed to load game');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		const id = page.params.id;
		untrack(() => {
			let gameId: bigint;
			try {
				gameId = BigInt(id);
			} catch {
				error = 'Invalid game ID';
				loading = false;
				return;
			}
			loadGame(gameId);
		});
	});

	async function randomize() {
		if (!game) return;
		randomizing = true;
		error = '';
		try {
			const resp = await client.randomizeRoles({ gameId: game.id });
			game = resp.game;
		} catch (err) {
			error = getErrorMessage(err, 'Failed to randomize roles');
		} finally {
			randomizing = false;
		}
	}

	async function toggleRole(id: string) {
		if (!game) return;
		error = '';

		// If it's an extra character, toggle via the extra characters API.
		if (extraCharIdSet.has(id)) {
			const newIds = (game.extraCharacterIds ?? []).filter((eid) => eid !== id);
			try {
				const resp = await client.updateGameExtraCharacters({
					gameId: game.id,
					extraCharacterIds: newIds
				});
				game = resp.game;
			} catch (err) {
				error = getErrorMessage(err, 'Failed to update roles');
			}
			return;
		}

		// Otherwise toggle via the normal roles API.
		const newIds = selectedRoleIdSet.has(id)
			? game.selectedRoleIds.filter((rid) => rid !== id)
			: [...game.selectedRoleIds, id];
		try {
			const resp = await client.updateGameRoles({
				gameId: game.id,
				selectedRoleIds: newIds
			});
			game = resp.game;
		} catch (err) {
			error = getErrorMessage(err, 'Failed to update roles');
		}
	}

	async function openCharacterPicker(forTeam?: Team) {
		error = '';
		if (allCharacters.length === 0) {
			try {
				const resp = await client.listCharacters({});
				allCharacters = resp.characters;
			} catch (err) {
				error = getErrorMessage(err, 'Failed to load characters');
				return;
			}
		}
		pickerTeam = forTeam;
		showCharacterPicker = true;
	}

	async function addExtraChar(char: Character) {
		if (!game) return;
		error = '';
		const newIds = [...(game.extraCharacterIds ?? []), char.id];
		try {
			const resp = await client.updateGameExtraCharacters({
				gameId: game.id,
				extraCharacterIds: newIds
			});
			game = resp.game;
		} catch (err) {
			error = getErrorMessage(err, 'Failed to add character');
		}
	}

	async function removeExtraChar(charId: string) {
		if (!game) return;
		error = '';
		const newIds = (game.extraCharacterIds ?? []).filter((eid) => eid !== charId);
		try {
			const resp = await client.updateGameExtraCharacters({
				gameId: game.id,
				extraCharacterIds: newIds
			});
			game = resp.game;
		} catch (err) {
			error = getErrorMessage(err, 'Failed to remove character');
		}
	}

	function handlePickerSelect(char: Character) {
		if (char.team === Team.TRAVELLER) {
			addTraveller(char);
		} else if (scriptCharIdSet.has(char.id)) {
			toggleRole(char.id);
		} else {
			addExtraChar(char);
		}
	}

	function handlePickerDeselect(charId: string) {
		if (selectedTravellerIdSet.has(charId)) {
			removeTraveller(charId);
		} else if (scriptCharIdSet.has(charId)) {
			toggleRole(charId);
		} else {
			removeExtraChar(charId);
		}
	}

	async function addTraveller(char: Character) {
		if (!game) return;
		error = '';
		const newIds = [...game.selectedTravellerIds, char.id];
		try {
			const resp = await client.updateGameTravellers({
				gameId: game.id,
				selectedTravellerIds: newIds
			});
			game = resp.game;
		} catch (err) {
			error = getErrorMessage(err, 'Failed to add traveller');
		}
	}

	async function removeTraveller(charId: string) {
		if (!game) return;
		error = '';
		const newIds = game.selectedTravellerIds.filter((tid) => tid !== charId);
		try {
			const resp = await client.updateGameTravellers({
				gameId: game.id,
				selectedTravellerIds: newIds
			});
			game = resp.game;
		} catch (err) {
			error = getErrorMessage(err, 'Failed to remove traveller');
		}
	}
</script>

{#if loading}
	<p class="text-secondary">Loading...</p>
{:else if error && !game}
	<div class="rounded-lg bg-error-bg border border-error-border px-4 py-2 text-sm text-error-text">{error}</div>
{:else if game}
	<div class="space-y-6 pb-16 2xl:pb-0">
		<!-- Header -->
		<div class="no-print flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-bold text-primary">Game</h1>
				<p class="mt-1 text-secondary">
					{game.playerCount} players{#if game.travellerCount > 0}
						+ {game.travellerCount} {game.travellerCount === 1 ? 'traveller' : 'travellers'}
						= {game.playerCount + game.travellerCount} total{/if}
				</p>
			</div>
		</div>

		<!-- Tab bar -->
		<div class="no-print flex gap-1 rounded-lg bg-element p-1">
			{#each tabs as t}
				<button
					onclick={() => setTab(t.id)}
					class="rounded-md px-4 py-2 text-sm font-medium transition-colors {activeTab === t.id
						? 'bg-surface text-primary shadow-sm'
						: 'text-secondary hover:text-medium'}"
				>
					{t.label}
				</button>
			{/each}
		</div>

		{#if error}
			<div class="rounded-lg bg-error-bg border border-error-border px-4 py-2 text-sm text-error-text">{error}</div>
		{/if}

		<!-- ===== SETUP TAB ===== -->
		{#if activeTab === 'setup'}
			<div class="space-y-6">
				<div class="flex items-center justify-end">
					<button
						onclick={randomize}
						disabled={randomizing}
						class="btn-primary rounded-lg bg-indigo-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-400 disabled:opacity-50"
					>
						{randomizing ? 'Randomizing...' : 'Randomize Roles'}
					</button>
				</div>

				<!-- Distribution -->
				<div class="rounded-lg border border-border bg-surface p-4">
					<DistributionBar current={currentDist} expected={game.distribution} travellers={game.selectedTravellerCharacters.length} />
				</div>

				<!-- Characters — click to toggle selection (script + extra merged) -->
				{#if script}
					<div class="space-y-6">
						{#each teamOrder as team}
							{@const chars = charactersByTeam[team]}
							{#if chars && chars.length > 0}
								<div>
									<h3 class="mb-3 text-sm font-semibold uppercase tracking-wide {teamHeaderColors[team] ?? 'text-secondary'}">
										{teamLabels[team] ?? team} ({chars.filter((c) => selectedRoleIdSet.has(c.id)).length}/{chars.length})
									</h3>
									<div class="grid gap-2 sm:grid-cols-2">
										{#each chars as char (char.id)}
											<button
												onclick={() => toggleRole(char.id)}
												class="h-full w-full text-left transition-transform active:scale-[0.98]"
											>
												<CharacterCard
													character={char}
													selected={selectedRoleIdSet.has(char.id)}
												/>
											</button>
										{/each}
										<!-- Add button -->
										<button
											onclick={() => openCharacterPicker(team)}
											class="flex h-full min-h-[4rem] items-center justify-center gap-2 rounded-lg border-2 border-dashed transition-colors {addCardColors[team] ?? 'border-border text-muted hover:bg-hover'}"
										>
											<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
											</svg>
											<span class="text-sm font-medium">Add {teamLabels[team]?.slice(0, -1) ?? 'Character'}</span>
										</button>
									</div>
								</div>
							{/if}
						{/each}
					</div>
				{/if}

				<!-- Optional teams: Travellers, Fabled, Lorics -->
				{#each optionalTeams as opt}
					{#if opt.chars.length > 0}
						<section class="space-y-3">
							<h2 class="text-sm font-semibold uppercase tracking-wide {teamHeaderColors[opt.team]}">
								{opt.label} ({opt.chars.length})
							</h2>
							<div class="grid gap-2 sm:grid-cols-2">
								{#each opt.chars as char (char.id)}
									<CharacterCard
										character={char}
										removable
										onremove={() => opt.remove(char.id)}
									/>
								{/each}
								<button
									onclick={() => openCharacterPicker(opt.team)}
									class="flex min-h-[4rem] items-center justify-center gap-2 rounded-lg border-2 border-dashed transition-colors {addCardColors[opt.team]}"
								>
									<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
									</svg>
									<span class="text-sm font-medium">Add {opt.singular}</span>
								</button>
							</div>
						</section>
					{/if}
				{/each}

				<!-- Compact row for empty teams -->
				{#if emptyOptionals.length > 0}
					<div class="grid gap-2" style="grid-template-columns: repeat({emptyOptionals.length}, 1fr)">
						{#each emptyOptionals as opt}
							<button
								onclick={() => openCharacterPicker(opt.team)}
								class="flex min-h-[4rem] items-center justify-center gap-2 rounded-lg border-2 border-dashed transition-colors {addCardColors[opt.team] ?? 'border-border text-muted hover:bg-hover'}"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
								</svg>
								<span class="text-sm font-medium">{opt.label}</span>
							</button>
						{/each}
					</div>
				{/if}

				<!-- Reminder tokens -->
				{#if game.reminderTokens.length > 0}
					<section>
						<h2 class="mb-3 text-lg font-semibold text-medium">Reminder Tokens</h2>
						<div class="flex flex-wrap gap-4">
							{#each game.reminderTokens as token}
								{@const char = characterById.get(token.characterId)}
								<ReminderToken
									characterId={token.characterId}
									characterName={token.characterName}
									text={token.text}
									edition={char?.edition ?? ''}
									team={char?.team ?? Team.UNSPECIFIED}
								/>
							{/each}
						</div>
					</section>
				{/if}
			</div>

		<!-- ===== NIGHT ORDER TAB ===== -->
		{:else if activeTab === 'nightorder'}
			<NightOrder {game} scriptCharacters={script?.characters ?? []} />

		<!-- ===== GRIMOIRE TAB ===== -->
		{:else if activeTab === 'grimoire'}
			<div class="flex flex-col items-center justify-center py-16 text-center">
				<svg class="mb-4 h-16 w-16 text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
				</svg>
				<h2 class="text-lg font-semibold text-primary">Grimoire</h2>
				<p class="mt-2 max-w-md text-sm text-secondary">
					The Grimoire will let you track tokens, player states, and game progression. Coming soon.
				</p>
			</div>
		{/if}
	</div>

	<!-- Character picker modal -->
	{#if showCharacterPicker}
		<CharacterPickerModal
			title={pickerTeam ? `Add ${teamLabels[pickerTeam] ?? 'Character'}` : 'Add Character'}
			characters={allCharacters}
			selectedIds={pickerSelectedIds}
			team={pickerTeam}
			onselect={handlePickerSelect}
			ondeselect={handlePickerDeselect}
			onclose={() => (showCharacterPicker = false)}
		/>
	{/if}

	<!-- Setup sidebar (always visible on setup tab) -->
	{#if activeTab === 'setup'}
		<SetupSidebar gameId={game.id} selectedIds={[...(game.selectedRoleIds ?? []), ...(game.extraCharacterIds ?? [])]} />
	{/if}
{/if}
