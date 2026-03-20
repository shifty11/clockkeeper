<script lang="ts">
	import type { Character } from '~/lib/gen/clockkeeper/v1/clockkeeper_pb';
	import { Team } from '~/lib/gen/clockkeeper/v1/clockkeeper_pb';
	import CharacterCard from './CharacterCard.svelte';

	let {
		team,
		characters,
		removable = false,
		onremove
	}: {
		team: Team;
		characters: Character[];
		removable?: boolean;
		onremove?: (id: string) => void;
	} = $props();

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
</script>

<div>
	<h3 class="mb-3 text-sm font-semibold uppercase tracking-wide {teamHeaderColors[team] ?? 'text-secondary'}">
		{teamLabels[team] ?? team} ({characters.length})
	</h3>
	<div class="grid gap-2 sm:grid-cols-2">
		{#each characters as char (char.id)}
			<CharacterCard
				character={char}
				{removable}
				onremove={onremove ? () => onremove(char.id) : undefined}
			/>
		{/each}
	</div>
</div>
