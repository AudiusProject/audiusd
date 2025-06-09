function searchBar() {
    return {
        query: '',
        showSuggestions: false,
        searchType: '',
        suggestions: [],
        
        mockData: {
            blocks: [
                { id: 1, title: 'Block #12345', subtitle: 'Added 2 hours ago', type: 'block' },
                { id: 2, title: 'Block #12344', subtitle: 'Added 3 hours ago', type: 'block' },
            ],
            accounts: [
                { id: 3, title: '0x1234567890123456789012345678901234567890', subtitle: 'Last active 5 min ago', type: 'account' },
                { id: 4, title: '0xabcdef1234567890abcdef1234567890abcdef12', subtitle: 'Last active 10 min ago', type: 'account' },
            ],
            transactions: [
                { id: 5, title: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890', subtitle: 'Confirmed 10 min ago', type: 'transaction' },
                { id: 6, title: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef', subtitle: 'Confirmed 15 min ago', type: 'transaction' },
            ],
            content: [
                { id: 7, title: 'Summer Vibes', subtitle: 'Track by Artist123', type: 'track' },
                { id: 8, title: 'Winter Dreams', subtitle: 'Track by Artist456', type: 'track' },
                { id: 9, title: 'Artist123', subtitle: 'Verified Artist', type: 'username' },
                { id: 10, title: 'Artist456', subtitle: 'Verified Artist', type: 'username' },
                { id: 11, title: 'Playlist: Summer Hits', subtitle: 'By Artist123', type: 'playlist' },
                { id: 12, title: 'Album: Winter Collection', subtitle: 'By Artist456', type: 'album' },
            ]
        },

        init() {
            console.log('Search component initialized');
            // Show all suggestions when focused
            this.$watch('showSuggestions', (value) => {
                console.log('showSuggestions changed:', value);
                if (value) {
                    console.log('Showing all suggestions');
                    this.showAllSuggestions();
                }
            });
        },

        showAllSuggestions() {
            console.log('Showing all suggestions');
            this.searchType = 'All';
            this.suggestions = [
                ...this.mockData.blocks,
                ...this.mockData.accounts,
                ...this.mockData.transactions,
                ...this.mockData.content
            ];
            console.log('All suggestions:', this.suggestions);
            this.suggestions = this.groupSuggestionsByType(this.suggestions);
            console.log('Grouped suggestions:', this.suggestions);
            this.showSuggestions = true;
        },

        handleInput() {
            console.log('Input changed:', this.query);
            // Clear type and suggestions if query is empty
            if (!this.query.trim()) {
                this.showAllSuggestions();
                return;
            }

            // For 0x inputs, show filtered suggestions
            if (this.query.startsWith('0x')) {
                console.log('Starts with 0x');
                // Show both accounts and transactions for any 0x input
                this.suggestions = [
                    ...this.mockData.accounts,
                    ...this.mockData.transactions
                ].filter(suggestion => 
                    suggestion.title.toLowerCase().includes(this.query.toLowerCase())
                );
                
                // Set search type based on length and format
                if (this.query.match(/^0x[a-fA-F0-9]{40}$/)) {
                    console.log('Full address match');
                    this.searchType = 'Account';
                    this.suggestions = this.mockData.accounts;
                } else if (this.query.match(/^0x[a-fA-F0-9]{64}$/)) {
                    console.log('Full transaction match');
                    this.searchType = 'Transaction';
                    this.suggestions = this.mockData.transactions;
                } else if (this.query.length <= 42) {
                    console.log('Partial address - showing filtered suggestions');
                    this.searchType = 'Account';
                } else {
                    console.log('Partial transaction - showing filtered suggestions');
                    this.searchType = 'Transaction';
                }
            } else if (this.query.match(/^[0-9]+$/)) {
                console.log('Block number match');
                this.searchType = 'Block';
                this.suggestions = this.mockData.blocks;
            } else if (this.query.match(/^[a-zA-Z0-9_\- ]+$/)) {
                console.log('Content match');
                this.searchType = 'Content';
                this.suggestions = this.mockData.content;
            } else {
                console.log('No match');
                this.searchType = '';
                this.suggestions = [];
                this.showSuggestions = false;
                return;
            }

            // Group suggestions by type
            this.suggestions = this.groupSuggestionsByType(this.suggestions);

            // Always show suggestions if we have them
            this.showSuggestions = true;
            console.log('Final suggestions:', this.suggestions);
        },

        groupSuggestionsByType(suggestions) {
            const grouped = {};
            suggestions.forEach(suggestion => {
                if (!grouped[suggestion.type]) {
                    grouped[suggestion.type] = [];
                }
                grouped[suggestion.type].push(suggestion);
            });

            // Convert grouped object to array with headers
            const result = [];
            Object.entries(grouped).forEach(([type, items]) => {
                if (items.length > 0) {
                    result.push({
                        id: `header-${type}`,
                        isHeader: true,
                        title: this.formatTypeHeader(type),
                        type: type
                    });
                    result.push(...items);
                }
            });
            return result;
        },

        formatTypeHeader(type) {
            const headers = {
                'track': 'Tracks',
                'username': 'Artists',
                'playlist': 'Playlists',
                'album': 'Albums',
                'block': 'Blocks',
                'account': 'Accounts',
                'transaction': 'Transactions'
            };
            return headers[type] || type.charAt(0).toUpperCase() + type.slice(1) + 's';
        },

        selectSuggestion(suggestion) {
            if (suggestion.isHeader) return;
            this.query = suggestion.title;
            // Re-run pattern matching for the selected suggestion
            if (suggestion.type === 'account') {
                this.searchType = 'Account';
                this.suggestions = this.mockData.accounts;
            } else if (suggestion.type === 'transaction') {
                this.searchType = 'Transaction';
                this.suggestions = this.mockData.transactions;
            } else if (suggestion.type === 'block') {
                this.searchType = 'Block';
                this.suggestions = this.mockData.blocks;
            } else {
                this.searchType = 'Content';
                this.suggestions = this.mockData.content;
            }
            // Keep suggestions visible after selection
            this.showSuggestions = true;
            // Here you would typically navigate to the appropriate page
            console.log('Selected:', suggestion);
        }
    }
}
