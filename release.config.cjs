// semantic-release configuration
// Docs: https://semantic-release.gitbook.io/semantic-release/
//

// Commit & PR title format:
//   feat(scope): [YLP-XXXX] description    ← Jira ticket required for feat/fix/perf/revert/refactor
//   fix(scope): [YLP-XXXX] description
//   chore(deps): bump go-common to 1.9.0   ← Jira ticket optional for chore/ci/docs/test/style
//

/** @type {import('semantic-release').GlobalConfig} */
module.exports = {
  branches: ['dblp'],
  tagFormat: 'v${version}',
  plugins: [
    [
      '@semantic-release/commit-analyzer',
      {
        preset: 'conventionalcommits',
        releaseRules: [
          { type: 'feat',     release: 'minor' },
          { type: 'fix',      release: 'patch' },
          { type: 'perf',     release: 'patch' },
          { type: 'revert',   release: 'patch' },
          { type: 'refactor', release: false   },
          { type: 'docs',     release: false   },
          { type: 'style',    release: false   },
          { type: 'test',     release: false   },
          { type: 'chore',    release: false   },
          { type: 'ci',       release: false   },
          { breaking: true,   release: 'major' },
        ],
      },
    ],
    [
      '@semantic-release/release-notes-generator',
      {
        preset: 'conventionalcommits',
        presetConfig: {
          types: [
            { type: 'feat',     section: '✨ Features',       hidden: false },
            { type: 'fix',      section: '🐛 Bug Fixes',      hidden: false },
            { type: 'perf',     section: '⚡ Performance',    hidden: false },
            { type: 'revert',   section: '⏪ Reverts',        hidden: false },
            { type: 'refactor', section: '♻️  Refactoring',   hidden: false },
            { type: 'docs',     section: '📚 Documentation',  hidden: false },
            { type: 'test',     section: '✅ Tests',           hidden: false },
            { type: 'chore',    section: '🔧 Chores',         hidden: true  },
            { type: 'ci',       section: '👷 CI/CD',          hidden: true  },
          ],
        },
      },
    ],
    '@semantic-release/github',
  ],
};

