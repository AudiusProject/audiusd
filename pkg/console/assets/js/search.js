function searchBar() {
    return {
        query: '',
        showSuggestions: false,
        searchType: '',
        suggestions: [],
        isLoading: false,
        
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
            this.$watch('showSuggestions', (value) => {
                console.log('showSuggestions changed:', value);
                if (value) {
                    console.log('Showing all suggestions');
                    this.fetchAllSuggestions();
                }
            });
        },

        async fetchAllSuggestions() {
            this.isLoading = true;
            try {
                // Simulate API delay
                await new Promise(resolve => setTimeout(resolve, 300));
                
                const allSuggestions = [
                    ...this.mockData.blocks,
                    ...this.mockData.accounts,
                    ...this.mockData.transactions,
                    ...this.mockData.content
                ];
                
                this.searchType = 'All';
                this.suggestions = this.groupSuggestionsByType(allSuggestions);
                this.showSuggestions = true;
            } catch (error) {
                console.error('Error fetching suggestions:', error);
            } finally {
                this.isLoading = false;
            }
        },

        async handleInput() {
            console.log('Input changed:', this.query);
            
            // Clear type and suggestions if query is empty
            if (!this.query.trim()) {
                await this.fetchAllSuggestions();
                return;
            }

            this.isLoading = true;
            try {
                // Simulate API delay
                await new Promise(resolve => setTimeout(resolve, 300));

                let results = [];
                
                // For 0x inputs, show filtered suggestions
                if (this.query.startsWith('0x')) {
                    console.log('Starts with 0x');
                    results = [
                        ...this.mockData.accounts,
                        ...this.mockData.transactions
                    ].filter(suggestion => 
                        suggestion.title.toLowerCase().includes(this.query.toLowerCase())
                    );
                    
                    // Set search type based on length and format
                    if (this.query.match(/^0x[a-fA-F0-9]{40}$/)) {
                        console.log('Full address match');
                        this.searchType = 'Account';
                        results = this.mockData.accounts;
                    } else if (this.query.match(/^0x[a-fA-F0-9]{64}$/)) {
                        console.log('Full transaction match');
                        this.searchType = 'Transaction';
                        results = this.mockData.transactions;
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
                    results = this.mockData.blocks;
                } else if (this.query.match(/^[a-zA-Z0-9_\- ]+$/)) {
                    console.log('Content match');
                    this.searchType = 'Content';
                    results = this.mockData.content;
                } else {
                    console.log('No match');
                    this.searchType = '';
                    this.suggestions = [];
                    this.showSuggestions = false;
                    return;
                }

                // Group suggestions by type
                this.suggestions = this.groupSuggestionsByType(results);
                this.showSuggestions = true;
            } catch (error) {
                console.error('Error handling input:', error);
            } finally {
                this.isLoading = false;
            }
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

        getNavigationPath(suggestion) {
            if (suggestion.isHeader) return '';
            
            switch(suggestion.type) {
                case 'block':
                    const blockNum = suggestion.title.match(/#(\d+)/)?.[1];
                    return blockNum ? `/block/${blockNum}` : '';
                case 'account':
                    return `/accounts/${suggestion.title}`;
                case 'transaction':
                    return `/transactions/${suggestion.title}`;
                case 'track':
                    return `/tracks/${suggestion.id}`;
                case 'username':
                    return `/users/${suggestion.title}`;
                case 'playlist':
                    return `/playlists/${suggestion.id}`;
                case 'album':
                    return `/albums/${suggestion.id}`;
                default:
                    return '';
            }
        },

        selectSuggestion(suggestion) {
            if (suggestion.isHeader) return;
            
            const path = this.getNavigationPath(suggestion);
            if (path) {
                this.$dispatch('navigate', { path });
            }
            
            this.query = suggestion.title;
            this.showSuggestions = false;
        }
    }
}
