module.exports = {
	disableEmoji: false,
	format: "{type}: {subject}",
	list: [
	  "fix",
	  "feat",
	  "refactor",
	  "test",
	  "style",
	  "chore",
	  "docs",
	  "perf",
	  "ci",
	],
	maxMessageLength: 64,
	minMessageLength: 3,
	questions: ["type", "subject"],
	scopes: [],
	types: {
	  chore: {
		description: "ドキュメントの生成やビルドプロセス、ライブラリなどの変更",
		value: "chore",
	  },
	  ci: {
		description: "CI用の設定やスクリプトに関する変更",
		value: "ci",
	  },
	  docs: {
		description: "ドキュメントのみの変更",
		value: "docs",
	  },
	  feat: {
		description: "新機能",
		value: "feat",
	  },
	  fix: {
		description: "不具合の修正",
		value: "fix",
	  },
	  perf: {
		description: "パフォーマンス改善を行うためのコードの変更",
		value: "perf",
	  },
	  refactor: {
		description: "バグ修正や機能の追加を行わないコードの変更",
		value: "refactor",
	  },
	  style: {
		description: "コードの処理に影響しない変更（スペースや書式設定など）",
		value: "style",
	  },
	  test: {
		description: "テストコードの変更",
		value: "test",
	  },
	},
  };