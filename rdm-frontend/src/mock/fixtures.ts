import type { Category, Suggestion, User } from '../types'

export const mockUser: User = {
  id: '208412739182465024',
  displayName: 'Mara Quill',
  username: 'maraquill',
  avatarUrl: null,
  roles: ['voter'],
}

export const mockCategories: Category[] = [
  {
    id: 'goty',
    name: 'Game of the Year',
    description: 'The single finest game of the season, judged across every discipline.',
    criteria: 'Any title released between 1 January and 30 November. Re-releases are not eligible.',
    maxSuggestions: 3,
  },
  {
    id: 'direction',
    name: 'Outstanding Direction',
    description: 'For the creative vision that held an entire production together.',
    criteria: 'Name the game; the jury identifies the credited director.',
  },
  {
    id: 'narrative',
    name: 'Achievement in Narrative',
    description: 'Writing, structure and character work that lingered after the credits.',
  },
  {
    id: 'art-direction',
    name: 'Distinguished Art Direction',
    description: 'Visual identity, world-building and the discipline behind a coherent look.',
  },
  {
    id: 'score',
    name: 'Original Score and Sound',
    description: 'Composition, sound design and the craft of making a world audible.',
  },
  {
    id: 'performance',
    name: 'Best Performance',
    description: 'A single leading or supporting performance in a released title.',
    criteria: 'Name the performer and the role, e.g. "A. Nolan as Vesper".',
  },
  {
    id: 'independent',
    name: 'Independent Game of the Year',
    description: 'The finest work from a studio operating without a publisher.',
  },
  {
    id: 'ongoing',
    name: 'Best Ongoing Game',
    description: 'A live title that grew meaningfully over the past twelve months.',
  },
  {
    id: 'debut',
    name: 'Debut Studio of the Year',
    description: 'A first commercial release that announced a studio worth watching.',
    locked: true,
  },
  {
    id: 'community',
    name: 'Community Choice',
    description: 'The people\u2019s pick, tallied from the open floor.',
    maxSuggestions: 1,
  },
]

export const mockSuggestions: Suggestion[] = [
  { id: 'sg_1', categoryId: 'goty', text: 'Hollow Fields', updatedAt: '2026-09-02T18:11:00Z' },
  { id: 'sg_2', categoryId: 'goty', text: 'Sable Harbour', updatedAt: '2026-09-02T18:11:00Z' },
  { id: 'sg_3', categoryId: 'narrative', text: 'The Cartographer\u2019s Widow', updatedAt: '2026-09-04T09:40:00Z' },
  { id: 'sg_4', categoryId: 'debut', text: 'Nine Tenths Studio', updatedAt: '2026-08-21T12:02:00Z' },
]
